package app

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestExtractOpenAIAccountDisplayNamePrefersEmail(t *testing.T) {
	tokens := openAITokenResponse{IDToken: testJWT(t, map[string]any{
		"email": "user@example.com",
		"name":  "Example User",
	})}

	if got := extractOpenAIAccountDisplayName(tokens); got != "user@example.com" {
		t.Fatalf("extractOpenAIAccountDisplayName() = %q, want %q", got, "user@example.com")
	}
}

func TestExtractOpenAIAccountDisplayNameUsesNestedAuthClaims(t *testing.T) {
	tokens := openAITokenResponse{AccessToken: testJWT(t, map[string]any{
		"https://api.openai.com/auth": map[string]any{
			"name": "Example User",
		},
	})}

	if got := extractOpenAIAccountDisplayName(tokens); got != "Example User" {
		t.Fatalf("extractOpenAIAccountDisplayName() = %q, want %q", got, "Example User")
	}
}

func testJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "none"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload) + "."
}
