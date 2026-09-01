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

func TestSessionContextInjectsProjectInstructionsWithoutReadingUnrelatedFiles(t *testing.T) {
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
	joined := ""
	for _, section := range result.Sections {
		joined += section.Content
	}
	if !contains(joined, "project rules") || contains(joined, "SECRET=value") {
		t.Fatalf("project instruction handling = %q", joined)
	}
	recap, err := service.ResumeRecap(ctx, domain.ResumeSessionRequest{SessionID: session.ID})
	if err != nil {
		t.Fatal(err)
	}
	if recap.ProjectPath != projectRoot {
		t.Fatalf("resume project path = %q, want %q", recap.ProjectPath, projectRoot)
	}
}

func TestSessionContextIncludesLiveTerminalInventory(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	projectRoot := t.TempDir()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{
		Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: projectRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	manager := NewAgentPTYRegistry()
	service.ptyManager = manager
	defer manager.Shutdown()
	runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	terminal, err := manager.Start(runCtx, SandboxRequest{
		WorkspaceRoot: projectRoot, CWD: projectRoot, SessionID: session.ID,
		Command: `printf 'ready\n'; read answer; printf '%s\n' "$answer"`, EnvAllowlist: defaultEnvAllowlist(),
	}, 24, 80, time.Second, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if terminal.Status != AgentPTYStatusRunning {
		t.Fatalf("terminal status = %q, want running", terminal.Status)
	}

	result, err := service.BuildSessionContext(ctx, domain.BuildSessionContextRequest{SessionID: session.ID, CharacterBudget: 8000})
	if err != nil {
		t.Fatal(err)
	}
	var live string
	for _, section := range result.Sections {
		if section.Name == "live_terminals" {
			live = section.Content
			break
		}
	}
	if !contains(live, terminal.ProcessRef) || !contains(live, "use write_stdin") || !contains(live, "read answer") {
		t.Fatalf("live terminal context = %q", live)
	}
	history, err := service.modelVisibleSessionHistory(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	for _, message := range history {
		joined += "\n" + message.Text
	}
	if !contains(joined, terminal.ProcessRef) || !contains(joined, "Do not call exec_command") {
		t.Fatalf("model-visible history missing live terminal guidance: %s", joined)
	}
}

func TestCreateRuntimeSessionWithoutProjectPathUsesConfiguredInitialWorkspace(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	workspace := t.TempDir()
	if _, err := service.CompleteInitialization(ctx, domain.CompleteInitializationInput{InitialWorkspacePath: workspace}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, Title: "https://github.com/example/repo"})
	if err != nil {
		t.Fatal(err)
	}
	cc, err := service.GetCodingContext(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cc.ProjectPath != workspace || session.ProjectPath != "" {
		t.Fatalf("workspace paths = session %q, context %q, want %q", session.ProjectPath, cc.ProjectPath, workspace)
	}
	persisted, err := service.GetRuntimeSession(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.ProjectPath != "" {
		t.Fatalf("unscoped session persisted project path = %q", persisted.ProjectPath)
	}
	entries, err := os.ReadDir(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("initial workspace contains per-session entries: %#v", entries)
	}
}

func TestCreateCodingSessionDefaultsToAssistantAgent(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if session.AgentMode != domain.AgentModeAssistant {
		t.Fatalf("agent mode = %q, want %q", session.AgentMode, domain.AgentModeAssistant)
	}
}

func TestNewCodingSessionsInheritSavedPermissionModePreference(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	if _, err := service.UpdateModelPreferences(ctx, domain.ModelPreferencesInput{DefaultPermissionMode: domain.PermissionModeFullAccess}); err != nil {
		t.Fatal(err)
	}
	first, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	firstMode, err := service.GetPermissionMode(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if firstMode.Mode != domain.PermissionModeFullAccess {
		t.Fatalf("first mode = %q, want full access", firstMode.Mode)
	}
	if _, err := service.UpdateModelPreferences(ctx, domain.ModelPreferencesInput{DefaultPermissionMode: domain.PermissionModeRequestApproval}); err != nil {
		t.Fatal(err)
	}
	second, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	secondMode, err := service.GetPermissionMode(ctx, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if secondMode.Mode != domain.PermissionModeRequestApproval {
		t.Fatalf("second mode = %q, want request approval", secondMode.Mode)
	}
	firstMode, err = service.GetPermissionMode(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if firstMode.Mode != domain.PermissionModeFullAccess {
		t.Fatalf("existing session mode = %q, want unchanged full access", firstMode.Mode)
	}
	if _, err := service.UpdateModelPreferences(ctx, domain.ModelPreferencesInput{DefaultPermissionMode: legacyPermissionModeAutoApprove}); err == nil {
		t.Fatal("removed automatic approval mode was accepted as a default preference")
	}
}

func TestSubmitSessionMessageRecreatesDeletedInitialWorkspace(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	workspace := filepath.Join(t.TempDir(), "unscoped-workspace")
	if _, err := service.CompleteInitialization(ctx, domain.CompleteInitializationInput{InitialWorkspacePath: workspace}); err != nil {
		t.Fatal(err)
	}
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
	if before.ProjectPath != workspace {
		t.Fatalf("workspace path = %q, want %q", before.ProjectPath, workspace)
	}
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
	if info, statErr := os.Stat(after.ProjectPath); statErr != nil || !info.IsDir() {
		t.Fatalf("initial workspace was not recreated: %v", statErr)
	}
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

func TestForkSessionCopiesVisibleHistoryAndSettledToolsAtBoundary(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "Parent", ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	user, err := service.AppendEvent(ctx, domain.AppendEventRequest{SessionID: session.ID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser, Content: "before"})
	if err != nil {
		t.Fatal(err)
	}
	turn, err := service.StartTurn(ctx, domain.StartTurnRequest{SessionID: session.ID, UserEventID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveToolCall(ctx, domain.CreateToolCallRequest{ID: "fork-tool", SessionID: session.ID, TurnID: turn.ID, Name: "read_file", Status: domain.ToolCallStatusSuccess, ResultSummary: "read"}); err != nil {
		t.Fatal(err)
	}
	assistant, err := service.AppendEvent(ctx, domain.AppendEventRequest{SessionID: session.ID, TurnID: turn.ID, Type: domain.EventTypeAssistantMessage, Role: domain.EventRoleAssistant, Content: "boundary"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CompleteTurn(ctx, domain.CompleteTurnRequest{TurnID: turn.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AppendEvent(ctx, domain.AppendEventRequest{SessionID: session.ID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser, Content: "after"}); err != nil {
		t.Fatal(err)
	}

	fork, err := service.ForkSession(ctx, domain.ForkSessionRequest{SessionID: session.ID, AtEventID: assistant.ID})
	if err != nil {
		t.Fatal(err)
	}
	events, err := service.ListEvents(ctx, fork.ID, false, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Content != "before" || events[1].Content != "boundary" {
		t.Fatalf("fork events = %#v", events)
	}
	turns, err := service.ListTurns(ctx, fork.ID, 10)
	if err != nil || len(turns) != 1 || turns[0].ID == turn.ID || turns[0].UserEventID == user.ID {
		t.Fatalf("fork turns = %#v err = %v", turns, err)
	}
	tools, err := service.ListToolCalls(ctx, fork.ID)
	if err != nil || len(tools) != 1 || tools[0].ID == "fork-tool" || tools[0].Status != domain.ToolCallStatusSuccess {
		t.Fatalf("fork tools = %#v err = %v", tools, err)
	}
	state, err := service.GetSessionExecutionState(ctx, fork.ID)
	if err != nil || state.Status != domain.ExecutionStatusIdle {
		t.Fatalf("fork execution state = %#v err = %v", state, err)
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
	if captured[0].Role != "system" ||
		!contains(captured[0].Content, "<agent_instructions>") ||
		!contains(captured[0].Content, `<default name="agent_mode">`) ||
		!contains(captured[0].Content, `<global name="tool_protocol">`) {
		t.Fatalf("first message = %#v", captured[0])
	}
	if contains(captured[0].Content, "<aivo_context>") {
		t.Fatalf("runtime context mixed into agent instructions: %#v", captured[0])
	}
	joined := ""
	for _, message := range captured {
		joined += "\n" + message.Role + ": " + message.Content
	}
	if contains(joined, "You are Aivo") {
		t.Fatalf("removed fixed Aivo system prompt was injected: %s", joined)
	}
	if !contains(joined, "<aivo_context>") {
		t.Fatalf("runtime context missing from model messages: %s", joined)
	}
	if !contains(joined, "Durable architecture summary") ||
		!contains(joined, "Keep the conversation coherent") ||
		!contains(joined, "current request") {
		t.Fatalf("context missing expected content: %s", joined)
	}
	if !contains(joined, "Always run focused tests") {
		t.Fatalf("project instructions missing from context: %s", joined)
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

func TestSubmitSessionMessageSendsTextAttachmentToProvider(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	requests := make(chan []struct {
		Role    string `json:"role"`
		Content any    `json:"content"`
	}, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct {
				Role    string `json:"role"`
				Content any    `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		requests <- body.Messages
		w.Header().Set("Content-Type", "application/json")
		for _, message := range body.Messages {
			content, _ := message.Content.(string)
			if message.Role == domain.EventRoleSystem && contains(content, "Host resource-group selector") {
				_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"intent\":\"use\",\"resources\":[]}"}}]}`))
				return
			}
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"attachment ok"}}]}`))
	}))
	defer server.Close()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "custom-api", Type: "openai-compatible", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{
		Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	fileText := "\n  package main\n\nfunc main() {}  \n"
	if _, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{
		SessionID: session.ID,
		Text:      "inspect the attached source",
		Attachments: []domain.MessageAttachment{{
			Name: "main.go", MIMEType: "text/plain", Kind: "file", Text: fileText, Size: int64(len([]byte(fileText))),
		}},
	}); err != nil {
		t.Fatal(err)
	}

	for len(requests) > 0 {
		messages := <-requests
		for _, message := range messages {
			if message.Role != domain.EventRoleUser {
				continue
			}
			parts, _ := message.Content.([]any)
			for _, rawPart := range parts {
				part, _ := rawPart.(map[string]any)
				if part["type"] == "text" && part["text"] == "main.go\n"+fileText {
					return
				}
			}
		}
	}
	t.Fatal("provider messages did not contain the exact text attachment")
}

func TestAgentPromptBuilderSeparatesDefaultAndGlobalInjections(t *testing.T) {
	prompt := buildAgentSystemPrompt(domain.AgentModeDefinition{
		DisplayName: "Code",
		Prompt:      "Default mode behavior.",
	})
	if !contains(prompt, "<agent_instructions>") {
		t.Fatalf("prompt missing wrapper: %q", prompt)
	}
	if !contains(prompt, `<default name="agent_mode">`) || !contains(prompt, "Default mode behavior.") {
		t.Fatalf("prompt missing default mode injection: %q", prompt)
	}
	if !contains(prompt, `<global name="tool_protocol">`) || !contains(prompt, "stable filtered automatic tool set") || !contains(prompt, `mode "use"`) {
		t.Fatalf("prompt missing global tool protocol injection: %q", prompt)
	}
	if !contains(prompt, `mode "inspect"`) || !contains(prompt, "bounded summaries and does not activate tools") {
		t.Fatalf("agent prompt is missing temporary inspection lifetime guidance: %q", prompt)
	}
	if contains(prompt, "call the skill tool") || !contains(prompt, "call resource_resolve") {
		t.Fatalf("agent prompt is missing the replaceable selection control: %q", prompt)
	}
	if contains(prompt, "<aivo_context>") || contains(prompt, "You are Aivo") {
		t.Fatalf("agent prompt included runtime or removed fixed prompt: %q", prompt)
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
		Type:        domain.SessionTypeCoding,
		Title:       "帮我写一个 Redis 缓存方案",
		ProjectPath: t.TempDir(),
		Model:       &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5.5"},
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
