package app

import (
	"errors"
	"fmt"
	"time"

	"aivo/core/domain"
)

type preparedShellCommand struct {
	request   SandboxRequest
	detect    CommandDetection
	policy    CommandPolicyEvaluation
	cwd       string
	timeout   time.Duration
	timeoutS  int
	metadata  map[string]any
	toolName  string
	workspace string
}

func prepareShellCommand(workspaceRoot string, execCtx domain.ToolExecutionContext, toolName string, command string, cwdArg string, timeoutSeconds int, network string, mode string, stdin string, shell string, login bool, env map[string]string) (preparedShellCommand, error) {
	prepared := preparedShellCommand{toolName: toolName, workspace: workspaceRoot}
	mode = normalizeSandboxMode(mode)
	externalCWD, cwd, err := cwdIsExternal(workspaceRoot, cwdArg)
	if err != nil {
		return prepared, err
	}
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeoutSeconds <= 0 {
		timeout = defaultCommandTimeout
		timeoutSeconds = int(defaultCommandTimeout.Seconds())
	}
	timeout = clampCommandTimeout(timeout)
	timeoutSeconds = int(timeout.Seconds())
	detection := DetectCommand(command, cwd, workspaceRoot, toolName)
	rawCommand := detection.RawCommand
	capabilities := []string{"shell.exec." + mode}
	if stdin != "" {
		capabilities = append(capabilities, "shell.stdin")
	}
	if login {
		capabilities = append(capabilities, "shell.login")
	}
	if len(env) > 0 {
		for key := range env {
			if isSecretEnvName(key) {
				prepared.detect = detection
				return prepared, fmt.Errorf("env override %s is denied because it looks secret-bearing", key)
			}
			if !envOverrideKeyAllowed(key) {
				prepared.detect = detection
				return prepared, fmt.Errorf("env override %s is not allowed by Phase 3 env policy", key)
			}
		}
		capabilities = append(capabilities, "shell.env.override")
	}
	if externalCWD {
		capabilities = append(capabilities, "shell.cwd.external")
		detection.ExternalPaths = appendUniqueStrings(detection.ExternalPaths, cwd)
	}
	if network == "inherit" || detection.Category == CommandCategoryNetwork {
		capabilities = append(capabilities, "shell.network")
	}
	detection.Capabilities = appendUniqueStrings(detection.Capabilities, capabilities...)
	detection.ApprovalKey = commandApprovalKey(workspaceRoot, cwd, rawCommand, detection.Argv, toolName, "local", "default", firstNonEmpty(network, detection.NetworkHint), detection.Category, detection.RiskLevel, shell, login, detection.Capabilities)
	policy := EvaluateCommandPolicy(detection, toolName)
	if network == "" {
		network = policy.NetworkPolicy
	}
	if network == "" {
		network = "deny"
	}
	if network == "deny" && policy.Category == CommandCategoryNetwork {
		policy.Decision = CommandDecisionAsk
		policy.NetworkPolicy = "deny"
		policy.Justification = firstNonEmpty(policy.Justification, "network command requested with denied network policy")
	}
	prepared.detect = detection
	prepared.policy = policy
	prepared.cwd = cwd
	prepared.timeout = timeout
	prepared.timeoutS = timeoutSeconds
	prepared.metadata = commandPolicyMetadata(detection, policy, timeoutSeconds, cwd, shell, login)
	prepared.request = SandboxRequest{
		WorkspaceRoot:    workspaceRoot,
		CWD:              cwd,
		Command:          rawCommand,
		Argv:             detection.Argv,
		Mode:             mode,
		Timeout:          timeout,
		Stdin:            stdin,
		NetworkPolicy:    network,
		Backend:          "local",
		OutputPolicy:     execCtx.OutputPolicy,
		SessionID:        execCtx.SessionID,
		TurnID:           execCtx.TurnID,
		ToolCallID:       execCtx.ToolCallID,
		ToolName:         toolName,
		ApprovalKey:      detection.ApprovalKey,
		EnvAllowlist:     defaultEnvAllowlist(),
		Env:              nil,
		EnvOverrides:     env,
		Shell:            shell,
		LoginShell:       login,
		AllowExternalCWD: externalCWD,
	}
	if policy.Decision == CommandDecisionDeny {
		return prepared, commandPolicyDeniedError(policy)
	}
	if toolName == "run_tests" && policy.Decision != CommandDecisionAllow {
		return prepared, errors.New("run_tests command is not an allowed declared command")
	}
	return prepared, nil
}
