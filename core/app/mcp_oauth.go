package app

import (
	"time"
)

const mcpOAuthDiscoveryLimit = 2 << 20

const (
	mcpOAuthPort = 1456
	mcpOAuthPath = "/mcp/oauth/callback"
)

type mcpOAuthFlow struct {
	ServerID      string
	Status        string
	State         string
	Verifier      string
	URL           string
	Error         string
	ClientID      string
	TokenEndpoint string
	Resource      string
	RedirectURI   string
	ExpiresAt     time.Time
}

type mcpOAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}
