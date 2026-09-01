package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"aivo/core/domain"
)

const (
	permissionActionRead  = "read"
	permissionActionSkill = "skill"
	permissionActionWrite = "write"
	permissionActionShell = "shell"
	permissionActionTest  = "test"

	permissionModeScopePrefix       = "permission_mode:"
	legacyPermissionModeAutoApprove = "auto_approve"
	permissionRuleWildcard          = "*"
	permissionApprovalKeyPath       = "approvalKey:"
)

type PermissionStore interface {
	CreatePermissionRequest(context.Context, domain.PermissionRequest) (domain.PermissionRequest, error)
	GetPermissionRequest(context.Context, string) (domain.PermissionRequest, error)
	ListPermissionRequests(context.Context, string, string) ([]domain.PermissionRequest, error)
	UpdatePermissionRequest(context.Context, string, string, bool, string) (domain.PermissionRequest, error)
	SavePermissionRule(context.Context, domain.PermissionRule) (domain.PermissionRule, error)
	ListPermissionRules(context.Context, string, string) ([]domain.PermissionRule, error)
}

type PermissionEvaluation struct {
	Decision  string
	RequestID string
	Reason    string
	Code      string
}

type PermissionEngine struct {
	store                    PermissionStore
	now                      func() time.Time
	onRequest                func(domain.PermissionRequest)
	notifier                 *permissionNotifier
	ProjectPreflight         func(context.Context, string, json.RawMessage, domain.ToolExecutionContext) ([]string, map[string]any, bool, error)
	MCPRegistrationPreflight func(context.Context, string, json.RawMessage, domain.ToolExecutionContext) ([]string, map[string]any, bool, error)
	PTYRegistry              *AgentPTYRegistry
}

func NewPermissionEngine(store PermissionStore) *PermissionEngine {
	return &PermissionEngine{store: store, now: time.Now}
}

func (e *PermissionEngine) Evaluate(ctx context.Context, tool domain.Tool, args json.RawMessage, execCtx domain.ToolExecutionContext) PermissionEvaluation {
	if tool == nil {
		return PermissionEvaluation{Decision: domain.PermissionDecisionDeny, Reason: "tool is not configured"}
	}
	spec := tool.Spec()
	if permissionExemptTool(spec) {
		return PermissionEvaluation{Decision: domain.PermissionDecisionAllow}
	}
	action := permissionActionForSpec(spec)
	if evaluation := modePermissionDecision(execCtx.AgentMode, spec, action); evaluation.Decision != "" {
		return evaluation
	}
	if evaluation := scopePermissionDecision(execCtx.PermissionScope, spec, action); evaluation.Decision != "" {
		return evaluation
	}
	if action == permissionActionRead || action == permissionActionSkill {
		return PermissionEvaluation{Decision: domain.PermissionDecisionAllow}
	}
	var paths []string
	var metadata map[string]any
	var idempotent bool
	var err error
	if e.ProjectPreflight != nil && (spec.Name == projectAddToolName || spec.Name == projectAssociateToolName) {
		paths, metadata, idempotent, err = e.ProjectPreflight(ctx, spec.Name, args, execCtx)
	} else if e.MCPRegistrationPreflight != nil && isExactRegistrationToolName(spec.Name) {
		paths, metadata, idempotent, err = e.MCPRegistrationPreflight(ctx, spec.Name, args, execCtx)
	} else {
		paths, metadata, err = e.permissionPathsForTool(spec.Name, args, execCtx)
	}
	if err != nil {
		code := projectErrorCode(err, "permission_denied")
		if spec.Name == toolRegistrationMCPName {
			code = mcpRegistrationErrorCode(err)
		} else if spec.Name == toolRegistrationResourceName {
			code = resourceRegistrationErrorCode(err)
		}
		return PermissionEvaluation{Decision: domain.PermissionDecisionDeny, Reason: err.Error(), Code: code}
	}
	if idempotent {
		return PermissionEvaluation{Decision: domain.PermissionDecisionAllow}
	}
	if reason := deniedPathReason(paths); reason != "" {
		return PermissionEvaluation{Decision: domain.PermissionDecisionDeny, Reason: reason}
	}
	if e == nil || e.store == nil {
		return PermissionEvaluation{Decision: domain.PermissionDecisionDeny, Reason: "permission store is not configured"}
	}
	if requiresExactNativeConfirmation(spec) {
		if e.currentPermissionMode(ctx, execCtx) == domain.PermissionModeFullAccess {
			return PermissionEvaluation{Decision: domain.PermissionDecisionAllow}
		}
	} else if decision := e.savedDecision(ctx, execCtx, spec.Name, action, paths, metadata); decision != "" {
		return PermissionEvaluation{Decision: decision}
	}
	var arguments map[string]any
	_ = json.Unmarshal(args, &arguments)
	if arguments == nil {
		arguments = map[string]any{}
	}
	sanitizePermissionArguments(spec.Name, arguments)
	for key, value := range metadata {
		arguments[key] = value
	}
	if strings.TrimSpace(execCtx.AgentMode) != "" {
		arguments["agentMode"] = execCtx.AgentMode
	}
	if len(spec.Toolsets) > 0 {
		arguments["toolsets"] = spec.Toolsets
	}
	now := domain.NowString(e.now())
	request, err := e.store.CreatePermissionRequest(ctx, domain.PermissionRequest{
		ID: uuid.NewString(), SessionID: execCtx.SessionID, TurnID: execCtx.TurnID, ToolCallID: execCtx.ToolCallID,
		ToolName: spec.Name, Action: action, Paths: paths, Arguments: arguments,
		Status: domain.PermissionRequestStatusPending, TimeCreated: now, TimeUpdated: now,
	})
	if err != nil {
		return PermissionEvaluation{Decision: domain.PermissionDecisionDeny, Reason: err.Error()}
	}
	if e.notifier == nil {
		return PermissionEvaluation{Decision: domain.PermissionDecisionAsk, RequestID: request.ID, Reason: "permission approval is required"}
	}
	decisionCh := e.notifier.watch(request.ID)
	defer e.notifier.forget(request.ID, decisionCh)
	if e.onRequest != nil {
		e.onRequest(request)
	}
	return e.waitForDecision(ctx, request.ID, decisionCh)
}

