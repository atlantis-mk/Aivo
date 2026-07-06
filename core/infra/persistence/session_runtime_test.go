package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"aivo/core/domain"
)

func TestSessionRuntimeMigrationEmptyDatabase(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "aivo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for _, table := range []string{"turns", "session_events", "tool_calls", "permission_requests", "permission_rules", "session_summaries", "session_checkpoints", "coding_contexts", "agent_runs", "todo_items", "scheduled_jobs"} {
		if !store.db.Migrator().HasTable(table) {
			t.Fatalf("table %s missing", table)
		}
	}
	if ok, err := store.hasColumn(context.Background(), "sessions", "status"); err != nil || !ok {
		t.Fatalf("sessions.status migration = %v, %v", ok, err)
	}
	if ok, err := store.hasColumn(context.Background(), "app_config", "web_search"); err != nil || !ok {
		t.Fatalf("app_config.web_search migration = %v, %v", ok, err)
	}
	if ok, err := store.hasColumn(context.Background(), "app_config", "native_tools"); err != nil || !ok {
		t.Fatalf("app_config.native_tools migration = %v, %v", ok, err)
	}
}

func TestWebSearchConfigPersistsInAppConfig(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	cfg, err := store.LoadConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cfg.WebSearch = domain.WebSearchConfig{
		Mode:              domain.WebSearchModeLive,
		Route:             domain.WebSearchRouteProvider,
		LocalProvider:     "duckduckgo",
		SearchContextSize: "high",
		AllowedDomains:    []string{"Example.com", "https://docs.example.com/"},
		UserLocation:      &domain.WebSearchUserLocation{Country: "US", Region: "CA", City: "San Francisco"},
	}
	if err := store.SaveConfig(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	got, err := reopened.LoadConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.WebSearch.Route != domain.WebSearchRouteProvider || got.WebSearch.SearchContextSize != "high" {
		t.Fatalf("web search config = %#v", got.WebSearch)
	}
	if len(got.WebSearch.AllowedDomains) != 2 || got.WebSearch.AllowedDomains[0] != "example.com" || got.WebSearch.AllowedDomains[1] != "docs.example.com" {
		t.Fatalf("allowed domains = %#v", got.WebSearch.AllowedDomains)
	}
	if got.WebSearch.UserLocation == nil || got.WebSearch.UserLocation.Type != "approximate" || got.WebSearch.UserLocation.City != "San Francisco" {
		t.Fatalf("user location = %#v", got.WebSearch.UserLocation)
	}
}

func TestNativeToolsConfigPersistsInAppConfig(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	cfg, err := store.LoadConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cfg.NativeTools = domain.NativeToolsConfig{
		XSearch:       domain.NativeToolToggle{Enabled: true},
		CodeExecution: domain.NativeCodeExecutionConfig{Enabled: true, FileIDs: []string{" file_1 ", "file_1", "file_2"}},
		FileSearch:    domain.NativeFileSearchConfig{Enabled: true, VectorStoreIDs: []string{"vs_1", "vs_1"}},
		RemoteMCP: []domain.NativeMCPToolConfig{
			{Enabled: true, ServerURL: " https://mcp.example.com ", ServerLabel: "docs", AllowedTools: []string{"search", "search"}},
			{Enabled: false, ServerURL: "https://ignored.example.com", ServerLabel: "ignored"},
		},
	}
	if err := store.SaveConfig(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	got, err := reopened.LoadConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !got.NativeTools.XSearch.Enabled || !got.NativeTools.CodeExecution.Enabled || len(got.NativeTools.CodeExecution.FileIDs) != 2 {
		t.Fatalf("native tools = %#v", got.NativeTools)
	}
	if !got.NativeTools.FileSearch.Enabled || len(got.NativeTools.FileSearch.VectorStoreIDs) != 1 || got.NativeTools.FileSearch.VectorStoreIDs[0] != "vs_1" {
		t.Fatalf("file search config = %#v", got.NativeTools.FileSearch)
	}
	if len(got.NativeTools.RemoteMCP) != 1 || got.NativeTools.RemoteMCP[0].ServerURL != "https://mcp.example.com" || len(got.NativeTools.RemoteMCP[0].AllowedTools) != 1 {
		t.Fatalf("remote mcp config = %#v", got.NativeTools.RemoteMCP)
	}
}

func TestAgentRuntimePersistenceCRUD(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "aivo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	session, err := store.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	run, err := store.SaveAgentRun(ctx, domain.AgentRun{ParentSessionID: session.ID, SessionID: session.ID, Mode: domain.AgentModePlanner, Status: domain.AgentRunStatusRunning, Prompt: "plan"})
	if err != nil || run.ID == "" {
		t.Fatalf("run = %#v, %v", run, err)
	}
	runs, err := store.ListAgentRuns(ctx, domain.AgentRunListRequest{SessionID: session.ID, Status: domain.AgentRunStatusRunning})
	if err != nil || len(runs) != 1 {
		t.Fatalf("runs = %#v, %v", runs, err)
	}
	plan, err := store.ReplaceTodoItems(ctx, domain.TodoListInput{SessionID: session.ID, ProjectPath: session.ProjectPath}, []domain.TodoItem{
		{Title: "inspect code", Status: domain.TodoStatusCompleted, OwnerMode: domain.AgentModeAssistant},
		{Title: "write tests", Status: domain.TodoStatusInProgress, OwnerMode: domain.AgentModeAssistant},
	})
	if err != nil || len(plan) != 2 || plan[0].ID == "" {
		t.Fatalf("plan = %#v, %v", plan, err)
	}
	todos, err := store.ListTodoItems(ctx, domain.TodoListInput{SessionID: session.ID, Status: domain.TodoStatusInProgress})
	if err != nil || len(todos) != 1 {
		t.Fatalf("todos = %#v, %v", todos, err)
	}
	job, err := store.SaveScheduledJob(ctx, domain.ScheduledJob{SessionID: session.ID, Title: "watch", Prompt: "check status", Schedule: "once", WorkerMode: domain.AgentModeSchedulerWorker, Toolsets: []string{"safe"}, PermissionScope: "read_only", Status: domain.ScheduledJobStatusActive, NextRunAt: "2026-01-01T00:00:00Z"})
	if err != nil || job.ID == "" {
		t.Fatalf("job = %#v, %v", job, err)
	}
	due, err := store.ListDueScheduledJobs(ctx, "2026-01-01T00:00:01Z", 10)
	if err != nil || len(due) != 1 {
		t.Fatalf("due = %#v, %v", due, err)
	}
	if err := store.DeleteScheduledJob(ctx, job.ID); err != nil {
		t.Fatal(err)
	}
	jobs, err := store.ListScheduledJobs(ctx, domain.ScheduledJobListInput{SessionID: session.ID})
	if err != nil || len(jobs) != 0 {
		t.Fatalf("jobs = %#v, %v", jobs, err)
	}
}

func TestSessionRuntimeMigrationLegacyCompatibility(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE sessions (id TEXT PRIMARY KEY, title TEXT NOT NULL, project_id TEXT, model_provider_id TEXT, model_id TEXT, time_created TEXT NOT NULL, time_updated TEXT NOT NULL);
CREATE TABLE messages (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, role TEXT NOT NULL, text TEXT NOT NULL, time_created TEXT NOT NULL);
INSERT INTO sessions(id, title, time_created, time_updated) VALUES ('s1', 'legacy', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
INSERT INTO messages(id, session_id, role, text, time_created) VALUES ('m1', 's1', 'user', 'hello', '2026-01-01T00:00:00Z');`)
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	session, err := store.GetRuntimeSession(context.Background(), "s1")
	if err != nil {
		t.Fatal(err)
	}
	if session.Status != domain.SessionStatusActive || session.Type != domain.SessionTypeCoding {
		t.Fatalf("legacy defaults = %#v", session)
	}
	events, err := store.ListSessionEvents(context.Background(), "s1", false, 10)
	if err != nil || len(events) != 1 || events[0].Content != "hello" || events[0].Type != domain.EventTypeUserMessage {
		t.Fatalf("legacy events = %#v, %v", events, err)
	}
	if store.db.Migrator().HasTable("messages") {
		t.Fatalf("legacy messages table was not dropped")
	}
}

func TestConfigPersistsModelPreferencesAcrossReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	ctx := context.Background()
	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConfig(ctx, domain.AppConfig{
		Initialized: true,
		Provider: &domain.ProviderConfig{
			ID:    "openai",
			Type:  "openai",
			Model: "gpt-5.4-mini",
		},
		DefaultModel:    &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5.4-mini"},
		ReasoningEffort: "high",
		ServiceTier:     "priority",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	cfg, err := reopened.LoadConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultModel == nil || cfg.DefaultModel.ModelID != "gpt-5.4-mini" {
		t.Fatalf("default model = %#v, want gpt-5.4-mini", cfg.DefaultModel)
	}
	if cfg.ReasoningEffort != "high" {
		t.Fatalf("reasoningEffort = %q, want high", cfg.ReasoningEffort)
	}
	if cfg.ServiceTier != "priority" {
		t.Fatalf("serviceTier = %q, want priority", cfg.ServiceTier)
	}
}

func TestProviderConfigPersistsAcrossReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	ctx := context.Background()
	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	err = store.SaveProvider(ctx, domain.ProviderConfig{
		ID:        "team-proxy",
		Type:      "anthropic_messages",
		BaseURL:   "https://proxy.example.com/anthropic/v1",
		APIKeyEnv: "TEAM_PROXY_KEY",
		Model:     "claude-sonnet-4-proxy",
		Headers:   map[string]string{"X-Team": "agent"},
		RequestParams: map[string]any{
			"temperature": 0.2,
			"provider":    map[string]any{"sort": "throughput"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	providers, err := reopened.ListProviders(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(providers) != 1 {
		t.Fatalf("len(providers) = %d, want 1: %+v", len(providers), providers)
	}
	got := providers[0]
	if got.ID != "team-proxy" || got.Type != "anthropic_messages" || got.BaseURL != "https://proxy.example.com/anthropic/v1" || got.APIKeyEnv != "TEAM_PROXY_KEY" || got.Model != "claude-sonnet-4-proxy" {
		t.Fatalf("provider = %+v, want persisted config", got)
	}
	if got.Headers["X-Team"] != "agent" {
		t.Fatalf("headers = %+v, want X-Team", got.Headers)
	}
	if got.RequestParams["temperature"] != float64(0.2) {
		t.Fatalf("requestParams.temperature = %#v, want 0.2", got.RequestParams["temperature"])
	}
	providerParams, _ := got.RequestParams["provider"].(map[string]any)
	if providerParams["sort"] != "throughput" {
		t.Fatalf("requestParams.provider = %#v, want sort throughput", providerParams)
	}
}

func TestProviderModelCacheAndValidationPersistAcrossReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	ctx := context.Background()
	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	cache := domain.ProviderModelCache{
		ProviderID: "custom-api",
		Models: []domain.ModelInfo{{
			ID: "team-model", ProviderID: "custom-api", Name: "Team Model", Recommended: true,
			Capabilities: []string{"tools"}, ToolSupport: true,
		}},
		DefaultModel: "team-model",
		Strategy:     "openai_compatible",
		ParserType:   "openai-compatible",
		Endpoint:     "https://proxy.example.com/v1/models",
		CacheSource:  "remote",
		Status:       "ready",
		RefreshedAt:  "2026-01-01T00:00:00Z",
		UpdatedAt:    "2026-01-01T00:00:00Z",
	}
	if err := store.SaveProviderModelCache(ctx, cache); err != nil {
		t.Fatal(err)
	}
	validation := domain.ProviderValidationResult{
		ProviderID: "custom-api", Ready: true, Status: "ready", Transport: "openai_compatible",
		AuthMode: "api-key", BaseURL: "https://proxy.example.com/v1", DefaultModel: "team-model",
		ModelCount: 1, Models: cache.Models, CheckedAt: "2026-01-01T00:00:00Z",
	}
	if err := store.SaveProviderValidation(ctx, validation); err != nil {
		t.Fatal(err)
	}
	health := domain.ProviderHealth{
		ProviderID:       "custom-api",
		Status:           "degraded",
		LastFailureAt:    "2026-01-01T00:01:00Z",
		LastLatencyMs:    1234,
		LastErrorClass:   "rate_limit",
		LastErrorMessage: "too many requests",
		LastHTTPStatus:   429,
		FailureCount:     2,
		UpdatedAt:        "2026-01-01T00:01:00Z",
	}
	if err := store.SaveProviderHealth(ctx, health); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	gotCache, err := reopened.LoadProviderModelCache(ctx, "custom-api")
	if err != nil {
		t.Fatal(err)
	}
	if gotCache == nil || gotCache.DefaultModel != "team-model" || len(gotCache.Models) != 1 || !gotCache.Models[0].ToolSupport {
		t.Fatalf("cache = %+v, want persisted model metadata", gotCache)
	}
	gotValidation, err := reopened.LoadProviderValidation(ctx, "custom-api")
	if err != nil {
		t.Fatal(err)
	}
	if gotValidation == nil || !gotValidation.Ready || gotValidation.DefaultModel != "team-model" || gotValidation.ModelCount != 1 {
		t.Fatalf("validation = %+v, want persisted ready validation", gotValidation)
	}
	gotHealth, err := reopened.LoadProviderHealth(ctx, "custom-api")
	if err != nil {
		t.Fatal(err)
	}
	if gotHealth == nil || gotHealth.Status != "degraded" || gotHealth.LastErrorClass != "rate_limit" || gotHealth.LastHTTPStatus != 429 || gotHealth.FailureCount != 2 {
		t.Fatalf("health = %+v, want persisted degraded health", gotHealth)
	}
}

func TestProviderAuthSecretReferencesPersistAcrossReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	ctx := context.Background()
	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProviderAuth(ctx, domain.ProviderAuthRecord{
		ProviderID:      "custom-api",
		Method:          "api-key",
		AccountID:       "API Key",
		APIKeyRef:       "provider-auth/custom-api/api-key/default/api-key",
		AccessTokenRef:  "provider-auth/custom-api/oauth/default/access-token",
		RefreshTokenRef: "provider-auth/custom-api/oauth/default/refresh-token",
		UpdatedAt:       "2026-01-01T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	auth, err := reopened.LoadProviderAuth(ctx, "custom-api")
	if err != nil {
		t.Fatal(err)
	}
	if auth == nil {
		t.Fatal("auth missing")
	}
	if auth.APIKey != "" || auth.AccessToken != "" || auth.RefreshToken != "" {
		t.Fatalf("plaintext secrets = %+v, want empty", auth)
	}
	if auth.APIKeyRef == "" || auth.AccessTokenRef == "" || auth.RefreshTokenRef == "" {
		t.Fatalf("secret refs = %+v, want refs", auth)
	}
}

func TestAppConfigFallbackModelsPersistAcrossReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	ctx := context.Background()
	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg := domain.AppConfig{
		Initialized:     true,
		Provider:        &domain.ProviderConfig{ID: "openai", Type: "openai", Model: "gpt-5.5"},
		DefaultModel:    &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5.5"},
		AuxiliaryModel:  &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5.4-mini"},
		FallbackModels:  []domain.ModelRef{{ProviderID: "anthropic", ModelID: "claude-sonnet-4"}},
		ReasoningEffort: "medium",
		ServiceTier:     "default",
	}
	enableFallback := false
	bufferStreaming := false
	cfg.ProviderPolicy = domain.ProviderRuntimePolicy{
		EnableFallback:           &enableFallback,
		BufferStreamingFallback:  &bufferStreaming,
		MaxRetries:               0,
		RetryBaseDelayMs:         250,
		RateLimitCooldownSeconds: 45,
	}
	if err := store.SaveConfig(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	got, err := reopened.LoadConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.FallbackModels) != 1 || got.FallbackModels[0].ProviderID != "anthropic" || got.FallbackModels[0].ModelID != "claude-sonnet-4" {
		t.Fatalf("fallback models = %+v, want persisted anthropic fallback", got.FallbackModels)
	}
	if got.AuxiliaryModel == nil || got.AuxiliaryModel.ProviderID != "openai" || got.AuxiliaryModel.ModelID != "gpt-5.4-mini" {
		t.Fatalf("auxiliary model = %+v, want persisted openai gpt-5.4-mini", got.AuxiliaryModel)
	}
	if got.ProviderPolicy.EnableFallback == nil || *got.ProviderPolicy.EnableFallback || got.ProviderPolicy.BufferStreamingFallback == nil || *got.ProviderPolicy.BufferStreamingFallback {
		t.Fatalf("provider policy = %+v, want fallback/buffer disabled", got.ProviderPolicy)
	}
	if got.ProviderPolicy.MaxRetries != 0 || got.ProviderPolicy.RetryBaseDelayMs != 250 || got.ProviderPolicy.RateLimitCooldownSeconds != 45 {
		t.Fatalf("provider policy = %+v, want persisted retry/cooldown policy", got.ProviderPolicy)
	}
}

func TestProviderCallEventsPersistAcrossReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	ctx := context.Background()
	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	event := domain.ProviderCallEvent{
		ID:            "event-1",
		ProviderID:    "openai",
		ModelID:       "gpt-5.5",
		Transport:     "openai_responses",
		Status:        "failed",
		ErrorClass:    "rate_limit",
		ErrorMessage:  "too many requests",
		HTTPStatus:    429,
		LatencyMs:     123,
		InputTokens:   10,
		OutputTokens:  5,
		TotalTokens:   15,
		CostMicros:    100,
		Estimated:     true,
		Attempt:       2,
		FallbackIndex: 0,
		Streaming:     true,
		ToolCallCount: 1,
		CreatedAt:     "2026-01-01T00:00:00Z",
	}
	if err := store.SaveProviderCallEvent(ctx, event); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	events, err := reopened.ListProviderCallEvents(ctx, "openai", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %+v, want one event", events)
	}
	got := events[0]
	if got.ID != "event-1" || got.ErrorClass != "rate_limit" || got.HTTPStatus != 429 || !got.Streaming || got.ToolCallCount != 1 || got.TotalTokens != 15 || got.CostMicros != 100 || !got.Estimated {
		t.Fatalf("event = %+v, want persisted call event", got)
	}
}

func TestDeleteProviderRemovesProviderStateButKeepsCallEvents(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	ctx := context.Background()
	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProvider(ctx, domain.ProviderConfig{ID: "custom-api", Type: "openai_compatible", Model: "local-model"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProviderModelCache(ctx, domain.ProviderModelCache{ProviderID: "custom-api", Models: []domain.ModelInfo{{ID: "local-model", ProviderID: "custom-api", Name: "Local"}}, Status: "ready"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProviderValidation(ctx, domain.ProviderValidationResult{ProviderID: "custom-api", Status: "ready", Ready: true, CheckedAt: "2026-01-01T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProviderHealth(ctx, domain.ProviderHealth{ProviderID: "custom-api", Status: "ready", UpdatedAt: "2026-01-01T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProviderAuth(ctx, domain.ProviderAuthRecord{ProviderID: "custom-api", Method: "api-key", APIKeyRef: "ref", UpdatedAt: "2026-01-01T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProviderCallEvent(ctx, domain.ProviderCallEvent{ID: "event-1", ProviderID: "custom-api", ModelID: "local-model", Status: "success", CreatedAt: "2026-01-01T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteProvider(ctx, "custom-api"); err != nil {
		t.Fatal(err)
	}
	providers, err := store.ListProviders(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(providers) != 0 {
		t.Fatalf("providers = %+v, want deleted", providers)
	}
	cache, err := store.LoadProviderModelCache(ctx, "custom-api")
	if err != nil || cache != nil {
		t.Fatalf("cache = %+v, err = %v, want deleted", cache, err)
	}
	health, err := store.LoadProviderHealth(ctx, "custom-api")
	if err != nil || health != nil {
		t.Fatalf("health = %+v, err = %v, want deleted", health, err)
	}
	auth, err := store.LoadProviderAuth(ctx, "custom-api")
	if err != nil || auth != nil {
		t.Fatalf("auth = %+v, err = %v, want deleted", auth, err)
	}
	events, err := store.ListProviderCallEvents(ctx, "custom-api", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %+v, want retained audit event", events)
	}
}

func TestSessionRuntimeJSONRoundTripAndFilters(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "aivo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	session, err := store.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceWeb, Title: "Build runtime", ProjectPath: t.TempDir(), Metadata: map[string]string{"k": "v"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.UpsertCodingContext(ctx, domain.CodingContext{SessionID: session.ID, ProjectPath: session.ProjectPath, ChangedFiles: []string{"main.go"}, LanguageStack: []string{"go"}, Permissions: []string{"local-filesystem"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AppendSessionEvent(ctx, domain.SessionEvent{ID: "e1", SessionID: session.ID, Type: domain.EventTypeToolCall, Role: domain.EventRoleTool, Visibility: domain.EventVisibilityNormal, Content: "ran test", Payload: map[string]any{"command": "go test"}, TimeCreated: "2026-01-01T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	events, err := store.ListSessionEvents(ctx, session.ID, false, 10)
	if err != nil || len(events) != 1 || events[0].Payload["command"] != "go test" {
		t.Fatalf("events = %#v, %v", events, err)
	}
	results, err := store.ListRuntimeSessions(ctx, domain.ListSessionsRequest{Type: domain.SessionTypeCoding, Search: "runtime", Limit: 10})
	if err != nil || len(results) != 1 || results[0].Metadata["k"] != "v" {
		t.Fatalf("results = %#v, %v", results, err)
	}
}

func TestListSessionEventsReturnsLatestLimitInChronologicalOrder(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "aivo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	session, err := store.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeGeneric, Source: domain.SessionSourceDesktop, Title: "limited"})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if err := store.AppendSessionEvent(ctx, domain.SessionEvent{
			ID:          fmt.Sprintf("e%d", i),
			SessionID:   session.ID,
			Type:        domain.EventTypeUserMessage,
			Role:        domain.EventRoleUser,
			Visibility:  domain.EventVisibilityNormal,
			Content:     fmt.Sprintf("event %d", i),
			TimeCreated: fmt.Sprintf("2026-01-01T00:00:0%dZ", i),
		}); err != nil {
			t.Fatal(err)
		}
	}
	events, err := store.ListSessionEvents(ctx, session.ID, false, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 || events[0].Content != "event 2" || events[1].Content != "event 3" || events[2].Content != "event 4" {
		t.Fatalf("events = %#v", events)
	}
}
