package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"aivo/core/domain"
	"aivo/core/infra/persistence"
)

func TestConversationalMCPRegistrationRequiresExactApprovalEvenInFullAccess(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	root := t.TempDir()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "Register MCP", ProjectPath: root})
	if err != nil {
		t.Fatal(err)
	}
	turn, err := service.StartTurn(ctx, domain.StartTurnRequest{SessionID: session.ID, AgentMode: domain.AgentModeAssistant})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetPermissionMode(ctx, domain.PermissionModeInput{SessionID: session.ID, Mode: domain.PermissionModeFullAccess}); err != nil {
		t.Fatal(err)
	}
	registry, _ := service.toolsForWorkspace(root)
	tool, ok := registry.Get(toolRegistrationMCPName)
	if !ok {
		t.Fatal("builtin conversational registration tool is missing")
	}
	input := domain.MCPRegistrationProposalInput{
		ID: "approval_test", DisplayName: "Approval Test", Description: "Test MCP registration approval", Transport: domain.MCPTransportStdio,
		Command: os.Args[0], Args: []string{"-test.run=TestMCPProbeHelperProcess", "--", "mcp-helper"},
	}
	engine := NewPermissionEngine(service.store)
	engine.MCPRegistrationPreflight = service.prepareMCPRegistrationPermission
	evaluation := engine.Evaluate(ctx, tool, mustJSON(input), domain.ToolExecutionContext{
		WorkspaceRoot: root, SessionID: session.ID, TurnID: turn.ID, ToolCallID: "register-approval", AgentMode: domain.AgentModeAssistant,
	})
	if evaluation.Decision != domain.PermissionDecisionAsk || evaluation.RequestID == "" {
		t.Fatalf("evaluation = %#v, want exact approval request despite full-access mode", evaluation)
	}
	items, err := service.ListMCPServers(ctx, domain.MCPServerListInput{IncludeDisabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("unapproved proposal mutated MCP persistence: %#v", items)
	}
	request, err := service.store.GetPermissionRequest(ctx, evaluation.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if request.Arguments["registrationTarget"] == "" || request.Arguments["registrationProposalId"] == "" || request.Remember {
		t.Fatalf("permission request = %#v, want exact non-rememberable registration metadata", request)
	}
	if _, err := service.DenyPermissionRequest(ctx, domain.DenyPermissionRequestInput{RequestID: request.ID, Remember: true, Reason: "no"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.commitMCPRegistrationProposal(ctx, input, domain.ToolExecutionContext{SessionID: session.ID, TurnID: turn.ID, ToolCallID: "register-approval"}); mcpRegistrationErrorCode(err) != "proposal_expired" {
		t.Fatalf("commit after denial err = %v, want discarded proposal", err)
	}
}

func TestConversationalMCPRegistrationIntentFindsHostOwnedToolLocally(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	defer service.Shutdown()
	entries, err := service.ListToolCatalog(context.Background(), domain.ToolCatalogInput{})
	if err != nil {
		t.Fatal(err)
	}
	matches := searchToolCatalog(entries, "请帮我注册一个 MCP 工具来源", 4)
	for _, match := range matches {
		if match.Name == toolRegistrationMCPName {
			return
		}
	}
	t.Fatalf("local registration intent did not select %s: %#v", toolRegistrationMCPName, matches)
}

func TestMCPAutomaticSelectionDoesNotDependOnAgentSourceListTool(t *testing.T) {
	store := &memoryProviderStore{
		mcpServers: []domain.MCPServerConfig{{
			ID: "selection-mcp", Name: "Selection MCP", Description: "Look up exact documentation for the current task", Transport: domain.MCPTransportStdio,
			Command: "not-used-from-cache", Enabled: true,
		}},
		mcpTools: map[string][]domain.MCPToolRecord{
			"selection-mcp": {{
				ID: "selection-mcp:lookup", ServerID: "selection-mcp", Name: "lookup",
				Description: "Look up exact documentation for the current task", InputSchema: map[string]any{"type": "object"},
				Capability: "mcp.read", RiskLevel: "low",
			}},
		},
	}
	service := NewService(store)
	defer service.Shutdown()
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	registry, _ := service.toolsForWorkspace(t.TempDir())
	if _, exists := registry.Get("aivo_tools_list_mcp"); exists {
		t.Fatal("removed Agent-visible MCP source-list tool is still registered")
	}
	activations, candidates := service.preCallToolCandidates(ctx, session.ID, "turn", registry, registry.Specs())
	const selectedName = "mcp_selection_mcp_lookup"
	found := false
	for _, candidate := range candidates {
		if candidate.Name == selectedName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("concrete MCP tool is absent from automatic candidates: %#v", candidates)
	}
	activations[selectedName] = "automatic"
	assembly := AssembleToolSpecsWithSources(registry, registry.Specs(), activations)
	names := toolSpecNames(assembly.Specs)
	if !containsToolNames(names, selectedName, ToolResolveName) || containsToolNames(names, "aivo_tools_list_mcp") {
		t.Fatalf("selected Provider tools = %v, want exact MCP tool without source-list executor", names)
	}
	for _, name := range names {
		if strings.HasPrefix(name, "mcp_selection_mcp_") && name != selectedName {
			t.Fatalf("unselected MCP tool %q entered Provider declarations: %v", name, names)
		}
	}
}

func TestConversationalMCPProposalDoesNotConnectBeforeApproval(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	defer server.Close()
	input := domain.MCPRegistrationProposalInput{
		ID: "remote_test", DisplayName: "Remote Test", Description: "Query the remote test service", Transport: domain.MCPTransportStreamableHTTP, URL: server.URL,
	}
	_, metadata, _, err := service.prepareMCPRegistrationPermission(context.Background(), toolRegistrationMCPName, mustJSON(input), domain.ToolExecutionContext{
		SessionID: "session", TurnID: "turn", ToolCallID: "remote-proposal",
	})
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 0 {
		t.Fatalf("proposal made %d pre-approval network requests", requests.Load())
	}
	if metadata["registrationTarget"] != server.URL || metadata["registrationGlobal"] != true {
		t.Fatalf("metadata = %#v, want exact remote target and global scope", metadata)
	}
}

func TestConversationalMCPRegistrationPersistsReadySourceForLaterConversations(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	input := domain.MCPRegistrationProposalInput{
		ID: "conversation_mcp", DisplayName: "Conversation MCP", Description: "Registered through chat",
		Transport: domain.MCPTransportStdio, Command: os.Args[0],
		Args: []string{"-test.run=TestMCPProbeHelperProcess", "--", "mcp-helper"}, TimeoutSeconds: 5, ConnectTimeoutSeconds: 5,
	}
	execCtx := domain.ToolExecutionContext{SessionID: "session-one", TurnID: "turn-one", ToolCallID: "register-success"}
	if _, _, _, err := service.prepareMCPRegistrationPermission(ctx, toolRegistrationMCPName, mustJSON(input), execCtx); err != nil {
		t.Fatal(err)
	}
	result, err := service.commitMCPRegistrationProposal(ctx, input, execCtx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != domain.MCPServerStatusReady || !registrationContainsString(result.ToolNames, "mcp_conversation_mcp_echo") {
		t.Fatalf("result = %#v, want ready discovered MCP tool", result)
	}
	items, err := service.ListMCPServers(ctx, domain.MCPServerListInput{IncludeDisabled: true, IncludeTools: true})
	if err != nil || len(items) != 1 || !items[0].Server.Enabled || items[0].Server.Status != domain.MCPServerStatusReady || len(items[0].Tools) != 1 {
		t.Fatalf("persisted items = %#v err = %v, want one ready enabled source", items, err)
	}
	registry := service.globalToolCatalogRegistry(ctx)
	if _, ok := registry.Get("mcp_conversation_mcp_echo"); !ok {
		t.Fatal("ready registered tool is not eligible in the global catalog")
	}
	later, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "Later", ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	active, err := service.GetSessionActiveTools(ctx, later.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(active.ToolNames) != 0 {
		t.Fatalf("global installation leaked into manual session activation: %#v", active.ToolNames)
	}
	if _, err := service.commitMCPRegistrationProposal(ctx, input, execCtx); mcpRegistrationErrorCode(err) != "proposal_expired" {
		t.Fatalf("replayed commit err = %v, want single-use proposal refusal", err)
	}
	if service.mcpManager.connections != nil {
		service.mcpManager.connections.Close()
	}
}

func TestConversationalMCPRegistrationFailureStaysDisabled(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	input := domain.MCPRegistrationProposalInput{
		ID: "missing_mcp", DisplayName: "Missing MCP", Description: "Exercise a missing MCP executable", Transport: domain.MCPTransportStdio,
		Command: filepathForMissingMCPExecutable(t), TimeoutSeconds: 1, ConnectTimeoutSeconds: 1,
	}
	execCtx := domain.ToolExecutionContext{SessionID: "session", TurnID: "turn", ToolCallID: "register-failure"}
	if _, _, _, err := service.prepareMCPRegistrationPermission(ctx, toolRegistrationMCPName, mustJSON(input), execCtx); err != nil {
		t.Fatal(err)
	}
	if _, err := service.commitMCPRegistrationProposal(ctx, input, execCtx); mcpRegistrationErrorCode(err) != "mcp_probe_failed" {
		t.Fatalf("commit err = %v, want probe failure", err)
	}
	items, err := service.ListMCPServers(context.Background(), domain.MCPServerListInput{IncludeDisabled: true, IncludeTools: true})
	if err != nil || len(items) != 1 || items[0].Server.Enabled || items[0].Server.Status != domain.MCPServerStatusError {
		t.Fatalf("items = %#v err = %v, want disabled error source", items, err)
	}
	registry := service.globalToolCatalogRegistry(context.Background())
	for _, entry := range registry.CatalogEntries() {
		if strings.HasPrefix(entry.Name, "mcp_missing_mcp_") {
			t.Fatalf("failed registration leaked eligible tool %q", entry.Name)
		}
	}
}

func TestConversationalMCPRegistrationReloadsAfterServiceRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	store, err := persistence.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store)
	input := domain.MCPRegistrationProposalInput{
		ID: "restart_mcp", DisplayName: "Restart MCP", Description: "Test MCP restart restoration", Transport: domain.MCPTransportStdio,
		Command: os.Args[0], Args: []string{"-test.run=TestMCPProbeHelperProcess", "--", "mcp-helper"},
		TimeoutSeconds: 5, ConnectTimeoutSeconds: 5,
	}
	execCtx := domain.ToolExecutionContext{SessionID: "session", TurnID: "turn", ToolCallID: "restart-register"}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if _, _, _, err := service.prepareMCPRegistrationPermission(ctx, toolRegistrationMCPName, mustJSON(input), execCtx); err != nil {
		t.Fatal(err)
	}
	if _, err := service.commitMCPRegistrationProposal(ctx, input, execCtx); err != nil {
		t.Fatal(err)
	}
	service.Shutdown()
	if service.mcpManager.connections != nil {
		service.mcpManager.connections.Close()
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := persistence.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	restarted := NewService(reopened)
	defer restarted.Shutdown()
	registry := restarted.globalToolCatalogRegistry(context.Background())
	if _, ok := registry.Get("mcp_restart_mcp_echo"); !ok {
		t.Fatal("restarted service did not reload the globally enabled MCP tool")
	}
}

func TestConversationalMCPRegistrationHasOneConcurrentWinner(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	input := domain.MCPRegistrationProposalInput{
		ID: "race_mcp", DisplayName: "Race MCP", Description: "Test concurrent MCP registration", Transport: domain.MCPTransportStdio,
		Command: os.Args[0], Args: []string{"-test.run=TestMCPProbeHelperProcess", "--", "mcp-helper"},
		TimeoutSeconds: 5, ConnectTimeoutSeconds: 5,
	}
	contexts := []domain.ToolExecutionContext{
		{SessionID: "session-one", TurnID: "turn-one", ToolCallID: "race-one"},
		{SessionID: "session-two", TurnID: "turn-two", ToolCallID: "race-two"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	for _, execCtx := range contexts {
		if _, _, _, err := service.prepareMCPRegistrationPermission(ctx, toolRegistrationMCPName, mustJSON(input), execCtx); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	errs := make([]error, len(contexts))
	for index, execCtx := range contexts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[index] = service.commitMCPRegistrationProposal(ctx, input, execCtx)
		}()
	}
	wg.Wait()
	successes := 0
	conflicts := 0
	for _, err := range errs {
		if err == nil {
			successes++
		} else if mcpRegistrationErrorCode(err) == "source_conflict" {
			conflicts++
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent results = %#v, want one success and one source conflict", errs)
	}
	if service.mcpManager.connections != nil {
		service.mcpManager.connections.Close()
	}
}

func TestConversationalMCPRegistrationRejectsUnsafeConfiguration(t *testing.T) {
	tests := []struct {
		name  string
		input domain.MCPRegistrationProposalInput
		code  string
	}{
		{name: "oversized description", input: domain.MCPRegistrationProposalInput{ID: "bad", DisplayName: "Bad", Description: strings.Repeat("a", 501), Transport: domain.MCPTransportStdio, Command: "npx"}, code: "invalid_description"},
		{name: "shell command", input: domain.MCPRegistrationProposalInput{ID: "bad", DisplayName: "Bad", Description: "Bad MCP", Transport: domain.MCPTransportStdio, Command: "npx --yes bad"}, code: "invalid_command"},
		{name: "raw secret", input: domain.MCPRegistrationProposalInput{ID: "bad", DisplayName: "Bad", Description: "Bad MCP", Transport: domain.MCPTransportStdio, Command: "npx", Args: []string{"--token=secret"}}, code: "raw_secret_refused"},
		{name: "credential URL", input: domain.MCPRegistrationProposalInput{ID: "bad", DisplayName: "Bad", Description: "Bad MCP", Transport: domain.MCPTransportStreamableHTTP, URL: "https://user:secret@example.com/mcp"}, code: "invalid_url"},
		{name: "remote HTTP", input: domain.MCPRegistrationProposalInput{ID: "bad", DisplayName: "Bad", Description: "Bad MCP", Transport: domain.MCPTransportStreamableHTTP, URL: "http://example.com/mcp"}, code: "insecure_url"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := normalizeMCPRegistrationInput(test.input)
			if mcpRegistrationErrorCode(err) != test.code {
				t.Fatalf("err = %v code = %q, want %q", err, mcpRegistrationErrorCode(err), test.code)
			}
		})
	}
}

func TestConversationalMCPRegistrationAllowsBlankDescription(t *testing.T) {
	input, err := normalizeMCPRegistrationInput(domain.MCPRegistrationProposalInput{
		ID: "blank_description", DisplayName: "Blank Description",
		Transport: domain.MCPTransportStdio, Command: "npx",
	})
	if err != nil || input.Description != "" {
		t.Fatalf("normalized input = %#v, err = %v", input, err)
	}
}

func registrationContainsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func filepathForMissingMCPExecutable(t *testing.T) string {
	t.Helper()
	return t.TempDir() + string(os.PathSeparator) + "missing-mcp-executable"
}
