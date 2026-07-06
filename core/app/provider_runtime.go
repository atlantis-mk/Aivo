package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"aivo/core/domain"
)

const (
	providerErrorAuth         = "auth"
	providerErrorRateLimit    = "rate_limit"
	providerErrorUnavailable  = "unavailable"
	providerErrorBadRequest   = "bad_request"
	providerErrorNetwork      = "network"
	providerErrorTimeout      = "timeout"
	providerErrorContext      = "context"
	providerErrorUnsupported  = "unsupported"
	providerErrorEmpty        = "empty_response"
	providerErrorUnknown      = "unknown"
	providerHealthReady       = "ready"
	providerHealthDegraded    = "degraded"
	providerHealthUnavailable = "unavailable"
)

type ProviderRequestError struct {
	Class      string
	StatusCode int
	Message    string
	Err        error
}

func (e *ProviderRequestError) Error() string {
	if e == nil {
		return ""
	}
	msg := strings.TrimSpace(e.Message)
	if msg == "" && e.Err != nil {
		msg = e.Err.Error()
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("provider %s error (%d): %s", e.Class, e.StatusCode, msg)
	}
	return fmt.Sprintf("provider %s error: %s", e.Class, msg)
}

func (e *ProviderRequestError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func providerHTTPError(statusCode int, status string, body string) error {
	return &ProviderRequestError{
		Class:      classifyHTTPStatus(statusCode),
		StatusCode: statusCode,
		Message:    strings.TrimSpace(strings.TrimSpace(status) + ": " + strings.TrimSpace(body)),
	}
}

func providerResponseError(message string) error {
	return &ProviderRequestError{Class: providerErrorEmpty, Message: message}
}

func classifyProviderError(err error) ProviderRequestError {
	if err == nil {
		return ProviderRequestError{}
	}
	var providerErr *ProviderRequestError
	if errors.As(err, &providerErr) && providerErr != nil {
		return *providerErr
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		class := providerErrorContext
		if errors.Is(err, context.DeadlineExceeded) {
			class = providerErrorTimeout
		}
		return ProviderRequestError{Class: class, Message: err.Error(), Err: err}
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		class := providerErrorNetwork
		if netErr.Timeout() {
			class = providerErrorTimeout
		}
		return ProviderRequestError{Class: class, Message: err.Error(), Err: err}
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "credentials") || strings.Contains(msg, "unauthorized") || strings.Contains(msg, "forbidden"):
		return ProviderRequestError{Class: providerErrorAuth, Message: err.Error(), Err: err}
	case strings.Contains(msg, "unsupported provider transport") || strings.Contains(msg, "capability unsupported"):
		return ProviderRequestError{Class: providerErrorUnsupported, Message: err.Error(), Err: err}
	case strings.Contains(msg, "did not include text"):
		return ProviderRequestError{Class: providerErrorEmpty, Message: err.Error(), Err: err}
	default:
		return ProviderRequestError{Class: providerErrorUnknown, Message: err.Error(), Err: err}
	}
}

func classifyHTTPStatus(statusCode int) string {
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return providerErrorAuth
	case statusCode == http.StatusTooManyRequests:
		return providerErrorRateLimit
	case statusCode == http.StatusRequestTimeout || statusCode == http.StatusGatewayTimeout:
		return providerErrorTimeout
	case statusCode >= 500:
		return providerErrorUnavailable
	case statusCode >= 400:
		return providerErrorBadRequest
	default:
		return providerErrorUnknown
	}
}

func providerErrorRetryable(err error) bool {
	classified := classifyProviderError(err)
	switch classified.Class {
	case providerErrorRateLimit, providerErrorUnavailable, providerErrorTimeout, providerErrorNetwork:
		return true
	default:
		return false
	}
}

func providerHealthStatus(class string, failureCount int) string {
	if failureCount <= 0 {
		return providerHealthReady
	}
	switch class {
	case providerErrorAuth, providerErrorBadRequest, providerErrorUnsupported:
		return providerHealthUnavailable
	default:
		return providerHealthDegraded
	}
}

type providerCallFunc func() (domain.ChatResponse, error)

type providerRateLimiter struct {
	mu       sync.Mutex
	cooldown map[string]time.Time
}

func newProviderRateLimiter() *providerRateLimiter {
	return &providerRateLimiter{cooldown: map[string]time.Time{}}
}

