package app

import (
	"context"
	"net/http"
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
