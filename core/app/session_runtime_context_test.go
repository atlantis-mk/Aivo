package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"aivo/core/domain"
)

func TestSessionLifecycleVisibilityAndContextBuilder(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	service.now = func() time.Time { return time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC) }
	ctx := context.Background()
	service.now = func() time.Time { return time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC) }
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "Runtime", Goal: "Ship continuity", ProjectPath: t.TempDir(), SystemPromptSnapshot: "system"})
	if err != nil {
		t.Fatal(err)
	}
	normal, err := service.AppendEvent(ctx, domain.AppendEventRequest{SessionID: session.ID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser, Content: "visible"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AppendEvent(ctx, domain.AppendEventRequest{SessionID: session.ID, Type: domain.EventTypeSystemNote, Role: domain.EventRoleSystem, Visibility: domain.EventVisibilityInternal, Content: "secret"}); err != nil {
		t.Fatal(err)
	}
	turn, err := service.StartTurn(ctx, domain.StartTurnRequest{SessionID: session.ID, UserEventID: normal.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.FailTurn(ctx, domain.FailTurnRequest{TurnID: turn.ID, Error: "provider unavailable"}); err != nil {
		t.Fatal(err)
	}
	events, err := service.ListEvents(ctx, session.ID, false, 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Content == "secret" {
			t.Fatalf("internal event leaked in normal listing")
		}
	}
	result, err := service.BuildSessionContext(ctx, domain.BuildSessionContextRequest{SessionID: session.ID, CurrentInput: "next", CharacterBudget: 500})
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	for _, section := range result.Sections {
		joined += section.Content
	}
	if !contains(joined, "visible") || contains(joined, "secret") {
		t.Fatalf("context content = %q", joined)
	}
}

func TestSessionContextAndResumeDoNotInjectProjectInstructions(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	projectRoot := t.TempDir()
	writeSessionProjectFile(t, projectRoot, "AGENTS.md", "# Agent rules\nUse the project rules in context.\n")
	writeSessionProjectFile(t, projectRoot, ".env", "SECRET=value\n")
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: projectRoot})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.BuildSessionContext(ctx, domain.BuildSessionContextRequest{SessionID: session.ID, CharacterBudget: 4000})
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range result.Sections {
		if section.Name == "project_instructions" {
			t.Fatalf("project instructions section should not be injected: %q", section.Content)
		}
	}
	joined := ""
	for _, section := range result.Sections {
		joined += section.Content
	}
	if contains(joined, "project rules") || contains(joined, "SECRET=value") {
		t.Fatalf("project file content leaked into context: %q", joined)
	}
	recap, err := service.ResumeRecap(ctx, domain.ResumeSessionRequest{SessionID: session.ID})
	if err != nil {
		t.Fatal(err)
	}
	if recap.ProjectPath != projectRoot {
		t.Fatalf("resume project path = %q, want %q", recap.ProjectPath, projectRoot)
	}
}

func TestCreateRuntimeSessionWithoutProjectPathCreatesManagedCodingContext(t *testing.T) {
	t.Setenv(managedWorkspaceRootEnv, t.TempDir())
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	service.now = func() time.Time { return time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC) }
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, Title: "https://github.com/example/repo"})
	if err != nil {
		t.Fatal(err)
	}
	cc, err := service.GetCodingContext(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertManagedWorkspace(t, cc.ProjectPath, filepath.Join("2026-06-27", workspaceSlug(session.ID)))
}

func TestCreateCodingSessionDefaultsToCodeAgent(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if session.AgentMode != domain.AgentModeCode {
		t.Fatalf("agent mode = %q, want %q", session.AgentMode, domain.AgentModeCode)
	}
}

func TestSubmitSessionMessageRecreatesDeletedManagedWorkspace(t *testing.T) {
	t.Setenv(managedWorkspaceRootEnv, t.TempDir())
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	service.now = func() time.Time { return time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC) }
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "custom-api", Type: "openai-compatible", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop})
	if err != nil {
		t.Fatal(err)
	}
	before, err := service.GetCodingContext(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	wantSuffix := filepath.Join("2026-06-27", workspaceSlug(session.ID))
	assertManagedWorkspace(t, before.ProjectPath, wantSuffix)
	if err := os.RemoveAll(before.ProjectPath); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "hello"}); err != nil {
		t.Fatal(err)
	}
	after, err := service.GetCodingContext(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertManagedWorkspace(t, after.ProjectPath, wantSuffix)
	if after.ProjectPath != before.ProjectPath {
		t.Fatalf("managed workspace path changed: before=%q after=%q", before.ProjectPath, after.ProjectPath)
	}
}

