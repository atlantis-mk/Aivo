package domain

type ProviderAuthStartInput struct {
	ProviderID string `json:"providerId"`
	Method     string `json:"method"`
}

type ProviderAuthStartResult struct {
	ProviderID    string `json:"providerId"`
	Method        string `json:"method"`
	Status        string `json:"status"`
	URL           string `json:"url,omitempty"`
	Instructions  string `json:"instructions,omitempty"`
	UserCode      string `json:"userCode,omitempty"`
	ExpiresAt     string `json:"expiresAt,omitempty"`
	Authorization string `json:"authorization,omitempty"`
}

type ProviderAuthStatus struct {
	ProviderID   string `json:"providerId"`
	Method       string `json:"method"`
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
	AccountID    string `json:"accountId,omitempty"`
	Instructions string `json:"instructions,omitempty"`
	UserCode     string `json:"userCode,omitempty"`
}

type ProviderAuthRecord struct {
	ID              string `json:"id,omitempty"`
	ProviderID      string `json:"providerId"`
	Method          string `json:"method"`
	AccessToken     string `json:"accessToken,omitempty"`
	AccessTokenRef  string `json:"accessTokenRef,omitempty"`
	RefreshToken    string `json:"refreshToken,omitempty"`
	RefreshTokenRef string `json:"refreshTokenRef,omitempty"`
	ExpiresAt       string `json:"expiresAt,omitempty"`
	AccountID       string `json:"accountId,omitempty"`
	DisplayName     string `json:"displayName,omitempty"`
	APIKey          string `json:"apiKey,omitempty"`
	APIKeyRef       string `json:"apiKeyRef,omitempty"`
	UpdatedAt       string `json:"updatedAt"`
}

type ProviderAccountInfo struct {
	ID          string `json:"id"`
	ProviderID  string `json:"providerId"`
	Method      string `json:"method"`
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
	ConnectedAt string `json:"connectedAt,omitempty"`
}
