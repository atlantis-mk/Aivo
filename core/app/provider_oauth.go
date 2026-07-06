package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"aivo/core/domain"
)

const (
	openAIClientID  = "app_EMoamEEZ73f0CkXaXp7hrann"
	openAIIssuer    = "https://auth.openai.com"
	openAIOAuthPort = 1455
	openAIOAuthPath = "/auth/callback"
	openAIDeviceURL = "https://auth.openai.com/codex/device"
	openAIUserAgent = "opencode/aivo"
)

type ProviderAuthManager struct {
	service *Service
	mu      sync.Mutex
	server  *http.Server
	flows   map[string]*providerAuthFlow
	drivers map[string]ProviderAuthDriver
}

type ProviderAuthDriver interface {
	ProviderID() string
	SupportedMethods() []string
	Start(context.Context, domain.ProviderAuthStartInput) (domain.ProviderAuthStartResult, error)
}

type openAIAuthDriver struct {
	manager *ProviderAuthManager
}

type providerAuthFlow struct {
	ProviderID   string
	Method       string
	Status       string
	State        string
	Verifier     string
	Instructions string
	URL          string
	UserCode     string
	DeviceAuthID string
	Error        string
	AccountID    string
	NativeAuthID int64
	ExpiresAt    time.Time
}

type openAITokenResponse struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func NewProviderAuthManager(service *Service) *ProviderAuthManager {
	manager := &ProviderAuthManager{service: service, flows: map[string]*providerAuthFlow{}, drivers: map[string]ProviderAuthDriver{}}
	manager.RegisterDriver(&openAIAuthDriver{manager: manager})
	return manager
}

func (m *ProviderAuthManager) RegisterDriver(driver ProviderAuthDriver) {
	if driver == nil {
		return
	}
	providerID := normalizeProviderID(driver.ProviderID())
	if providerID == "" {
		return
	}
	m.drivers[providerID] = driver
}

func (m *ProviderAuthManager) driver(providerID string) (ProviderAuthDriver, bool) {
	driver, ok := m.drivers[normalizeProviderID(providerID)]
	return driver, ok
}

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

func (m *ProviderAuthManager) ensureOAuthServer() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.server != nil {
		return nil
	}
	mux := http.NewServeMux()
	mux.HandleFunc(openAIOAuthPath, m.handleOpenAICallback)
	mux.HandleFunc("/cancel", func(w http.ResponseWriter, r *http.Request) {
		m.cancel("openai")
		_, _ = io.WriteString(w, "Login cancelled")
	})
	server := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", openAIOAuthPort), Handler: mux}
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return err
	}
	m.server = server
	go func() {
		_ = server.Serve(listener)
	}()
	return nil
}

func (m *ProviderAuthManager) handleOpenAICallback(w http.ResponseWriter, r *http.Request) {
	if err := m.completeOpenAICallbackQuery(r.Context(), r.URL.Query()); err != nil {
		writeOAuthHTML(w, false, err.Error())
		return
	}
	writeOAuthHTML(w, true, "")
}

func (m *ProviderAuthManager) completeOpenAICallbackURL(ctx context.Context, rawCallbackURL string) error {
	callbackURL, err := url.Parse(rawCallbackURL)
	if err != nil {
		m.fail("openai", "invalid OAuth callback URL")
		return errors.New("Invalid OAuth callback URL.")
	}
	return m.completeOpenAICallbackQuery(ctx, callbackURL.Query())
}

