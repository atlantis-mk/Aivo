package app

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
)

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
