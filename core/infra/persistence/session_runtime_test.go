package persistence

import (
	"context"
	"database/sql"
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

func TestProjectDescriptionAndSessionProjectSwitchPersist(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "aivo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	first := t.TempDir()
	second := t.TempDir()
	if _, err := store.UpsertProject(ctx, first); err != nil {
		t.Fatal(err)
	}
	project, err := store.UpdateProjectDescription(ctx, first, "A test project for project search.")
	if err != nil {
		t.Fatal(err)
	}
	if project.Description != "A test project for project search." {
		t.Fatalf("description = %q", project.Description)
	}
	session, err := store.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: first})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.SetRuntimeSessionProject(ctx, session.ID, second)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ProjectPath != second {
		t.Fatalf("project path = %q, want %q", updated.ProjectPath, second)
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
		DefaultModel:          &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5.4-mini"},
		ReasoningEffort:       "high",
		ServiceTier:           "priority",
		DefaultPermissionMode: domain.PermissionModeFullAccess,
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
	if cfg.DefaultPermissionMode != domain.PermissionModeFullAccess {
		t.Fatalf("defaultPermissionMode = %q, want full access", cfg.DefaultPermissionMode)
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
			Capabilities: []string{"tools"}, DeclaredCapabilities: []string{"tools", "reasoning"},
			NativeTools: []string{"code_execution"}, NativeToolsKnown: true, ToolSupport: true,
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
	if gotCache == nil || gotCache.DefaultModel != "team-model" || len(gotCache.Models) != 1 || !gotCache.Models[0].ToolSupport || len(gotCache.Models[0].DeclaredCapabilities) != 2 || !gotCache.Models[0].NativeToolsKnown || len(gotCache.Models[0].NativeTools) != 1 {
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
