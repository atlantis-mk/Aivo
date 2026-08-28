package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aivo/core/domain"
)

func permissionRuleFromRequest(request domain.PermissionRequest, decision string) domain.PermissionRule {
	paths := request.Paths
	if request.Action == permissionActionShell || request.Action == permissionActionTest {
		if key, _ := request.Arguments["approvalKey"].(string); strings.TrimSpace(key) != "" {
			paths = []string{permissionApprovalKeyPath + strings.TrimSpace(key)}
		}
	}
	return domain.PermissionRule{
		Scope: "workspace", SessionID: request.SessionID, ToolName: request.ToolName,
		Action: request.Action, Decision: decision, Paths: paths,
	}
}

func permissionWorkspaceRoot(ctx context.Context, store Store, session domain.Session) string {
	workspaceRoot := strings.TrimSpace(session.ProjectPath)
	if workspaceRoot != "" {
		return workspaceRoot
	}
	if store == nil || strings.TrimSpace(session.ID) == "" {
		return ""
	}
	codingContext, err := store.GetCodingContext(ctx, session.ID)
	if err == nil {
		workspaceRoot = strings.TrimSpace(codingContext.ProjectPath)
	}
	return workspaceRoot
}

func normalizePermissionMode(mode string) (string, error) {
	switch strings.TrimSpace(mode) {
	case "", domain.PermissionModeRequestApproval:
		return domain.PermissionModeRequestApproval, nil
	case domain.PermissionModeFullAccess:
		return domain.PermissionModeFullAccess, nil
	default:
		return "", fmt.Errorf("unknown permission mode %q", mode)
	}
}

func permissionModeFromRule(rule domain.PermissionRule) string {
	if !strings.HasPrefix(rule.Scope, permissionModeScopePrefix) {
		return ""
	}
	rawMode := strings.TrimSpace(strings.TrimPrefix(rule.Scope, permissionModeScopePrefix))
	if rawMode == legacyPermissionModeAutoApprove {
		return domain.PermissionModeRequestApproval
	}
	mode, err := normalizePermissionMode(rawMode)
	if err != nil {
		return ""
	}
	return mode
}

func isPermissionModeRule(rule domain.PermissionRule) bool {
	return permissionModeFromRule(rule) != ""
}

func permissionRuleApplies(rule domain.PermissionRule, toolName string, action string) bool {
	return (rule.ToolName == permissionRuleWildcard || rule.ToolName == toolName) &&
		(rule.Action == permissionRuleWildcard || rule.Action == action)
}

func permissionExemptTool(spec domain.ToolSpec) bool {
	return spec.Name == "update_plan" && spec.Category == "plan" && spec.Capability == "plan.write"
}

func permissionActionForSpec(spec domain.ToolSpec) string {
	if spec.Name == SkillToolName || spec.Category == "skill" || spec.Capability == "skill.load" {
		return permissionActionSkill
	}
	if spec.Capability == "shell.exec" {
		return permissionActionShell
	}
	if spec.Capability == "shell.test" {
		return permissionActionTest
	}
	if strings.Contains(spec.Capability, ".write") || strings.Contains(spec.Capability, ".patch") {
		return permissionActionWrite
	}
	return permissionActionRead
}

func modePermissionDecision(mode string, spec domain.ToolSpec, action string) PermissionEvaluation {
	mode = strings.TrimSpace(mode)
	if mode == "" || mode == domain.AgentModeAssistant {
		return PermissionEvaluation{}
	}
	deny := func(reason string) PermissionEvaluation {
		return PermissionEvaluation{Decision: domain.PermissionDecisionDeny, Reason: reason}
	}
	switch mode {
	case domain.AgentModeExplore, domain.AgentModePlan, domain.AgentModePlanner, domain.AgentModeReview, domain.AgentModeSummary, domain.AgentModeTitle:
		if action == permissionActionWrite || action == permissionActionShell || action == permissionActionTest || isSchedulerTool(spec) {
			return deny(mode + " mode cannot mutate files, run shell commands, run tests, or change scheduled jobs")
		}
	case domain.AgentModeDebug:
		if action == permissionActionWrite || isSchedulerTool(spec) {
			return deny("debug mode cannot mutate files or change scheduled jobs")
		}
	case domain.AgentModeSchedulerWorker:
		if isAdminTool(spec) || isMCPTool(spec) {
			return deny("scheduler worker mode cannot use admin or mcp tools")
		}
	}
	return PermissionEvaluation{}
}