func (l *providerRateLimiter) Before(providerID string, now time.Time) error {
	if l == nil || providerID == "" {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	until := l.cooldown[providerID]
	if until.IsZero() || !until.After(now) {
		delete(l.cooldown, providerID)
		return nil
	}
	return &ProviderRequestError{
		Class:   providerErrorRateLimit,
		Message: fmt.Sprintf("provider %s is rate limited until %s", providerID, until.Format(time.RFC3339)),
	}
}

func (l *providerRateLimiter) After(providerID string, err error, now time.Time, cooldown time.Duration) {
	if l == nil || providerID == "" {
		return
	}
	classified := classifyProviderError(err)
	l.mu.Lock()
	defer l.mu.Unlock()
	if classified.Class == providerErrorRateLimit {
		l.cooldown[providerID] = now.Add(cooldown)
		return
	}
	if err == nil {
		delete(l.cooldown, providerID)
	}
}

func (s *Service) callProviderWithRuntime(ctx context.Context, route ResolvedModelRoute, req modelRequirement, policy domain.ProviderRuntimePolicy, fallbackIndex int, call providerCallFunc) (domain.ChatResponse, error) {
	start := time.Now()
	var resp domain.ChatResponse
	var err error
	policy = normalizeProviderRuntimePolicy(policy)
	attempts := 1 + policy.MaxRetries
	if req.Streaming {
		attempts = 1
	}
	for attempt := 1; attempt <= attempts; attempt++ {
		if s.rateLimiter != nil {
			if limitErr := s.rateLimiter.Before(route.Provider.ID, s.now()); limitErr != nil {
				err = limitErr
				s.recordProviderCallEvent(ctx, route, req, fallbackIndex, attempt, time.Since(start), domain.ChatResponse{}, err)
				break
			}
		}
		resp, err = call()
		if s.rateLimiter != nil {
			s.rateLimiter.After(route.Provider.ID, err, s.now(), time.Duration(policy.RateLimitCooldownSeconds)*time.Second)
		}
		s.recordProviderCallEvent(ctx, route, req, fallbackIndex, attempt, time.Since(start), resp, err)
		if err == nil {
			s.recordProviderHealth(ctx, route.Provider.ID, nil, time.Since(start))
			return resp, nil
		}
		if attempt == attempts || !providerErrorRetryable(err) || ctx.Err() != nil {
			break
		}
		timer := time.NewTimer(retryDelay(policy, attempt))
		select {
		case <-ctx.Done():
			timer.Stop()
			err = ctx.Err()
		case <-timer.C:
		}
		if ctx.Err() != nil {
			break
		}
	}
	s.recordProviderHealth(ctx, route.Provider.ID, err, time.Since(start))
	return domain.ChatResponse{}, err
}

func (s *Service) recordProviderCallEvent(ctx context.Context, route ResolvedModelRoute, req modelRequirement, fallbackIndex int, attempt int, latency time.Duration, response domain.ChatResponse, err error) {
	if s == nil || s.store == nil || strings.TrimSpace(route.Provider.ID) == "" {
		return
	}
	event := domain.ProviderCallEvent{
		ProviderID:    route.Provider.ID,
		ModelID:       route.Model.ModelID,
		Transport:     string(route.Transport),
		Status:        "success",
		LatencyMs:     latency.Milliseconds(),
		Attempt:       attempt,
		FallbackIndex: fallbackIndex,
		Streaming:     req.Streaming,
		ToolCallCount: req.ToolCallCount,
		CreatedAt:     domain.NowString(s.now()),
	}
	if err == nil {
		if response.Usage != nil {
			event.InputTokens = response.Usage.InputTokens
			event.OutputTokens = response.Usage.OutputTokens
			event.TotalTokens = response.Usage.TotalTokens
			event.Estimated = response.Usage.Estimated
		}
		if event.TotalTokens == 0 && (event.InputTokens > 0 || event.OutputTokens > 0) {
			event.TotalTokens = event.InputTokens + event.OutputTokens
		}
		if event.InputTokens == 0 && event.OutputTokens == 0 && event.TotalTokens == 0 {
			event.InputTokens = estimateRouteInputTokens(req)
			event.OutputTokens = estimateTokens(response.Text)
			event.TotalTokens = event.InputTokens + event.OutputTokens
			event.Estimated = true
		}
		event.CostMicros = estimateRouteCostMicros(route, event.InputTokens, event.OutputTokens)
	}
	if err != nil {
		classified := classifyProviderError(err)
		event.Status = "failed"
		event.ErrorClass = classified.Class
		event.ErrorMessage = strings.TrimSpace(classified.Message)
		if event.ErrorMessage == "" {
			event.ErrorMessage = err.Error()
		}
		event.HTTPStatus = classified.StatusCode
	}
	_ = s.store.SaveProviderCallEvent(ctx, event)
}

func (s *Service) recordProviderHealth(ctx context.Context, providerID string, err error, latency time.Duration) {
	if s == nil || s.store == nil || strings.TrimSpace(providerID) == "" {
		return
	}
	now := domain.NowString(s.now())
	health := domain.ProviderHealth{
		ProviderID:    providerID,
		Status:        providerHealthReady,
		LastSuccessAt: now,
		LastLatencyMs: latency.Milliseconds(),
		UpdatedAt:     now,
	}
	if previous, loadErr := s.store.LoadProviderHealth(ctx, providerID); loadErr == nil && previous != nil {
		health = *previous
		health.ProviderID = providerID
		health.LastLatencyMs = latency.Milliseconds()
		health.UpdatedAt = now
	}
	if err == nil {
		health.Status = providerHealthReady
		health.LastSuccessAt = now
		health.LastErrorClass = ""
		health.LastErrorMessage = ""
		health.LastHTTPStatus = 0
		health.FailureCount = 0
		_ = s.store.SaveProviderHealth(ctx, health)
		return
	}
	classified := classifyProviderError(err)
	health.LastFailureAt = now
	health.LastErrorClass = classified.Class
	health.LastErrorMessage = strings.TrimSpace(classified.Message)
	if health.LastErrorMessage == "" {
		health.LastErrorMessage = err.Error()
	}
	health.LastHTTPStatus = classified.StatusCode
	health.FailureCount++
	health.Status = providerHealthStatus(classified.Class, health.FailureCount)
	_ = s.store.SaveProviderHealth(ctx, health)
}