func sanitizePermissionArguments(toolName string, arguments map[string]any) {
	if arguments == nil {
		return
	}
	switch toolName {
	case WriteStdinToolName:
		value, present := arguments["chars"]
		delete(arguments, "chars")
		text, _ := value.(string)
		pressEnter, _ := arguments["press_enter"].(bool)
		arguments["stdinPresent"] = (present && text != "") || pressEnter
	case ExecCommandToolName:
		if value, present := arguments["stdin"]; present {
			text, _ := value.(string)
			arguments["stdinPresent"] = text != ""
			delete(arguments, "stdin")
		}
	case toolRegistrationResourceName:
		files, _ := arguments["files"].([]any)
		delete(arguments, "files")
		if len(files) > 0 {
			arguments["filesProvided"] = true
			arguments["fileCount"] = len(files)
		}
	}
}

func requiresExactNativeConfirmation(spec domain.ToolSpec) bool {
	return isExactRegistrationToolName(spec.Name)
}

func isExactRegistrationToolName(name string) bool {
	return name == toolRegistrationMCPName || name == toolRegistrationResourceName
}

func (e *PermissionEngine) waitForDecision(ctx context.Context, requestID string, decisionCh <-chan struct{}) PermissionEvaluation {
	for {
		select {
		case <-ctx.Done():
			reason := "permission request cancelled with its turn"
			if e != nil && e.store != nil {
				_, _ = e.store.UpdatePermissionRequest(context.Background(), requestID, domain.PermissionRequestStatusDenied, false, reason)
			}
			return PermissionEvaluation{Decision: domain.PermissionDecisionDeny, RequestID: requestID, Reason: reason}
		case <-decisionCh:
			request, err := e.store.GetPermissionRequest(ctx, requestID)
			if err != nil {
				return PermissionEvaluation{Decision: domain.PermissionDecisionDeny, RequestID: requestID, Reason: err.Error()}
			}
			switch request.Status {
			case domain.PermissionRequestStatusApproved:
				return PermissionEvaluation{Decision: domain.PermissionDecisionAllow, RequestID: requestID}
			case domain.PermissionRequestStatusDenied:
				return PermissionEvaluation{Decision: domain.PermissionDecisionDeny, RequestID: requestID, Reason: firstNonEmpty(request.Reason, "permission denied")}
			}
		}
	}
}

