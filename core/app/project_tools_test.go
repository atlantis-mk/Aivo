package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aivo/core/domain"
)

func TestBuiltinProjectExtensionPreservesFourCorePrimitives(t *testing.T) {
	root := t.TempDir()
	coreRegistry, err := NewCodingToolRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	coreNames := map[string]bool{}
	for _, spec := range AssembleToolSpecs(coreRegistry, coreRegistry.Specs()).Specs {
		coreNames[spec.Name] = true
	}
	if len(coreNames) != 4 {
		t.Fatalf("core registry names = %#v, want exactly four primitives", coreNames)
	}
	for _, name := range []string{"read", "bash", "edit", "write"} {
		if !coreNames[name] {
			t.Fatalf("core registry missing %q: %#v", name, coreNames)
		}
	}
	if coreNames["search_projects"] {
		t.Fatal("legacy search_projects remained in the core registry")
	}

	service, cleanup := newSessionTestService(t)
	defer cleanup()
	registry, _ := service.toolsForWorkspace(root)
	activation := map[string]string{}
	for _, name := range []string{projectQueryToolName, projectAddToolName, projectAssociateToolName} {
		entry, ok := catalogEntryNamed(registry.CatalogEntries(), name)
		if !ok {
			t.Fatalf("builtin project tool %q was not registered", name)
		}
		if entry.Source != domain.ToolSourceExtension || entry.SourceID != projectExtensionID || entry.ActivationPolicy != "auto" || entry.ImplementationHash == "" {
			t.Fatalf("builtin project tool identity = %#v", entry)
		}
		activation[name] = "automatic"
	}
	assembly := AssembleToolSpecsWithSources(registry, registry.Specs(), activation)
	for _, name := range []string{projectQueryToolName, projectAddToolName, projectAssociateToolName} {
		identity, ok := assembly.ExpectedRegistrations[name]
		if !ok || identity.SourceID != projectExtensionID || identity.RegistrationID == "" {
			t.Fatalf("frozen registration for %q = %#v", name, identity)
		}
	}
}

func TestAgentProjectRegistrationQueryAndImmutableAssociation(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	initial := t.TempDir()
	if _, err := service.CompleteInitialization(ctx, domain.CompleteInitializationInput{InitialWorkspacePath: initial}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "unscoped"})
	if err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	registered, err := service.RegisterAgentProject(ctx, target)
	if err != nil || registered.Status != domain.ProjectRegistrationCreated {
		t.Fatalf("registration = %#v, err = %v", registered, err)
	}
	query, err := service.QueryAgentProjects(ctx, session.ID, domain.ProjectQueryInput{ProjectID: registered.Project.ID})
	if err != nil || len(query.Projects) != 1 || query.CurrentProject != nil {
		t.Fatalf("query = %#v, err = %v", query, err)
	}
	if _, err := service.AssociateAgentProject(ctx, session.ID, registered.Project.ID, initial); projectErrorCode(err, "") != "project_reference_mismatch" {
		t.Fatalf("stale path error = %v", err)
	}

	var updated *domain.Session
	service.SetSessionUpdatedHook(func(updatedID string, value *domain.Session) {
		if updatedID == session.ID && value != nil {
			copy := *value
			updated = &copy
		}
	})
	bound, err := service.AssociateAgentProject(ctx, session.ID, registered.Project.ID, registered.Project.RootPath)
	if err != nil || !bound.Changed || bound.Session.ProjectPath != target {
		t.Fatalf("binding = %#v, err = %v", bound, err)
	}
	if updated == nil || updated.ProjectPath != target {
		t.Fatalf("session.updated did not include the full bound session: %#v", updated)
	}
	cc, err := service.GetCodingContext(ctx, session.ID)
	if err != nil || cc.ProjectPath != target || cc.CWD != target {
		t.Fatalf("coding context = %#v, err = %v", cc, err)
	}
	retry, err := service.AssociateAgentProject(ctx, session.ID, registered.Project.ID, registered.Project.RootPath)
	if err != nil || retry.Changed || retry.Conflict {
		t.Fatalf("idempotent retry = %#v, err = %v", retry, err)
	}
	other, err := service.RegisterAgentProject(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AssociateAgentProject(ctx, session.ID, other.Project.ID, other.Project.RootPath); projectErrorCode(err, "") != "project_already_bound" {
		t.Fatalf("different-project binding error = %v", err)
	}
	if _, err := service.SetProjectSidebarHidden(ctx, registered.Project.RootPath, true); err != nil {
		t.Fatal(err)
	}
	hiddenQuery, err := service.QueryAgentProjects(ctx, session.ID, domain.ProjectQueryInput{})
	if err != nil || hiddenQuery.CurrentProject == nil || hiddenQuery.CurrentProject.ID != registered.Project.ID {
		t.Fatalf("hidden current project query = %#v, err = %v", hiddenQuery, err)
	}
	hiddenRetry, err := service.AssociateAgentProject(ctx, session.ID, registered.Project.ID, registered.Project.RootPath)
	if err != nil || hiddenRetry.Changed {
		t.Fatalf("hidden current-project retry = %#v, err = %v", hiddenRetry, err)
	}
}