func (m *ProviderAuthManager) completeOpenAICallbackQuery(ctx context.Context, query url.Values) error {
	if errText := query.Get("error"); errText != "" {
		message := firstNonEmpty(query.Get("error_description"), errText)
		m.fail("openai", message)
		return errors.New(message)
	}
	code := query.Get("code")
	state := query.Get("state")
	m.mu.Lock()
	flow := m.flows["openai"]
	m.mu.Unlock()
	if flow == nil || flow.State != state || code == "" || time.Now().After(flow.ExpiresAt) {
		m.fail("openai", "invalid or expired OAuth callback")
		return errors.New("Invalid or expired OAuth callback.")
	}
	tokens, err := exchangeOpenAICode(ctx, code, openAIBrowserRedirectURI(), flow.Verifier)
	if err != nil {
		m.fail("openai", err.Error())
		return err
	}
	if err := m.saveOpenAITokens(ctx, tokens, "oauth-browser"); err != nil {
		m.fail("openai", err.Error())
		return err
	}
	nativeAuthID := flow.NativeAuthID
	m.succeed("openai", extractOpenAIAccountID(tokens))
	if nativeAuthID != 0 {
		cancelOpenAINativeWebAuthSession(nativeAuthID)
	}
	if m.service.onAuthSuccess != nil {
		go func() {
			time.Sleep(800 * time.Millisecond)
			m.service.onAuthSuccess()
		}()
	}
	return nil
}

func (m *ProviderAuthManager) pollOpenAIDevice(flow *providerAuthFlow, interval time.Duration) {
	for {
		if time.Now().After(flow.ExpiresAt) {
			m.fail(flow.ProviderID, "device authorization expired")
			return
		}
		tokens, done, err := exchangeOpenAIDevice(flow)
		if err != nil {
			m.fail(flow.ProviderID, err.Error())
			return
		}
		if done {
			if err := m.saveOpenAITokens(context.Background(), tokens, "oauth-headless"); err != nil {
				m.fail(flow.ProviderID, err.Error())
				return
			}
			m.succeed(flow.ProviderID, extractOpenAIAccountID(tokens))
			return
		}
		time.Sleep(interval)
	}
}

func (m *ProviderAuthManager) saveOpenAITokens(ctx context.Context, tokens openAITokenResponse, method string) error {
	expiresIn := tokens.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	provider := domain.ProviderConfig{ID: "openai", Type: "openai", BaseURL: "https://api.openai.com/v1", Model: "gpt-5-codex"}
	if err := m.service.store.SaveProvider(ctx, provider); err != nil {
		return err
	}
	cfg, err := m.service.AppConfig(ctx)
	if err != nil {
		return err
	}
	cfg.Provider = &provider
	cfg.DefaultModel = &domain.ModelRef{ProviderID: provider.ID, ModelID: provider.Model}
	if err := m.service.store.SaveConfig(ctx, cfg); err != nil {
		return err
	}
	return m.service.saveProviderAuth(ctx, domain.ProviderAuthRecord{
		ProviderID:   "openai",
		Method:       method,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresAt:    domain.NowString(time.Now().Add(time.Duration(expiresIn) * time.Second)),
		AccountID:    extractOpenAIAccountID(tokens),
		DisplayName:  extractOpenAIAccountDisplayName(tokens),
		UpdatedAt:    domain.NowString(time.Now()),
	})
}

func (m *ProviderAuthManager) status(providerID string) domain.ProviderAuthStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	flow := m.flows[providerID]
	if flow == nil {
		return domain.ProviderAuthStatus{ProviderID: providerID, Status: "idle"}
	}
	return domain.ProviderAuthStatus{
		ProviderID:   flow.ProviderID,
		Method:       flow.Method,
		Status:       flow.Status,
		Error:        flow.Error,
		AccountID:    flow.AccountID,
		Instructions: flow.Instructions,
		UserCode:     flow.UserCode,
	}
}

func (m *ProviderAuthManager) cancel(providerID string) domain.ProviderAuthStatus {
	m.mu.Lock()
	flow := m.flows[providerID]
	if flow == nil {
		m.mu.Unlock()
		return domain.ProviderAuthStatus{ProviderID: providerID, Status: "idle"}
	}
	flow.Status = "cancelled"
	status := domain.ProviderAuthStatus{
		ProviderID:   flow.ProviderID,
		Method:       flow.Method,
		Status:       flow.Status,
		Error:        flow.Error,
		AccountID:    flow.AccountID,
		Instructions: flow.Instructions,
		UserCode:     flow.UserCode,
	}
	m.mu.Unlock()
	m.emitProviderAuthUpdated(status)
	return status
}