func TestSummaryCheckpointForkAndResume(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	projectPath := t.TempDir()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "Parent", Goal: "Goal", ProjectPath: projectPath})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateSummary(ctx, domain.CreateSummaryRequest{SessionID: session.ID, Summary: "Summary", OpenTasks: []string{"todo"}, NextSuggestedAction: "continue"}); err != nil {
		t.Fatal(err)
	}
	checkpoint, err := service.CreateCheckpoint(ctx, domain.CreateCheckpointRequest{SessionID: session.ID, ConversationSummary: "Checkpoint", OpenTodos: []string{"todo"}, NextSuggestedAction: "resume"})
	if err != nil {
		t.Fatal(err)
	}
	if len(checkpoint.KnownIssues) == 0 {
		t.Fatalf("checkpoint without git metadata should record known issue")
	}
	fork, err := service.ForkSession(ctx, domain.ForkSessionRequest{SessionID: session.ID, Title: "Fork"})
	if err != nil {
		t.Fatal(err)
	}
	if fork.ForkedFromSessionID != session.ID || fork.Status != domain.SessionStatusActive {
		t.Fatalf("fork = %#v", fork)
	}
	if _, err := service.AppendEvent(ctx, domain.AppendEventRequest{SessionID: fork.ID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser, Content: "fork only"}); err != nil {
		t.Fatal(err)
	}
	sourceEvents, err := service.ListEvents(ctx, session.ID, false, 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range sourceEvents {
		if event.Content == "fork only" {
			t.Fatalf("fork event leaked to source session")
		}
	}
	recap, err := service.ResumeRecap(ctx, domain.ResumeSessionRequest{SessionID: session.ID})
	if err != nil {
		t.Fatal(err)
	}
	if recap.LatestSummary == nil || recap.LatestCheckpoint == nil || recap.ProjectPath == "" || recap.NextSuggestedAction == "" {
		t.Fatalf("recap = %#v", recap)
	}
	latest, err := service.ContinueProjectSession(ctx, projectPath)
	if err != nil || latest == nil || latest.SessionID == "" {
		t.Fatalf("continue project = %#v, %v", latest, err)
	}
}