func scopePermissionDecision(scope string, spec domain.ToolSpec, action string) PermissionEvaluation {
	scope = strings.TrimSpace(strings.ToLower(scope))
	if scope == "" || scope == "workspace" || scope == "workspace_approval" {
		return PermissionEvaluation{}
	}
	deny := func(reason string) PermissionEvaluation {
		return PermissionEvaluation{Decision: domain.PermissionDecisionDeny, Reason: reason}
	}
	switch scope {
	case "read_only", "readonly", "safe":
		if action != permissionActionRead && action != permissionActionSkill {
			return deny("permission scope is read-only")
		}
	case "no_shell":
		if action == permissionActionShell || action == permissionActionTest {
			return deny("permission scope forbids shell and test execution")
		}
	default:
		return deny("unknown permission scope: " + scope)
	}
	return PermissionEvaluation{}
}

func isSchedulerTool(spec domain.ToolSpec) bool {
	return spec.Category == "scheduler" || strings.HasPrefix(spec.Name, "scheduler_") || strings.HasPrefix(spec.Capability, "scheduler.")
}

func isAdminTool(spec domain.ToolSpec) bool {
	if spec.Category == "admin" || strings.HasPrefix(spec.Capability, "admin.") {
		return true
	}
	for _, toolset := range spec.Toolsets {
		if toolset == "admin" {
			return true
		}
	}
	return false
}

func isMCPTool(spec domain.ToolSpec) bool {
	if spec.Category == "mcp" || strings.HasPrefix(spec.Capability, "mcp.") {
		return true
	}
	for _, toolset := range spec.Toolsets {
		if toolset == "mcp" {
			return true
		}
	}
	return false
}

