package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	coreapp "aivo/core/app"
	"aivo/core/domain"
	"aivo/core/infra/persistence"
)

func TestRunProviderSmokeReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Fatalf("path = %q, want /models", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"local-model","name":"Local Model"}]}`))
	}))
	defer server.Close()
	store, err := persistence.Open(filepath.Join(t.TempDir(), "aivo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.SaveProvider(context.Background(), domain.ProviderConfig{
		ID:      "smoke-provider",
		Type:    "openai_compatible",
		BaseURL: server.URL,
		Model:   "local-model",
	}); err != nil {
		t.Fatal(err)
	}
	service := coreapp.NewService(store)
	if err := service.RegisterProviderDefinition(coreapp.ProviderDefinition{
		ID:              "smoke-provider",
		DisplayName:     "Smoke Provider",
		Transport:       coreapp.TransportOpenAICompatible,
		DefaultBaseURL:  server.URL,
		DefaultModelID:  "local-model",
		AuthTypes:       []coreapp.AuthType{coreapp.AuthNone},
		DefaultAuthType: coreapp.AuthNone,
		Models: []domain.ModelInfo{{
			ID:         "local-model",
			ProviderID: "smoke-provider",
			Name:       "Local Model",
			Streaming:  true,
		}},
	}); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	var stderr bytes.Buffer
	code := runProviderSmoke(context.Background(), service, domain.ProviderIntegrationCheckInput{ProviderID: "smoke-provider"}, &out, &stderr)
	if code != 0 {
		t.Fatalf("code = %d stderr=%q out=%q", code, stderr.String(), out.String())
	}
	if !strings.Contains(out.String(), `"ready": true`) || !strings.Contains(out.String(), `"providerId": "smoke-provider"`) {
		t.Fatalf("out = %q, want ready smoke-provider JSON", out.String())
	}
}

func TestRunProviderSmokeCommandRequiresProvider(t *testing.T) {
	var out bytes.Buffer
	var stderr bytes.Buffer
	code := runProviderSmokeCommand(context.Background(), nil, &out, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "requires --provider") {
		t.Fatalf("stderr = %q, want provider requirement", stderr.String())
	}
}

func TestCoreServerAddrUsesEnvOverride(t *testing.T) {
	t.Setenv("AIVO_CORE_ADDR", "")
	if got := coreServerAddr(); got != "127.0.0.1:43117" {
		t.Fatalf("addr = %q, want default", got)
	}
	t.Setenv("AIVO_CORE_ADDR", "127.0.0.1:0")
	if got := coreServerAddr(); got != "127.0.0.1:0" {
		t.Fatalf("addr = %q, want env override", got)
	}
}

func TestWriteCoreReadyRecordAnnouncesVersionedLoopbackOrigin(t *testing.T) {
	var out bytes.Buffer
	if err := writeCoreReadyRecord(&out, "127.0.0.1:54321"); err != nil {
		t.Fatalf("writeCoreReadyRecord: %v", err)
	}
	want := "AIVO_CORE_READY {\"version\":1,\"url\":\"http://127.0.0.1:54321\"}\n"
	if got := out.String(); got != want {
		t.Fatalf("record = %q, want %q", got, want)
	}
}

func TestWriteCoreReadyRecordRejectsUnsafeEndpoints(t *testing.T) {
	for _, addr := range []string{
		"127.0.0.1:0",
		"0.0.0.0:54321",
		"[::1]:54321",
		"localhost:54321",
		"127.0.0.1:not-a-port",
	} {
		t.Run(addr, func(t *testing.T) {
			if err := writeCoreReadyRecord(io.Discard, addr); err == nil {
				t.Fatalf("writeCoreReadyRecord(%q) succeeded, want refusal", addr)
			}
		})
	}
}
