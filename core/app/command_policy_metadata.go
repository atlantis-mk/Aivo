package app

import (
	"fmt"
	"path/filepath"
)

func commandPolicyMetadata(detection CommandDetection, policy CommandPolicyEvaluation, timeoutSeconds int, cwd string, shell string, login bool) map[string]any {
	return map[string]any{
		"cmd":            detection.RawCommand,
		"workdir":        filepath.ToSlash(cwd),
		"argv":           detection.Argv,
		"shell":          shell,
		"loginShell":     login,
		"heredoc":        detection.HasHeredoc,
		"category":       policy.Category,
		"riskLevel":      policy.RiskLevel,
		"networkHint":    policy.NetworkPolicy,
		"timeoutSeconds": timeoutSeconds,
		"backend":        "local",
		"approvalKey":    detection.ApprovalKey,
		"policyDecision": policy.Decision,
		"detectorReason": firstNonEmpty(policy.Justification, detection.Reason),
		"hardline":       policy.Hardline,
		"matchedPattern": policy.MatchedPattern,
		"externalPaths":  detection.ExternalPaths,
		"pathPatterns":   detection.Paths,
		"capabilities":   detection.Capabilities,
		"rememberScope":  "exact_command_workdir",
		"sandboxProfile": "default",
	}
}

func commandPolicyDeniedError(policy CommandPolicyEvaluation) error {
	return fmt.Errorf("command denied: %s", firstNonEmpty(policy.Justification, "command policy denied execution"))
}