func (m *ProviderAuthManager) fail(providerID string, message string) {
	m.mu.Lock()
	var status domain.ProviderAuthStatus
	if flow := m.flows[providerID]; flow != nil {
		flow.Status = "failed"
		flow.Error = message
		status = m.statusFromFlow(flow)
	}
	m.mu.Unlock()
	m.emitProviderAuthUpdated(status)
}

func (m *ProviderAuthManager) succeed(providerID string, accountID string) {
	m.mu.Lock()
	var status domain.ProviderAuthStatus
	if flow := m.flows[providerID]; flow != nil {
		flow.Status = "success"
		flow.AccountID = accountID
		status = m.statusFromFlow(flow)
	}
	m.mu.Unlock()
	m.emitProviderAuthUpdated(status)
}

func (m *ProviderAuthManager) statusFromFlow(flow *providerAuthFlow) domain.ProviderAuthStatus {
	if flow == nil {
		return domain.ProviderAuthStatus{}
	}
	return domain.ProviderAuthStatus{
		ProviderID:   flow.ProviderID,
		Method:       flow.Method,
		Status:       flow.Status,
		Error:        flow.Error,
		AccountID:    flow.AccountID,
		Instructions: flow.Instructions,
		UserCode:     flow.UserCode,
	}
}

func (m *ProviderAuthManager) emitProviderAuthUpdated(status domain.ProviderAuthStatus) {
	if status.ProviderID == "" || m.service == nil || m.service.onProviderAuthUpdated == nil {
		return
	}
	m.service.onProviderAuthUpdated(status)
}

func (f *providerAuthFlow) startResult() domain.ProviderAuthStartResult {
	return domain.ProviderAuthStartResult{
		ProviderID:   f.ProviderID,
		Method:       f.Method,
		Status:       f.Status,
		URL:          f.URL,
		Instructions: f.Instructions,
		UserCode:     f.UserCode,
		ExpiresAt:    domain.NowString(f.ExpiresAt),
	}
}