func permissionPathsForTool(name string, args json.RawMessage, execCtx domain.ToolExecutionContext) ([]string, map[string]any, error) {
	switch name {
	case "bash":
		input, err := parsePrimitiveBashArgs(args)
		if err != nil {
			return nil, nil, err
		}
		prepared, err := prepareShellCommand(execCtx.WorkspaceRoot, execCtx, "bash", input.Command, input.CWD, input.TimeoutSeconds, input.Network, input.Mode, input.Stdin, input.Env)
		if err != nil {
			return prepared.detect.Paths, prepared.metadata, err
		}
		if strings.TrimSpace(input.Justification) != "" {
			prepared.metadata["justification"] = input.Justification
		}
		return prepared.detect.Paths, prepared.metadata, nil
	case ExecCommandToolName:
		input, err := parseExecCommandArgs(args)
		if err != nil {
			return nil, nil, err
		}
		prepared, err := prepareShellCommand(execCtx.WorkspaceRoot, execCtx, ExecCommandToolName, input.Command, input.CWD, 0, input.Network, "pty", "", input.Env)
		if err != nil {
			return prepared.detect.Paths, prepared.metadata, err
		}
		prepared.metadata["interactive"] = true
		prepared.metadata["yieldTimeMs"] = input.YieldTimeMS
		if input.Justification != "" {
			prepared.metadata["justification"] = input.Justification
		}
		return prepared.detect.Paths, prepared.metadata, nil
	case WriteStdinToolName:
		input, err := parseWriteStdinArgs(args)
		if err != nil {
			return nil, nil, err
		}
		if err := defaultAgentPTYRegistry.ValidateOwner(execCtx.WorkspaceRoot, execCtx.SessionID, input.ProcessRef); err != nil {
			return nil, nil, err
		}
		capabilities := []string{"shell.interactive.observe"}
		category := CommandCategoryRead
		riskLevel := CommandRiskLow
		hasInput := input.Chars != "" || input.PressEnter
		if hasInput {
			capabilities = append(capabilities, "shell.stdin")
			category = CommandCategoryUnknown
			riskLevel = CommandRiskHigh
		}
		if input.Rows > 0 && input.Cols > 0 {
			capabilities = append(capabilities, "shell.resize")
		}
		if input.Terminate {
			capabilities = append(capabilities, "shell.terminate")
			category = CommandCategoryDangerous
			riskLevel = CommandRiskHigh
		}
		approvalKey := commandApprovalKey(execCtx.WorkspaceRoot, execCtx.WorkspaceRoot, input.ProcessRef, []string{input.ProcessRef}, WriteStdinToolName, "local", "pty", "deny", category, riskLevel, capabilities)
		return nil, map[string]any{
			"approvalKey": approvalKey, "processRef": input.ProcessRef, "interactive": true,
			"stdinPresent": hasInput, "terminate": input.Terminate,
			"resize": input.Rows > 0 && input.Cols > 0, "capabilities": capabilities,
			"category": category, "riskLevel": riskLevel, "networkPolicy": "deny",
			"policyDecision": CommandDecisionAllow,
		}, nil
	case "run_tests":
		input, err := parseRunTestsArgs(args)
		if err != nil {
			return nil, nil, err
		}
		commands, err := runTestsCommands(input)
		if err != nil {
			return nil, nil, err
		}
		if len(commands) != 1 {
			return nil, nil, errors.New("run_tests currently supports one declared command per invocation")
		}
		prepared, err := prepareShellCommand(execCtx.WorkspaceRoot, execCtx, "run_tests", commands[0], "", input.TimeoutSeconds, "deny", "foreground", "", nil)
		if err != nil {
			return prepared.detect.Paths, prepared.metadata, err
		}
		return prepared.detect.Paths, prepared.metadata, nil
	case "read_diagnostics":
		input, err := parseDiagnosticsArgs(args)
		if err != nil {
			return nil, nil, err
		}
		command, err := diagnosticsCommand(input)
		if err != nil {
			return nil, nil, err
		}
		prepared, err := prepareShellCommand(execCtx.WorkspaceRoot, execCtx, "read_diagnostics", command, "", input.TimeoutSeconds, "deny", "foreground", "", nil)
		if err != nil {
			return prepared.detect.Paths, prepared.metadata, err
		}
		return prepared.detect.Paths, prepared.metadata, nil
	case "format_code":
		input, err := parseFormatCodeArgs(args)
		if err != nil {
			return nil, nil, err
		}
		files := make([]map[string]any, 0, len(input.Paths))
		for _, path := range input.Paths {
			target, err := safeTargetForWrite(execCtx.WorkspaceRoot, path)
			if err != nil {
				return nil, nil, err
			}
			if !formatPathSupported(path) {
				return nil, nil, fmt.Errorf("unsupported formatter target %q", path)
			}
			baseHash := ""
			if hash, exists, _ := fileHashIfExists(target); exists {
				baseHash = hash
			} else {
				baseHash = "<missing>"
			}
			files = append(files, map[string]any{"path": path, "fullPath": fullWorkspacePath(execCtx.WorkspaceRoot, path), "type": "format", "baseHash": baseHash, "currentHash": baseHash})
		}
		return input.Paths, map[string]any{"files": files, "riskLevel": "medium", "rememberScope": "exact_paths"}, nil
	case "write", "write_file":
		if name == "write" {
			var input struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			if err := decodeStrictToolArgs(args, &input); err != nil {
				return nil, nil, err
			}
			path := filepath.ToSlash(filepath.Clean(input.Path))
			oldText := ""
			baseHash := "<missing>"
			if target, err := safeTargetForWrite(execCtx.WorkspaceRoot, path); err == nil {
				if raw, readErr := os.ReadFile(target); readErr == nil {
					oldText = string(raw)
					if hash, exists, _ := fileHashIfExists(target); exists {
						baseHash = hash
					}
				}
			}
			diff := simpleFileDiff(path, path, oldText, input.Content)
			file := map[string]any{"path": path, "fullPath": fullWorkspacePath(execCtx.WorkspaceRoot, path), "type": "write", "diff": bounded(diff, primitiveDiffMaxChars), "baseHash": baseHash, "currentHash": baseHash}
			storePreparedExpectedHash(execCtx.ToolCallID, path, baseHash)
			return []string{path}, map[string]any{"files": []map[string]any{file}, "diff": bounded(diff, primitiveDiffMaxChars), "riskLevel": "high", "rememberScope": "exact_paths"}, nil
		}
		input, err := parseWriteFileArgs(args)
		if err != nil {
			return nil, nil, err
		}
		path := filepath.ToSlash(filepath.Clean(input.Path))
		oldText := ""
		if target, err := safeTargetForWrite(execCtx.WorkspaceRoot, path); err == nil {
			if raw, readErr := os.ReadFile(target); readErr == nil {
				oldText = string(raw)
			}
		}
		additions, deletions := countLineDelta(oldText, input.Content)
		diff := simpleFileDiff(path, path, oldText, input.Content)
		baseHash := ""
		if target, err := safeTargetForWrite(execCtx.WorkspaceRoot, path); err == nil {
			if hash, exists, _ := fileHashIfExists(target); exists {
				baseHash = hash
			} else {
				baseHash = "<missing>"
			}
		}
		storePreparedExpectedHash(execCtx.ToolCallID, path, baseHash)
		file := map[string]any{"path": path, "fullPath": fullWorkspacePath(execCtx.WorkspaceRoot, path), "type": "write", "additions": additions, "deletions": deletions, "diff": diff, "baseHash": baseHash, "currentHash": baseHash}
		return []string{path}, map[string]any{"files": []map[string]any{file}, "diff": diff, "riskLevel": "high", "rememberScope": "exact_paths"}, nil
	case "edit", "edit_file":
		if name == "edit" {
			var input struct {
				Path  string          `json:"path"`
				Edits []primitiveEdit `json:"edits"`
			}
			if err := decodeStrictToolArgs(args, &input); err != nil {
				return nil, nil, err
			}
			path := filepath.ToSlash(filepath.Clean(input.Path))
			target, err := safeTargetForWrite(execCtx.WorkspaceRoot, path)
			if err != nil {
				return nil, nil, err
			}
			raw, err := os.ReadFile(target)
			if err != nil {
				return nil, nil, err
			}
			before := string(raw)
			after := before
			for _, item := range input.Edits {
				if strings.Count(before, item.OldText) != 1 {
					return nil, nil, errors.New("each oldText must occur exactly once")
				}
				after = strings.Replace(after, item.OldText, item.NewText, 1)
			}
			diff := bounded(simpleFileDiff(path, path, before, after), primitiveDiffMaxChars)
			baseHash := ""
			if snapshot, snapErr := snapshotForBytes(path, target, raw, "all", false); snapErr == nil {
				baseHash = snapshot.SHA256
			}
			file := map[string]any{"path": path, "fullPath": fullWorkspacePath(execCtx.WorkspaceRoot, path), "type": "edit", "diff": diff, "baseHash": baseHash, "currentHash": baseHash}
			storePreparedExpectedHash(execCtx.ToolCallID, path, baseHash)
			return []string{path}, map[string]any{"files": []map[string]any{file}, "diff": diff, "riskLevel": "high", "rememberScope": "exact_paths"}, nil
		}
		input, err := parseEditFileArgs(args)
		if err != nil {
			return nil, nil, err
		}
		path := filepath.ToSlash(filepath.Clean(input.Path))
		target, err := safeTargetForWrite(execCtx.WorkspaceRoot, path)
		if err != nil {
			return nil, nil, err
		}
		raw, err := os.ReadFile(target)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read file: %s", path)
		}
		oldText := string(raw)
		replacements := strings.Count(oldText, input.OldString)
		if replacements == 0 {
			return nil, nil, errors.New("oldString was not found in the file")
		}
		if replacements > 1 && !input.ReplaceAll {
			return nil, nil, errors.New("oldString appears multiple times; provide more context or set replaceAll=true")
		}
		newText := strings.Replace(oldText, input.OldString, input.NewString, 1)
		if input.ReplaceAll {
			newText = strings.ReplaceAll(oldText, input.OldString, input.NewString)
		}
		additions, deletions := countLineDelta(oldText, newText)
		diff := simpleFileDiff(path, path, oldText, newText)
		baseHash := ""
		if snapshot, snapErr := snapshotForBytes(path, target, raw, "all", false); snapErr == nil {
			baseHash = snapshot.SHA256
		}
		storePreparedExpectedHash(execCtx.ToolCallID, path, baseHash)
		file := map[string]any{"path": path, "fullPath": fullWorkspacePath(execCtx.WorkspaceRoot, path), "type": "edit", "additions": additions, "deletions": deletions, "diff": diff, "baseHash": baseHash, "currentHash": baseHash}
		return []string{path}, map[string]any{"files": []map[string]any{file}, "diff": diff, "riskLevel": "high", "rememberScope": "exact_paths"}, nil
	default:
		var input struct {
			Path string `json:"path"`
		}
		_ = json.Unmarshal(args, &input)
		if strings.TrimSpace(input.Path) == "" {
			return nil, nil, nil
		}
		return []string{filepath.ToSlash(filepath.Clean(input.Path))}, nil, nil
	}
}