func (e *PermissionEngine) savedDecision(ctx context.Context, execCtx domain.ToolExecutionContext, toolName string, action string, paths []string, metadata map[string]any) string {
	rules, err := e.store.ListPermissionRules(ctx, execCtx.WorkspaceRoot, execCtx.SessionID)
	if err != nil {
		return ""
	}
	for _, rule := range rules {
		if isPermissionModeRule(rule) || !permissionRuleApplies(rule, toolName, action) {
			continue
		}
		if permissionRuleMatches(rule, action, paths, metadata) && rule.Decision == domain.PermissionDecisionDeny {
			return rule.Decision
		}
	}
	if latestPermissionModeIsLegacyAutoApprove(rules) {
		return ""
	}
	for _, rule := range rules {
		if isPermissionModeRule(rule) || !permissionRuleApplies(rule, toolName, action) {
			continue
		}
		if permissionRuleMatches(rule, action, paths, metadata) {
			if rule.Decision == domain.PermissionDecisionAllow || rule.Decision == domain.PermissionDecisionDeny {
				return rule.Decision
			}
		}
	}
	switch latestPermissionMode(rules) {
	case domain.PermissionModeRequestApproval:
		return ""
	case domain.PermissionModeFullAccess:
		// Commands that violate non-bypassable safety checks are rejected while
		// preparing their execution context. Reaching this point means the tool is
		// allowed to run, so full access must not turn an unclassified shell command
		// back into an approval request.
		return domain.PermissionDecisionAllow
	}
	return ""
}

func (e *PermissionEngine) currentPermissionMode(ctx context.Context, execCtx domain.ToolExecutionContext) string {
	if e == nil || e.store == nil {
		return ""
	}
	rules, err := e.store.ListPermissionRules(ctx, execCtx.WorkspaceRoot, execCtx.SessionID)
	if err != nil {
		return ""
	}
	if latestPermissionModeIsLegacyAutoApprove(rules) {
		return domain.PermissionModeRequestApproval
	}
	return latestPermissionMode(rules)
}

func latestPermissionMode(rules []domain.PermissionRule) string {
	for _, rule := range rules {
		if mode := permissionModeFromRule(rule); mode != "" {
			return mode
		}
	}
	return ""
}

func latestPermissionModeIsLegacyAutoApprove(rules []domain.PermissionRule) bool {
	for _, rule := range rules {
		if !strings.HasPrefix(rule.Scope, permissionModeScopePrefix) {
			continue
		}
		rawMode := strings.TrimSpace(strings.TrimPrefix(rule.Scope, permissionModeScopePrefix))
		if rawMode == legacyPermissionModeAutoApprove {
			return true
		}
		if permissionModeFromRule(rule) != "" {
			return false
		}
	}
	return false
}

func (s *Service) ListPermissionRequests(ctx context.Context, sessionID string, status string) ([]domain.PermissionRequest, error) {
	if s.store == nil {
		return nil, errors.New("store is not configured")
	}
	return s.store.ListPermissionRequests(ctx, sessionID, status)
}

func (s *Service) ApprovePermissionRequest(ctx context.Context, input domain.ApprovePermissionRequestInput) (domain.PermissionRequest, error) {
	if strings.TrimSpace(input.RequestID) == "" {
		return domain.PermissionRequest{}, errors.New("requestId is required")
	}
	current, err := s.store.GetPermissionRequest(ctx, input.RequestID)
	if err != nil {
		return domain.PermissionRequest{}, err
	}
	if current.Status != domain.PermissionRequestStatusPending {
		return current, nil
	}
	if isExactRegistrationToolName(current.ToolName) {
		input.Remember = false
	}
	request, err := s.store.UpdatePermissionRequest(ctx, input.RequestID, domain.PermissionRequestStatusApproved, input.Remember, "")
	if err != nil {
		return domain.PermissionRequest{}, err
	}
	if request.Status != domain.PermissionRequestStatusApproved {
		return request, nil
	}
	if input.Remember {
		_, _ = s.store.SavePermissionRule(ctx, s.permissionRuleFromRequest(ctx, request, domain.PermissionDecisionAllow))
	}
	if s.permissionNotifier != nil {
		s.permissionNotifier.resolve(request.ID)
	}
	if s.onPermissionResolved != nil {
		s.onPermissionResolved(request)
	}
	if s.onSessionUpdated != nil && request.SessionID != "" {
		s.onSessionUpdated(request.SessionID, nil)
	}
	return request, nil
}

