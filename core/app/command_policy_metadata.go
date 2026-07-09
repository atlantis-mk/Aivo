package app

import (
	"fmt"
	"path/filepath"
)

func commandPolicyMetadata(detection CommandDetection, policy CommandPolicyEvaluation, timeoutSeconds int, cwd string) map[string]any {
	return map[string]any{
		"command":           detection.NormalizedCommand,
		"cwd":               filepath.ToSlash(cwd),
		"argv":              detection.Argv,
		"category":          policy.Category,
		"riskLevel":         policy.RiskLevel,
		"networkHint":       policy.NetworkPolicy,
		"timeoutSeconds":    timeoutSeconds,
		"backend":           "local",
		"approvalKey":       detection.ApprovalKey,
		"policyDecision":    policy.Decision,
		"detectorReason":    firstNonEmpty(policy.Justification, detection.Reason),
		"hardline":          policy.Hardline,
		"matchedPattern":    policy.MatchedPattern,
		"externalPaths":     detection.ExternalPaths,
		"pathPatterns":      detection.Paths,
		"capabilities":      detection.Capabilities,
		"rememberScope":     "exact_command_cwd",
		"sandboxProfile":    "default",
		"normalizedCommand": detection.NormalizedCommand,
	}
}

func commandPolicyDeniedError(policy CommandPolicyEvaluation) error {
	return fmt.Errorf("command denied: %s", firstNonEmpty(policy.Justification, "command policy denied execution"))
}
