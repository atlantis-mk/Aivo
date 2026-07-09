package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"aivo/core/domain"
)

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