func (s *Service) DenyPermissionRequest(ctx context.Context, input domain.DenyPermissionRequestInput) (domain.PermissionRequest, error) {
	if strings.TrimSpace(input.RequestID) == "" {
		return domain.PermissionRequest{}, errors.New("requestId is required")
	}
	current, err := s.store.GetPermissionRequest(ctx, input.RequestID)
	if err != nil {
		return domain.PermissionRequest{}, err
	}
	if current.Status != domain.PermissionRequestStatusPending {
		return current, nil
	}
	if isExactRegistrationToolName(current.ToolName) {
		input.Remember = false
		if s.mcpRegistrationProposals != nil {
			s.mcpRegistrationProposals.discard(current.ToolCallID)
		}
		if s.resourceRegistrationProposals != nil {
			s.resourceRegistrationProposals.discard(current.ToolCallID)
		}
	}
	request, err := s.store.UpdatePermissionRequest(ctx, input.RequestID, domain.PermissionRequestStatusDenied, input.Remember, input.Reason)
	if err != nil {
		return domain.PermissionRequest{}, err
	}
	if request.Status != domain.PermissionRequestStatusDenied {
		return request, nil
	}
	if input.Remember {
		_, _ = s.store.SavePermissionRule(ctx, s.permissionRuleFromRequest(ctx, request, domain.PermissionDecisionDeny))
	}
	if s.permissionNotifier != nil {
		s.permissionNotifier.resolve(request.ID)
	}
	if s.onPermissionResolved != nil {
		s.onPermissionResolved(request)
	}
	if s.onSessionUpdated != nil && request.SessionID != "" {
		s.onSessionUpdated(request.SessionID, nil)
	}
	return request, nil
}

// permissionRuleFromRequest scopes remembered decisions to the workspace when
// the originating session is available. Keeping the session fallback preserves
// approval behavior for requests created without a persisted session.
func (s *Service) permissionRuleFromRequest(ctx context.Context, request domain.PermissionRequest, decision string) domain.PermissionRule {
	rule := permissionRuleFromRequest(request, decision)
	if s.store == nil || strings.TrimSpace(request.SessionID) == "" {
		return rule
	}
	session, err := s.store.GetRuntimeSession(ctx, request.SessionID)
	if err != nil {
		return rule
	}
	if workspaceRoot := permissionWorkspaceRoot(ctx, s.store, session); workspaceRoot != "" {
		rule.WorkspaceRoot = workspaceRoot
		rule.SessionID = ""
	}
	return rule
}

func (s *Service) GetPermissionMode(ctx context.Context, sessionID string) (domain.PermissionModeState, error) {
	if s.store == nil {
		return domain.PermissionModeState{}, errors.New("store is not configured")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return domain.PermissionModeState{Mode: domain.PermissionModeRequestApproval}, nil
	}
	session, err := s.store.GetRuntimeSession(ctx, sessionID)
	if err != nil {
		return domain.PermissionModeState{}, err
	}
	workspaceRoot := permissionWorkspaceRoot(ctx, s.store, session)
	rules, err := s.store.ListPermissionRules(ctx, workspaceRoot, sessionID)
	if err != nil {
		return domain.PermissionModeState{}, err
	}
	for _, rule := range rules {
		if mode := permissionModeFromRule(rule); mode != "" {
			return domain.PermissionModeState{
				SessionID:     sessionID,
				WorkspaceRoot: workspaceRoot,
				Mode:          mode,
				TimeUpdated:   firstNonEmpty(rule.TimeUpdated, rule.TimeCreated),
			}, nil
		}
	}
	return domain.PermissionModeState{SessionID: sessionID, WorkspaceRoot: workspaceRoot, Mode: domain.PermissionModeRequestApproval}, nil
}

func (s *Service) SetPermissionMode(ctx context.Context, input domain.PermissionModeInput) (domain.PermissionModeState, error) {
	if s.store == nil {
		return domain.PermissionModeState{}, errors.New("store is not configured")
	}
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return domain.PermissionModeState{}, errors.New("sessionId is required")
	}
	mode, err := normalizePermissionMode(input.Mode)
	if err != nil {
		return domain.PermissionModeState{}, err
	}
	session, err := s.store.GetRuntimeSession(ctx, sessionID)
	if err != nil {
		return domain.PermissionModeState{}, err
	}
	workspaceRoot := permissionWorkspaceRoot(ctx, s.store, session)
	decision := domain.PermissionDecisionAsk
	if mode == domain.PermissionModeFullAccess {
		decision = domain.PermissionDecisionAllow
	}
	rule, err := s.store.SavePermissionRule(ctx, domain.PermissionRule{
		Scope:         permissionModeScopePrefix + mode,
		SessionID:     sessionID,
		WorkspaceRoot: workspaceRoot,
		ToolName:      permissionRuleWildcard,
		Action:        permissionActionWrite,
		Decision:      decision,
		Paths:         []string{permissionRuleWildcard},
	})
	if err != nil {
		return domain.PermissionModeState{}, err
	}
	if s.onSessionUpdated != nil {
		s.onSessionUpdated(sessionID, nil)
	}
	return domain.PermissionModeState{
		SessionID:     sessionID,
		WorkspaceRoot: workspaceRoot,
		Mode:          mode,
		TimeUpdated:   firstNonEmpty(rule.TimeUpdated, rule.TimeCreated),
	}, nil
}