func TestSubmitSessionMessageBuildsModelVisibleContext(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time {
		now = now.Add(time.Millisecond)
		return now
	}
	var captured []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		captured = body.Messages
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"context ok"}}]}`))
	}))
	defer server.Close()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "custom-api", Type: "openai-compatible", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
		t.Fatal(err)
	}
	projectRoot := t.TempDir()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{
		Type:                 domain.SessionTypeCoding,
		Source:               domain.SessionSourceDesktop,
		Goal:                 "Keep the conversation coherent",
		ProjectPath:          projectRoot,
		SystemPromptSnapshot: "Prefer concise implementation notes",
	})
	if err != nil {
		t.Fatal(err)
	}
	writeSessionProjectFile(t, projectRoot, "AGENTS.md", "# Agent rules\nAlways run focused tests before final response.\n")
	writeSessionProjectFile(t, projectRoot, ".env", "SECRET=value\n")
	for i := 0; i < 35; i++ {
		if _, err := service.AppendEvent(ctx, domain.AppendEventRequest{SessionID: session.ID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser, Content: "old user message " + strconv.Itoa(i)}); err != nil {
			t.Fatal(err)
		}
		if _, err := service.AppendEvent(ctx, domain.AppendEventRequest{SessionID: session.ID, Type: domain.EventTypeAssistantMessage, Role: domain.EventRoleAssistant, Content: "old assistant message " + strconv.Itoa(i)}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := service.CreateSummary(ctx, domain.CreateSummaryRequest{SessionID: session.ID, Summary: "Durable architecture summary", OpenTasks: []string{"preserve current user request"}, NextSuggestedAction: "continue from summary"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "current request"}); err != nil {
		t.Fatal(err)
	}
	if len(captured) == 0 {
		t.Fatal("provider did not receive messages")
	}
	if captured[0].Role != "system" || !contains(captured[0].Content, "Aivo") {
		t.Fatalf("first message = %#v", captured[0])
	}
	joined := ""
	for _, message := range captured {
		joined += "\n" + message.Role + ": " + message.Content
	}
	if !contains(joined, "Durable architecture summary") ||
		!contains(joined, "Keep the conversation coherent") ||
		!contains(joined, "current request") {
		t.Fatalf("context missing expected content: %s", joined)
	}
	if contains(joined, "Always run focused tests") {
		t.Fatalf("project instructions leaked into context: %s", joined)
	}
	if contains(joined, "SECRET=value") {
		t.Fatalf("sensitive project file leaked into context: %s", joined)
	}
	if contains(joined, "old user message 0") {
		t.Fatalf("oldest tail leaked into bounded context: %s", joined)
	}
	if len(captured) >= 72 {
		t.Fatalf("captured %d messages, want bounded context", len(captured))
	}
}

func TestEnsureGeneratedSessionTitleUpdatesDefaultTitle(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	var notified string
	service.SetSessionUpdatedHook(func(sessionID string, _ *domain.Session) {
		notified = sessionID
	})
	service.titleGenerator = func(context.Context, string, *domain.ModelRef) (string, error) {
		return "\"Redis 缓存方案\"\nextra text", nil
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{
		Type:  domain.SessionTypeCoding,
		Title: "帮我写一个 Redis 缓存方案",
		Model: &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5.5"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AppendEvent(ctx, domain.AppendEventRequest{SessionID: session.ID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser, Content: "帮我写一个 Redis 缓存方案"}); err != nil {
		t.Fatal(err)
	}
	service.ensureGeneratedSessionTitle(ctx, session.ID, &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5.5"})
	updated, err := service.GetRuntimeSession(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Redis 缓存方案" {
		t.Fatalf("title = %q", updated.Title)
	}
	if notified != session.ID {
		t.Fatalf("notified session = %q", notified)
	}
}

func TestCleanGeneratedSessionTitle(t *testing.T) {
	got := cleanGeneratedSessionTitle("<think>reasoning</think>\n`Fix React TS2345 error`\nignored")
	if got != "Fix React TS2345 error" {
		t.Fatalf("title = %q", got)
	}
}

func TestResolveTitleModelsPrefersSmallConnectedProviderModel(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{
		ProviderID: "openai",
		Type:       "openai",
		ModelID:    "gpt-5.5",
		Method:     "env",
	}); err != nil {
		t.Fatal(err)
	}
	models := service.resolveTitleModels(ctx, &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5.5"})
	if len(models) != 2 {
		t.Fatalf("models = %#v", models)
	}
	if models[0].ModelID != "gpt-5.4-mini" || models[1].ModelID != "gpt-5.5" {
		t.Fatalf("models = %#v", models)
	}
}

func TestUpdateModelPreferencesPersistsDefaultModelAndReasoningEffort(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{
		ProviderID: "openai",
		Type:       "openai",
		ModelID:    "gpt-5.5",
		Method:     "env",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateModelPreferences(ctx, domain.ModelPreferencesInput{
		Model:           &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5.4-mini"},
		AuxiliaryModel:  &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5-mini"},
		ReasoningEffort: "high",
		ServiceTier:     "priority",
	}); err != nil {
		t.Fatal(err)
	}
	cfg, err := service.AppConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultModel == nil || cfg.DefaultModel.ModelID != "gpt-5.4-mini" {
		t.Fatalf("default model = %#v, want gpt-5.4-mini", cfg.DefaultModel)
	}
	if cfg.Provider == nil || cfg.Provider.Model != "gpt-5.4-mini" {
		t.Fatalf("provider = %#v, want model gpt-5.4-mini", cfg.Provider)
	}
	if cfg.AuxiliaryModel == nil || cfg.AuxiliaryModel.ModelID != "gpt-5-mini" {
		t.Fatalf("auxiliary model = %#v, want gpt-5-mini", cfg.AuxiliaryModel)
	}
	if cfg.ReasoningEffort != "high" {
		t.Fatalf("reasoningEffort = %q, want high", cfg.ReasoningEffort)
	}
	if cfg.ServiceTier != "priority" {
		t.Fatalf("serviceTier = %q, want priority", cfg.ServiceTier)
	}
}
