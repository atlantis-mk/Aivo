package domain

type ProviderConnectInput struct {
	ProviderID    string            `json:"providerId"`
	Name          string            `json:"name,omitempty"`
	Type          string            `json:"type,omitempty"`
	BaseURL       string            `json:"baseUrl,omitempty"`
	APIKey        string            `json:"apiKey,omitempty"`
	APIKeyEnv     string            `json:"apiKeyEnv,omitempty"`
	ModelID       string            `json:"modelId,omitempty"`
	Method        string            `json:"method,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	RequestParams map[string]any    `json:"requestParams,omitempty"`
}

type ProviderCallEventsInput struct {
	ProviderID string `json:"providerId,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type ProviderUsageInput struct {
	ProviderID string `json:"providerId,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type ProviderUsageStats struct {
	ProviderID        string `json:"providerId,omitempty"`
	CallCount         int    `json:"callCount"`
	SuccessCount      int    `json:"successCount"`
	FailureCount      int    `json:"failureCount"`
	InputTokens       int    `json:"inputTokens"`
	OutputTokens      int    `json:"outputTokens"`
	TotalTokens       int    `json:"totalTokens"`
	CostMicros        int64  `json:"costMicros"`
	Estimated         bool   `json:"estimated"`
	LastCallAt        string `json:"lastCallAt,omitempty"`
	LastErrorClass    string `json:"lastErrorClass,omitempty"`
	LastErrorMessage  string `json:"lastErrorMessage,omitempty"`
	LastErrorProvider string `json:"lastErrorProvider,omitempty"`
}

type ProviderIntegrationCheckInput struct {
	ProviderID       string `json:"providerId"`
	ModelID          string `json:"modelId,omitempty"`
	IncludeModelList bool   `json:"includeModelList,omitempty"`
}

type ProviderIntegrationCheckResult struct {
	ProviderID   string                         `json:"providerId"`
	ModelID      string                         `json:"modelId,omitempty"`
	Ready        bool                           `json:"ready"`
	Status       string                         `json:"status"`
	CheckedAt    string                         `json:"checkedAt"`
	Steps        []ProviderIntegrationCheckStep `json:"steps"`
	Transport    string                         `json:"transport,omitempty"`
	BaseURL      string                         `json:"baseUrl,omitempty"`
	AuthMode     string                         `json:"authMode,omitempty"`
	ModelCount   int                            `json:"modelCount,omitempty"`
	Capabilities []string                       `json:"capabilities,omitempty"`
	Policy       ProviderRuntimePolicy          `json:"policy,omitempty"`
	Health       *ProviderHealth                `json:"health,omitempty"`
	Usage        *ProviderUsageStats            `json:"usage,omitempty"`
	Validation   *ProviderValidationResult      `json:"validation,omitempty"`
	RecentEvents []ProviderCallEvent            `json:"recentEvents,omitempty"`
	Recommended  []string                       `json:"recommended,omitempty"`
}

type ProviderIntegrationCheckStep struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}