func TestAgentProjectAssociationRejectsInvalidSessionWorkspaceAndBusyTerminal(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	initial := t.TempDir()
	if _, err := service.CompleteInitialization(ctx, domain.CompleteInitializationInput{InitialWorkspacePath: initial}); err != nil {
		t.Fatal(err)
	}
	project, err := service.RegisterAgentProject(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	generic, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeGeneric, Title: "generic"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AssociateAgentProject(ctx, generic.ID, project.Project.ID, project.Project.RootPath); projectErrorCode(err, "") != "coding_session_required" {
		t.Fatalf("generic session error = %v", err)
	}

	specialized, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "specialized"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateOrUpdateCodingContext(ctx, specialized.ID, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AssociateAgentProject(ctx, specialized.ID, project.Project.ID, project.Project.RootPath); projectErrorCode(err, "") != "workspace_specialized" {
		t.Fatalf("specialized workspace error = %v", err)
	}

	busy, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "busy"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.ptyManager.Start(ctx, SandboxRequest{
		WorkspaceRoot: initial, CWD: initial, SessionID: busy.ID, Command: "sleep 30", EnvAllowlist: defaultEnvAllowlist(),
	}, 24, 80, 50*time.Millisecond, 4096)
	if err != nil {
		t.Skipf("pty unavailable: %v", err)
	}
	defer service.ptyManager.CleanupSession(busy.ID)
	if _, err := service.AssociateAgentProject(ctx, busy.ID, project.Project.ID, project.Project.RootPath); projectErrorCode(err, "") != "workspace_busy" {
		t.Fatalf("busy workspace error = %v", err)
	}
}

func TestAgentProjectPathValidationUsesStableErrors(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	for _, test := range []struct {
		name string
		path string
		code string
	}{
		{name: "relative", path: "relative/project", code: "absolute_path_required"},
		{name: "missing", path: filepath.Join(t.TempDir(), "missing"), code: "project_path_not_found"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.RegisterAgentProject(ctx, test.path)
			if projectErrorCode(err, "") != test.code {
				t.Fatalf("error = %v, want %s", err, test.code)
			}
		})
	}
	file := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(file, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RegisterAgentProject(ctx, file); projectErrorCode(err, "") != "project_not_directory" {
		t.Fatalf("file path error = %v", err)
	}
	initial := t.TempDir()
	if _, err := service.CompleteInitialization(ctx, domain.CompleteInitializationInput{InitialWorkspacePath: initial}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "errors"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AssociateAgentProject(ctx, "", "missing", initial); projectErrorCode(err, "") != "session_required" {
		t.Fatalf("missing session error = %v", err)
	}
	if _, err := service.AssociateAgentProject(ctx, session.ID, "missing", initial); projectErrorCode(err, "") != "project_not_found" {
		t.Fatalf("missing project error = %v", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := service.AssociateAgentProject(cancelled, session.ID, "missing", initial); projectErrorCode(err, "") != "cancelled" {
		t.Fatalf("cancelled association error = %v", err)
	}
	if _, err := service.QueryAgentProjects(ctx, session.ID, domain.ProjectQueryInput{ProjectID: "id", Limit: 1}); projectErrorCode(err, "") != "invalid_arguments" {
		t.Fatalf("mutually exclusive query error = %v", err)
	}
}

func TestProjectToolPermissionMatrixAndExactRememberScope(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	initial := t.TempDir()
	if _, err := service.CompleteInitialization(ctx, domain.CompleteInitializationInput{InitialWorkspacePath: initial}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "permissions"})
	if err != nil {
		t.Fatal(err)
	}
	firstRoot := t.TempDir()
	secondRoot := t.TempDir()
	first, err := service.RegisterAgentProject(ctx, firstRoot)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.RegisterAgentProject(ctx, secondRoot)
	if err != nil {
		t.Fatal(err)
	}
	registry, _ := service.toolsForWorkspace(initial)
	queryTool, _ := registry.Get(projectQueryToolName)
	addTool, _ := registry.Get(projectAddToolName)
	associateTool, _ := registry.Get(projectAssociateToolName)
	engine := NewPermissionEngine(service.store)
	engine.ProjectPreflight = service.prepareProjectPermission
	execCtx := domain.ToolExecutionContext{WorkspaceRoot: initial, SessionID: session.ID, TurnID: "turn", AgentMode: domain.AgentModeCode}

	if evaluation := engine.Evaluate(ctx, queryTool, json.RawMessage(`{}`), execCtx); evaluation.Decision != domain.PermissionDecisionAllow {
		t.Fatalf("query permission = %#v", evaluation)
	}
	firstAddArgs := mustJSONRaw(t, map[string]any{"rootPath": firstRoot})
	evaluation := engine.Evaluate(ctx, addTool, firstAddArgs, execCtx)
	if evaluation.Decision != domain.PermissionDecisionAsk || evaluation.RequestID == "" {
		t.Fatalf("default add permission = %#v", evaluation)
	}
	request, err := service.store.GetPermissionRequest(ctx, evaluation.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if request.Arguments["projectRoot"] != firstRoot || request.Arguments["projectOperation"] != "add" || len(request.Paths) != 1 || strings.Contains(request.Paths[0], firstRoot) {
		t.Fatalf("project permission request is not displayable and opaque: %#v", request)
	}
	if _, err := service.ApprovePermissionRequest(ctx, domain.ApprovePermissionRequestInput{RequestID: request.ID, Remember: true}); err != nil {
		t.Fatal(err)
	}
	if remembered := engine.Evaluate(ctx, addTool, firstAddArgs, execCtx); remembered.Decision != domain.PermissionDecisionAllow {
		t.Fatalf("exact remembered add = %#v", remembered)
	}
	secondAddArgs := mustJSONRaw(t, map[string]any{"rootPath": secondRoot})
	if other := engine.Evaluate(ctx, addTool, secondAddArgs, execCtx); other.Decision != domain.PermissionDecisionAsk {
		t.Fatalf("remembered add leaked to another project: %#v", other)
	}

	associateArgs := mustJSONRaw(t, map[string]any{"projectId": first.Project.ID, "rootPath": first.Project.RootPath})
	associateEvaluation := engine.Evaluate(ctx, associateTool, associateArgs, execCtx)
	if associateEvaluation.Decision != domain.PermissionDecisionAsk {
		t.Fatalf("default associate permission = %#v", associateEvaluation)
	}
	if _, err := service.ApprovePermissionRequest(ctx, domain.ApprovePermissionRequestInput{RequestID: associateEvaluation.RequestID, Remember: true}); err != nil {
		t.Fatal(err)
	}
	if remembered := engine.Evaluate(ctx, associateTool, associateArgs, execCtx); remembered.Decision != domain.PermissionDecisionAllow {
		t.Fatalf("exact remembered association = %#v", remembered)
	}
	otherAssociateArgs := mustJSONRaw(t, map[string]any{"projectId": second.Project.ID, "rootPath": second.Project.RootPath})
	if other := engine.Evaluate(ctx, associateTool, otherAssociateArgs, execCtx); other.Decision != domain.PermissionDecisionAsk {
		t.Fatalf("remembered association leaked to another project: %#v", other)
	}
	if _, err := service.AssociateAgentProject(ctx, session.ID, first.Project.ID, first.Project.RootPath); err != nil {
		t.Fatal(err)
	}
	if same := engine.Evaluate(ctx, associateTool, associateArgs, execCtx); same.Decision != domain.PermissionDecisionAllow {
		t.Fatalf("idempotent association requested approval: %#v", same)
	}
	if conflict := engine.Evaluate(ctx, associateTool, otherAssociateArgs, execCtx); conflict.Decision != domain.PermissionDecisionDeny || conflict.Code != "project_already_bound" {
		t.Fatalf("bound-session conflict permission = %#v", conflict)
	}

	rejectSession, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "reject"})
	if err != nil {
		t.Fatal(err)
	}
	rejectRoot := t.TempDir()
	rejectCtx := execCtx
	rejectCtx.SessionID = rejectSession.ID
	rejectEvaluation := engine.Evaluate(ctx, addTool, mustJSONRaw(t, map[string]any{"rootPath": rejectRoot}), rejectCtx)
	if rejectEvaluation.Decision != domain.PermissionDecisionAsk {
		t.Fatalf("rejection setup = %#v", rejectEvaluation)
	}
	if _, err := service.DenyPermissionRequest(ctx, domain.DenyPermissionRequestInput{RequestID: rejectEvaluation.RequestID, Remember: true, Reason: "test rejection"}); err != nil {
		t.Fatal(err)
	}
	if rejected := engine.Evaluate(ctx, addTool, mustJSONRaw(t, map[string]any{"rootPath": rejectRoot}), rejectCtx); rejected.Decision != domain.PermissionDecisionDeny {
		t.Fatalf("remembered rejection = %#v", rejected)
	}

	legacySession, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "legacy auto"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetPermissionMode(ctx, domain.PermissionModeInput{SessionID: legacySession.ID, Mode: legacyPermissionModeAutoApprove}); err == nil {
		t.Fatal("removed auto-approve mode was accepted")
	}
	if _, err := service.store.SavePermissionRule(ctx, domain.PermissionRule{
		Scope: permissionModeScopePrefix + legacyPermissionModeAutoApprove, SessionID: legacySession.ID,
		WorkspaceRoot: initial, ToolName: permissionRuleWildcard, Action: permissionActionWrite,
		Decision: domain.PermissionDecisionAllow, Paths: []string{permissionRuleWildcard},
	}); err != nil {
		t.Fatal(err)
	}
	legacyMode, err := service.GetPermissionMode(ctx, legacySession.ID)
	if err != nil {
		t.Fatal(err)
	}
	if legacyMode.Mode != domain.PermissionModeRequestApproval {
		t.Fatalf("legacy auto-approve mode = %q, want request approval", legacyMode.Mode)
	}
	legacyCtx := execCtx
	legacyCtx.SessionID = legacySession.ID
	if got := engine.Evaluate(ctx, addTool, firstAddArgs, legacyCtx); got.Decision != domain.PermissionDecisionAsk {
		t.Fatalf("legacy auto-approve permission = %#v, want approval request", got)
	}

	fullSession, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "full"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetPermissionMode(ctx, domain.PermissionModeInput{SessionID: fullSession.ID, Mode: domain.PermissionModeFullAccess}); err != nil {
		t.Fatal(err)
	}
	fullCtx := execCtx
	fullCtx.SessionID = fullSession.ID
	if got := engine.Evaluate(ctx, associateTool, associateArgs, fullCtx); got.Decision != domain.PermissionDecisionAllow {
		t.Fatalf("full-access associate = %#v", got)
	}

	for _, restricted := range []domain.ToolExecutionContext{
		{WorkspaceRoot: initial, SessionID: session.ID, AgentMode: domain.AgentModePlan},
		{WorkspaceRoot: initial, SessionID: session.ID, AgentMode: domain.AgentModeCode, PermissionScope: "read_only"},
	} {
		if got := engine.Evaluate(ctx, addTool, firstAddArgs, restricted); got.Decision != domain.PermissionDecisionDeny {
			t.Fatalf("restricted add = %#v", got)
		}
		if got := engine.Evaluate(ctx, queryTool, json.RawMessage(`{}`), restricted); got.Decision != domain.PermissionDecisionAllow {
			t.Fatalf("restricted query = %#v", got)
		}
	}
}

func TestAgentLoopReloadsWorkspaceAfterProjectAssociation(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	initial := t.TempDir()
	if _, err := service.CompleteInitialization(ctx, domain.CompleteInitializationInput{InitialWorkspacePath: initial}); err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "workspace-marker.txt"), []byte("TARGET_WORKSPACE_RELOADED"), 0o600); err != nil {
		t.Fatal(err)
	}
	project, err := service.RegisterAgentProject(ctx, target)
	if err != nil {
		t.Fatal(err)
	}

	primaryRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
			Tools []struct {
				Function struct {
					Name string `json:"name"`
				} `json:"function"`
			} `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		if len(body.Tools) == 0 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"tools\":[\"aivo_projects_associate\"],\"reason\":\"the task asks to bind this session to a project\"}"}}]}`))
			return
		}
		primaryRequests++
		switch primaryRequests {
		case 1:
			canonicalName := ""
			for _, tool := range body.Tools {
				if tool.Function.Name == projectAssociateToolName {
					canonicalName = tool.Function.Name
					break
				}
			}
			if canonicalName == "" {
				t.Error("associate tool was not advertised")
			}
			arguments := string(mustJSONRaw(t, map[string]any{"projectId": project.Project.ID, "rootPath": project.Project.RootPath}))
			_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"tool_calls": []any{map[string]any{"id": "call_associate", "type": "function", "function": map[string]any{"name": canonicalName, "arguments": arguments}}}}}}})
		case 2:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"tool_calls":[{"id":"call_read_marker","type":"function","function":{"name":"read","arguments":"{\"path\":\"workspace-marker.txt\"}"}}]}}]}`))
		default:
			joined := ""
			for _, message := range body.Messages {
				joined += message.Content
			}
			if !strings.Contains(joined, "TARGET_WORKSPACE_RELOADED") {
				t.Errorf("next model request did not contain target workspace read result: %s", joined)
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"workspace reloaded"}}]}`))
		}
	}))
	defer server.Close()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "custom-api", Type: "openai-compatible", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "reload"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetPermissionMode(ctx, domain.PermissionModeInput{SessionID: session.ID, Mode: domain.PermissionModeFullAccess}); err != nil {
		t.Fatal(err)
	}
	result, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "Bind this session and inspect the marker."})
	if err != nil {
		t.Fatal(err)
	}
	if result.AssistantEvent == nil || result.AssistantEvent.Content != "workspace reloaded" || primaryRequests != 3 {
		t.Fatalf("result = %#v, primary requests = %d", result, primaryRequests)
	}
	cc, err := service.GetCodingContext(ctx, session.ID)
	if err != nil || cc.ProjectPath != target {
		t.Fatalf("final coding context = %#v, err = %v", cc, err)
	}
}

func catalogEntryNamed(entries []domain.ToolCatalogEntry, name string) (domain.ToolCatalogEntry, bool) {
	for _, entry := range entries {
		if entry.Name == name {
			return entry, true
		}
	}
	return domain.ToolCatalogEntry{}, false
}

func mustJSONRaw(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