func buildOpenAIAuthorizeURL(redirectURI string, challenge string, state string) (string, error) {
	base, err := url.Parse(openAIIssuer + "/oauth/authorize")
	if err != nil {
		return "", err
	}
	query := base.Query()
	query.Set("response_type", "code")
	query.Set("client_id", openAIClientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("scope", "openid profile email offline_access")
	query.Set("code_challenge", challenge)
	query.Set("code_challenge_method", "S256")
	query.Set("id_token_add_organizations", "true")
	query.Set("codex_cli_simplified_flow", "true")
	query.Set("state", state)
	query.Set("originator", "opencode")
	base.RawQuery = query.Encode()
	return base.String(), nil
}

func openAIBrowserRedirectURI() string {
	return fmt.Sprintf("http://localhost:%d%s", openAIOAuthPort, openAIOAuthPath)
}

func exchangeOpenAICode(ctx context.Context, code string, redirectURI string, verifier string) (openAITokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", openAIClientID)
	form.Set("code_verifier", verifier)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIIssuer+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return openAITokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return doOpenAITokenRequest(req)
}

func exchangeOpenAIDevice(flow *providerAuthFlow) (openAITokenResponse, bool, error) {
	reqBody := map[string]string{"device_auth_id": flow.DeviceAuthID, "user_code": flow.UserCode}
	raw, _ := json.Marshal(reqBody)
	req, err := http.NewRequest(http.MethodPost, openAIIssuer+"/api/accounts/deviceauth/token", strings.NewReader(string(raw)))
	if err != nil {
		return openAITokenResponse{}, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", openAIUserAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return openAITokenResponse{}, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
		return openAITokenResponse{}, false, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return openAITokenResponse{}, false, fmt.Errorf("device token polling failed: %s", resp.Status)
	}
	var payload struct {
		AuthorizationCode string `json:"authorization_code"`
		CodeVerifier      string `json:"code_verifier"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return openAITokenResponse{}, false, err
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", payload.AuthorizationCode)
	form.Set("redirect_uri", openAIIssuer+"/deviceauth/callback")
	form.Set("client_id", openAIClientID)
	form.Set("code_verifier", payload.CodeVerifier)
	req, err = http.NewRequest(http.MethodPost, openAIIssuer+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return openAITokenResponse{}, false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokens, err := doOpenAITokenRequest(req)
	return tokens, err == nil, err
}

func doOpenAITokenRequest(req *http.Request) (openAITokenResponse, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return openAITokenResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return openAITokenResponse{}, fmt.Errorf("OpenAI token exchange failed: %s", resp.Status)
	}
	var tokens openAITokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return openAITokenResponse{}, err
	}
	if tokens.AccessToken == "" {
		return openAITokenResponse{}, errors.New("OpenAI token exchange did not return an access token")
	}
	return tokens, nil
}

func generatePKCE() (string, string, error) {
	verifier, err := randomToken(43)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func randomToken(size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func extractOpenAIAccountID(tokens openAITokenResponse) string {
	for _, token := range []string{tokens.IDToken, tokens.AccessToken} {
		claims := parseJWTClaims(token)
		if claims == nil {
			continue
		}
		if accountID, _ := claims["chatgpt_account_id"].(string); accountID != "" {
			return accountID
		}
		if nested, ok := claims["https://api.openai.com/auth"].(map[string]any); ok {
			if accountID, _ := nested["chatgpt_account_id"].(string); accountID != "" {
				return accountID
			}
		}
		if orgs, ok := claims["organizations"].([]any); ok && len(orgs) > 0 {
			if org, ok := orgs[0].(map[string]any); ok {
				if id, _ := org["id"].(string); id != "" {
					return id
				}
			}
		}
	}
	return ""
}

func extractOpenAIAccountDisplayName(tokens openAITokenResponse) string {
	for _, token := range []string{tokens.IDToken, tokens.AccessToken} {
		claims := parseJWTClaims(token)
		if claims == nil {
			continue
		}
		if displayName := firstJWTString(claims, "email", "name", "preferred_username", "nickname"); displayName != "" {
			return displayName
		}
		if nested, ok := claims["https://api.openai.com/auth"].(map[string]any); ok {
			if displayName := firstJWTString(nested, "email", "name", "preferred_username", "nickname"); displayName != "" {
				return displayName
			}
		}
		if orgs, ok := claims["organizations"].([]any); ok && len(orgs) > 0 {
			if org, ok := orgs[0].(map[string]any); ok {
				if displayName := firstJWTString(org, "title", "name", "display_name"); displayName != "" {
					return displayName
				}
			}
		}
	}
	return ""
}

func firstJWTString(claims map[string]any, keys ...string) string {
	for _, key := range keys {
		value, _ := claims[key].(string)
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func parseJWTClaims(token string) map[string]any {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	var claims map[string]any
	if err := json.Unmarshal(raw, &claims); err != nil {
		return nil
	}
	return claims
}

func writeOAuthHTML(w http.ResponseWriter, ok bool, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	title := "Authorization Successful"
	body := "You can close this window and return to Aivo."
	script := `<script>
setTimeout(function() {
  window.open("", "_self");
  window.close();
  setTimeout(function() {
    var body = document.getElementById("oauth-message");
    if (body) body.textContent = "Authorization is complete. Your browser may block automatic tab closing, so you can close this tab manually.";
  }, 500);
}, 2000);
</script>`
	if !ok {
		title = "Authorization Failed"
		body = message
		script = ""
	}
	_, _ = fmt.Fprintf(w, `<!doctype html><html><head><title>Aivo - %s</title><style>body{font-family:system-ui,-apple-system,sans-serif;display:flex;justify-content:center;align-items:center;height:100vh;margin:0;background:#131010;color:#f1ecec}.container{text-align:center;padding:2rem}p{color:#b7b1b1}</style></head><body><div class="container"><h1>%s</h1><p id="oauth-message">%s</p></div>%s</body></html>`, html.EscapeString(title), html.EscapeString(title), html.EscapeString(body), script)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