func deniedPathReason(paths []string) string {
	for _, path := range paths {
		clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
		if clean == "." || clean == "" {
			continue
		}
		if filepath.IsAbs(clean) {
			return workspaceRelativePathErrorMessage
		}
		if clean == ".." || strings.HasPrefix(clean, "../") {
			return "path escapes workspace root"
		}
		if strings.HasPrefix(clean, ".git/") || clean == ".git" {
			return "refusing to modify .git"
		}
		if isSensitiveRelPath(clean) {
			return "refusing to modify sensitive file"
		}
	}
	return ""
}

func pathsCovered(paths []string, allowed []string) bool {
	if len(allowed) == 0 {
		return len(paths) == 0
	}
	allowedSet := map[string]bool{}
	for _, path := range allowed {
		clean := filepath.ToSlash(filepath.Clean(path))
		if clean == permissionRuleWildcard {
			return true
		}
		allowedSet[clean] = true
	}
	for _, path := range paths {
		if !allowedSet[filepath.ToSlash(filepath.Clean(path))] {
			return false
		}
	}
	return true
}

func permissionRuleMatches(rule domain.PermissionRule, action string, paths []string, metadata map[string]any) bool {
	if action == permissionActionShell || action == permissionActionTest {
		key, _ := metadata["approvalKey"].(string)
		if strings.TrimSpace(key) == "" {
			return false
		}
		return pathsCovered([]string{permissionApprovalKeyPath + strings.TrimSpace(key)}, rule.Paths)
	}
	return pathsCovered(paths, rule.Paths)
}

func commandPolicyDecision(metadata map[string]any) string {
	if metadata == nil {
		return ""
	}
	decision, _ := metadata["policyDecision"].(string)
	return strings.TrimSpace(decision)
}

func commandPolicyCategoryAllowsFullAccess(metadata map[string]any) bool {
	if metadata == nil {
		return false
	}
	if rawCaps, ok := metadata["capabilities"].([]string); ok && commandHasAdvancedCapabilities(rawCaps) {
		return false
	}
	if rawCaps, ok := metadata["capabilities"].([]any); ok {
		caps := make([]string, 0, len(rawCaps))
		for _, item := range rawCaps {
			if value, ok := item.(string); ok {
				caps = append(caps, value)
			}
		}
		if commandHasAdvancedCapabilities(caps) {
			return false
		}
	}
	category, _ := metadata["category"].(string)
	switch strings.TrimSpace(category) {
	case CommandCategoryRead, CommandCategoryTest:
		return true
	default:
		return false
	}
}

func addPatchPath(seen map[string]bool, path string) {
	path = filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
	if path == "." || path == "" || path == "/dev/null" {
		return
	}
	seen[path] = true
}
