package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"aivo/core/domain"
)

func (s *Service) StartProviderAuth(ctx context.Context, input domain.ProviderAuthStartInput) (domain.ProviderAuthStartResult, error) {
	input.ProviderID = normalizeProviderID(input.ProviderID)
	if input.ProviderID == "" {
		return domain.ProviderAuthStartResult{}, errors.New("provider is required")
	}
	driver, ok := s.authFlows.driver(input.ProviderID)
	if !ok {
		return domain.ProviderAuthStartResult{}, fmt.Errorf("provider %q does not support interactive auth", input.ProviderID)
	}
	return driver.Start(ctx, input)
}

func (d *openAIAuthDriver) ProviderID() string {
	return "openai"
}

func (d *openAIAuthDriver) SupportedMethods() []string {
	return []string{"oauth", "oauth-browser", "browser", "oauth-headless", "headless"}
}

func (d *openAIAuthDriver) Start(ctx context.Context, input domain.ProviderAuthStartInput) (domain.ProviderAuthStartResult, error) {
	method := strings.TrimSpace(input.Method)
	switch method {
	case "oauth", "oauth-browser", "browser":
		return d.manager.startOpenAIBrowser(ctx)
	case "oauth-headless", "headless":
		return d.manager.startOpenAIHeadless(ctx)
	default:
		return domain.ProviderAuthStartResult{}, fmt.Errorf("unsupported OpenAI auth method: %s", method)
	}
}

func (s *Service) GetProviderAuthStatus(ctx context.Context, providerID string) (domain.ProviderAuthStatus, error) {
	return s.authFlows.status(providerID), nil
}

func (s *Service) CancelProviderAuth(ctx context.Context, providerID string) (domain.ProviderAuthStatus, error) {
	return s.authFlows.cancel(providerID), nil
}

func (m *ProviderAuthManager) startOpenAIBrowser(ctx context.Context) (domain.ProviderAuthStartResult, error) {
	if err := m.ensureOAuthServer(); err != nil {
		return domain.ProviderAuthStartResult{}, err
	}
	verifier, challenge, err := generatePKCE()
	if err != nil {
		return domain.ProviderAuthStartResult{}, err
	}
	state, err := randomToken(32)
	if err != nil {
		return domain.ProviderAuthStartResult{}, err
	}
	redirectURI := openAIBrowserRedirectURI()
	authURL, err := buildOpenAIAuthorizeURL(redirectURI, challenge, state)
	if err != nil {
		return domain.ProviderAuthStartResult{}, err
	}
	flow := &providerAuthFlow{
		ProviderID:   "openai",
		Method:       "oauth-browser",
		Status:       "pending",
		State:        state,
		Verifier:     verifier,
		URL:          authURL,
		Instructions: "Complete authorization in your browser.",
		ExpiresAt:    time.Now().Add(5 * time.Minute),
	}
	m.mu.Lock()
	m.flows["openai"] = flow
	m.mu.Unlock()
	return flow.startResult(), nil
}

func (m *ProviderAuthManager) startOpenAIHeadless(ctx context.Context) (domain.ProviderAuthStartResult, error) {
	body := strings.NewReader(`{"client_id":"` + openAIClientID + `"}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIIssuer+"/api/accounts/deviceauth/usercode", body)
	if err != nil {
		return domain.ProviderAuthStartResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", openAIUserAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return domain.ProviderAuthStartResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return domain.ProviderAuthStartResult{}, fmt.Errorf("device authorization failed: %s", resp.Status)
	}
	var payload struct {
		DeviceAuthID string `json:"device_auth_id"`
		UserCode     string `json:"user_code"`
		Interval     string `json:"interval"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return domain.ProviderAuthStartResult{}, err
	}
	flow := &providerAuthFlow{
		ProviderID:   "openai",
		Method:       "oauth-headless",
		Status:       "pending",
		URL:          openAIDeviceURL,
		UserCode:     payload.UserCode,
		DeviceAuthID: payload.DeviceAuthID,
		Instructions: "Open the device authorization URL and enter the displayed code.",
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	}
	m.mu.Lock()
	m.flows["openai"] = flow
	m.mu.Unlock()
	interval := 5 * time.Second
	if parsed, err := time.ParseDuration(payload.Interval + "s"); err == nil && parsed > 0 {
		interval = parsed
	}
	go m.pollOpenAIDevice(flow, interval+3*time.Second)
	return flow.startResult(), nil
}
