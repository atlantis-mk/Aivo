package app

import (
	"strings"
)

const (
	CommandCategoryRead      = "read"
	CommandCategoryTest      = "test"
	CommandCategoryBuild     = "build"
	CommandCategoryWrite     = "write"
	CommandCategoryNetwork   = "network"
	CommandCategoryDangerous = "dangerous"
	CommandCategoryUnknown   = "unknown"

	CommandRiskLow      = "low"
	CommandRiskMedium   = "medium"
	CommandRiskHigh     = "high"
	CommandRiskCritical = "critical"

	CommandDecisionAllow = "allow"
	CommandDecisionAsk   = "ask"
	CommandDecisionDeny  = "deny"
)

type CommandDetection struct {
	RawCommand        string
	NormalizedCommand string
	Category          string
	RiskLevel         string
	Paths             []string
	ExternalPaths     []string
	NetworkHint       string
	Capabilities      []string
	DenyReason        string
	ApprovalKey       string
	Argv              []string
	HasMetacharacters bool
	HasHeredoc        bool
	Reason            string
}

type CommandPolicyRule struct {
	Pattern       []string
	Decision      string
	RiskLevel     string
	Category      string
	NetworkPolicy string
	Justification string
	Hardline      bool
}

type CommandPolicyEvaluation struct {
	Decision       string
	RiskLevel      string
	Category       string
	NetworkPolicy  string
	Justification  string
	Hardline       bool
	MatchedPattern []string
}

func DetectCommand(rawCommand string, cwd string, workspaceRoot string, toolName string) CommandDetection {
	command := strings.TrimSpace(rawCommand)
	normalized := normalizeCommandText(rawCommand)
	detection := CommandDetection{
		RawCommand:        command,
		NormalizedCommand: normalized,
		Category:          CommandCategoryUnknown,
		RiskLevel:         CommandRiskHigh,
		NetworkHint:       "deny",
		Reason:            "command is unknown",
	}
	if normalized == "" {
		detection.DenyReason = "command is required"
		detection.Category = CommandCategoryDangerous
		detection.RiskLevel = CommandRiskCritical
		return detection
	}
	tokens, err := shellTokenize(normalized)
	if err != nil || len(tokens) == 0 {
		detection.RiskLevel = CommandRiskCritical
		detection.DenyReason = firstNonEmpty(errorString(err), "command could not be tokenized safely")
		return detection
	}
	detection.Argv = tokens
	detection.HasMetacharacters = commandHasMetacharacters(normalized)
	if heredocIndex := shellHeredocTokenIndex(tokens); heredocIndex >= 0 {
		detection.HasHeredoc = true
		detection.Capabilities = appendUniqueStrings(detection.Capabilities, "shell.heredoc")
	}
	if isSudoCommand(tokens) {
		detection.Capabilities = appendUniqueStrings(detection.Capabilities, "shell.sudo")
		detection.Category = CommandCategoryDangerous
		detection.RiskLevel = CommandRiskCritical
		detection.Reason = "sudo requires privileged shell capability"
	}
	if deny := hardlineCommandDenyReason(tokens, normalized); deny != "" {
		detection.Category = CommandCategoryDangerous
		detection.RiskLevel = CommandRiskCritical
		detection.DenyReason = deny
		detection.Reason = deny
	}

	pathTokens := tokens
	if heredocIndex := shellHeredocTokenIndex(tokens); heredocIndex >= 0 {
		pathTokens = tokens[:heredocIndex]
	}
	paths, external := commandPathHints(pathTokens, cwd, workspaceRoot)
	detection.Paths = paths
	detection.ExternalPaths = external
	if len(external) > 0 && detection.DenyReason == "" {
		detection.Category = CommandCategoryDangerous
		detection.RiskLevel = CommandRiskCritical
		detection.DenyReason = "command targets a path outside the workspace"
		detection.Reason = detection.DenyReason
	}
	if detection.DenyReason == "" {
		classifyCommand(&detection)
	}
	if detection.Category == CommandCategoryNetwork {
		detection.Capabilities = appendUniqueStrings(detection.Capabilities, "shell.network")
	}
	detection.ApprovalKey = commandApprovalKey(workspaceRoot, cwd, command, tokens, toolName, "local", "default", detection.NetworkHint, detection.Category, detection.RiskLevel, "", false, detection.Capabilities)
	return detection
}

func EvaluateCommandPolicy(detection CommandDetection, toolName string) CommandPolicyEvaluation {
	evaluation := CommandPolicyEvaluation{
		Decision:      CommandDecisionAsk,
		RiskLevel:     firstNonEmpty(detection.RiskLevel, CommandRiskHigh),
		Category:      firstNonEmpty(detection.Category, CommandCategoryUnknown),
		NetworkPolicy: firstNonEmpty(detection.NetworkHint, "deny"),
		Justification: detection.Reason,
	}
	if detection.DenyReason != "" {
		evaluation.Decision = CommandDecisionDeny
		evaluation.Hardline = true
		evaluation.Justification = detection.DenyReason
		return evaluation
	}
	rules := commandPolicyRules()
	for _, rule := range rules {
		if !tokensHavePrefix(detection.Argv, rule.Pattern) {
			continue
		}
		if policyDecisionRank(rule.Decision) > policyDecisionRank(evaluation.Decision) || len(evaluation.MatchedPattern) == 0 {
			evaluation.Decision = rule.Decision
			evaluation.RiskLevel = firstNonEmpty(rule.RiskLevel, evaluation.RiskLevel)
			evaluation.Category = firstNonEmpty(rule.Category, evaluation.Category)
			evaluation.NetworkPolicy = firstNonEmpty(rule.NetworkPolicy, evaluation.NetworkPolicy)
			evaluation.Justification = firstNonEmpty(rule.Justification, evaluation.Justification)
			evaluation.Hardline = rule.Hardline
			evaluation.MatchedPattern = append([]string(nil), rule.Pattern...)
		}
	}
	if toolName == "run_tests" && evaluation.Decision == CommandDecisionAsk && detection.Category == CommandCategoryUnknown {
		evaluation.Decision = CommandDecisionDeny
		evaluation.Justification = "run_tests only supports declared test, lint, and build commands"
	}
	if toolName == ExecCommandToolName && detection.HasMetacharacters && evaluation.Decision == CommandDecisionAllow {
		evaluation.Decision = CommandDecisionAsk
		evaluation.RiskLevel = maxRisk(evaluation.RiskLevel, CommandRiskHigh)
		evaluation.Justification = "shell metacharacters require explicit approval"
	}
	if detection.Category == CommandCategoryNetwork && evaluation.NetworkPolicy == "deny" && toolName == ExecCommandToolName {
		evaluation.Decision = CommandDecisionAsk
		evaluation.Justification = firstNonEmpty(evaluation.Justification, "network-capable command requires approval")
	}
	if commandHasAdvancedCapabilities(detection.Capabilities) && evaluation.Decision == CommandDecisionAllow {
		evaluation.Decision = CommandDecisionAsk
		evaluation.Justification = firstNonEmpty(evaluation.Justification, "advanced shell capabilities require approval")
	}
	return evaluation
}

func commandHasAdvancedCapabilities(capabilities []string) bool {
	for _, capability := range capabilities {
		switch strings.TrimSpace(capability) {
		case "", "shell.exec.foreground":
			continue
		default:
			return true
		}
	}
	return false
}

func commandPolicyRules() []CommandPolicyRule {
	return []CommandPolicyRule{
		{Pattern: []string{"rm", "-rf", "/"}, Decision: CommandDecisionDeny, RiskLevel: CommandRiskCritical, Category: CommandCategoryDangerous, Justification: "refusing destructive root deletion", Hardline: true},
		{Pattern: []string{"rm", "-rf", "."}, Decision: CommandDecisionDeny, RiskLevel: CommandRiskCritical, Category: CommandCategoryDangerous, Justification: "refusing destructive workspace deletion", Hardline: true},
		{Pattern: []string{"sudo"}, Decision: CommandDecisionDeny, RiskLevel: CommandRiskCritical, Category: CommandCategoryDangerous, Justification: "sudo is not supported in the sandbox", Hardline: true},
		{Pattern: []string{"chmod", "-R", "777"}, Decision: CommandDecisionDeny, RiskLevel: CommandRiskCritical, Category: CommandCategoryDangerous, Justification: "refusing broad world-writable chmod", Hardline: true},
		{Pattern: []string{"mkfs"}, Decision: CommandDecisionDeny, RiskLevel: CommandRiskCritical, Category: CommandCategoryDangerous, Justification: "disk formatting commands are blocked", Hardline: true},
		{Pattern: []string{"git", "status"}, Decision: CommandDecisionAllow, RiskLevel: CommandRiskLow, Category: CommandCategoryRead, NetworkPolicy: "deny", Justification: "read-only git status"},
		{Pattern: []string{"git", "diff"}, Decision: CommandDecisionAllow, RiskLevel: CommandRiskLow, Category: CommandCategoryRead, NetworkPolicy: "deny", Justification: "read-only git diff"},
		{Pattern: []string{"git", "log"}, Decision: CommandDecisionAllow, RiskLevel: CommandRiskLow, Category: CommandCategoryRead, NetworkPolicy: "deny", Justification: "read-only git log"},
		{Pattern: []string{"pwd"}, Decision: CommandDecisionAllow, RiskLevel: CommandRiskLow, Category: CommandCategoryRead, NetworkPolicy: "deny", Justification: "read-only working directory inspection"},
		{Pattern: []string{"ls"}, Decision: CommandDecisionAllow, RiskLevel: CommandRiskLow, Category: CommandCategoryRead, NetworkPolicy: "deny", Justification: "read-only directory listing"},
		{Pattern: []string{"find"}, Decision: CommandDecisionAllow, RiskLevel: CommandRiskLow, Category: CommandCategoryRead, NetworkPolicy: "deny", Justification: "read-only file discovery"},
		{Pattern: []string{"go", "test"}, Decision: CommandDecisionAllow, RiskLevel: CommandRiskMedium, Category: CommandCategoryTest, NetworkPolicy: "deny", Justification: "known test command"},
		{Pattern: []string{"gofmt", "-w"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskMedium, Category: CommandCategoryWrite, NetworkPolicy: "deny", Justification: "formatter writes source files"},
		{Pattern: []string{"node_modules/.bin/prettier", "--write"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskMedium, Category: CommandCategoryWrite, NetworkPolicy: "deny", Justification: "formatter writes source files"},
		{Pattern: []string{"node_modules/.bin/prettier.cmd", "--write"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskMedium, Category: CommandCategoryWrite, NetworkPolicy: "deny", Justification: "formatter writes source files"},
		{Pattern: []string{"npx", "--no-install", "prettier", "--write"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskMedium, Category: CommandCategoryWrite, NetworkPolicy: "deny", Justification: "project-local formatter writes source files"},
		{Pattern: []string{"node_modules/.bin/eslint", "--fix"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskMedium, Category: CommandCategoryWrite, NetworkPolicy: "deny", Justification: "project-local lint fixer writes source files"},
		{Pattern: []string{"node_modules/.bin/eslint.cmd", "--fix"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskMedium, Category: CommandCategoryWrite, NetworkPolicy: "deny", Justification: "project-local lint fixer writes source files"},
		{Pattern: []string{"npx", "--no-install", "eslint", "--fix"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskMedium, Category: CommandCategoryWrite, NetworkPolicy: "deny", Justification: "project-local lint fixer writes source files"},
		{Pattern: []string{"rustfmt"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskMedium, Category: CommandCategoryWrite, NetworkPolicy: "deny", Justification: "formatter writes source files"},
		{Pattern: []string{".venv/bin/black"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskMedium, Category: CommandCategoryWrite, NetworkPolicy: "deny", Justification: "formatter writes source files"},
		{Pattern: []string{"venv/bin/black"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskMedium, Category: CommandCategoryWrite, NetworkPolicy: "deny", Justification: "formatter writes source files"},
		{Pattern: []string{".venv/Scripts/black.exe"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskMedium, Category: CommandCategoryWrite, NetworkPolicy: "deny", Justification: "formatter writes source files"},
		{Pattern: []string{"venv/Scripts/black.exe"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskMedium, Category: CommandCategoryWrite, NetworkPolicy: "deny", Justification: "formatter writes source files"},
		{Pattern: []string{"python", "-m", "black"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskMedium, Category: CommandCategoryWrite, NetworkPolicy: "deny", Justification: "formatter writes source files"},
		{Pattern: []string{"shfmt", "-w"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskMedium, Category: CommandCategoryWrite, NetworkPolicy: "deny", Justification: "formatter writes source files"},
		{Pattern: []string{"npm", "run", "test:core"}, Decision: CommandDecisionAllow, RiskLevel: CommandRiskMedium, Category: CommandCategoryTest, NetworkPolicy: "deny", Justification: "known core test command"},
		{Pattern: []string{"npm", "run", "lint"}, Decision: CommandDecisionAllow, RiskLevel: CommandRiskMedium, Category: CommandCategoryTest, NetworkPolicy: "deny", Justification: "known lint command"},
		{Pattern: []string{"npm", "run", "build"}, Decision: CommandDecisionAllow, RiskLevel: CommandRiskMedium, Category: CommandCategoryBuild, NetworkPolicy: "deny", Justification: "known build command"},
		{Pattern: []string{"npm", "run", "diagnostics"}, Decision: CommandDecisionAllow, RiskLevel: CommandRiskMedium, Category: CommandCategoryBuild, NetworkPolicy: "deny", Justification: "known repository diagnostics command"},
		{Pattern: []string{"npm", "install"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskHigh, Category: CommandCategoryWrite, NetworkPolicy: "inherit", Justification: "dependency mutation"},
		{Pattern: []string{"pnpm", "add"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskHigh, Category: CommandCategoryWrite, NetworkPolicy: "inherit", Justification: "dependency mutation"},
		{Pattern: []string{"go", "get"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskHigh, Category: CommandCategoryWrite, NetworkPolicy: "inherit", Justification: "dependency mutation"},
		{Pattern: []string{"go", "mod", "tidy"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskHigh, Category: CommandCategoryWrite, NetworkPolicy: "inherit", Justification: "dependency mutation"},
		{Pattern: []string{"curl"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskHigh, Category: CommandCategoryNetwork, NetworkPolicy: "inherit", Justification: "network command"},
		{Pattern: []string{"wget"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskHigh, Category: CommandCategoryNetwork, NetworkPolicy: "inherit", Justification: "network command"},
		{Pattern: []string{"ssh"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskHigh, Category: CommandCategoryNetwork, NetworkPolicy: "inherit", Justification: "network command"},
		{Pattern: []string{"scp"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskHigh, Category: CommandCategoryNetwork, NetworkPolicy: "inherit", Justification: "network command"},
		{Pattern: []string{"rsync"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskHigh, Category: CommandCategoryNetwork, NetworkPolicy: "inherit", Justification: "network command"},
		{Pattern: []string{"git", "push"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskHigh, Category: CommandCategoryNetwork, NetworkPolicy: "inherit", Justification: "network write command"},
		{Pattern: []string{"npm", "publish"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskHigh, Category: CommandCategoryNetwork, NetworkPolicy: "inherit", Justification: "package publishing command"},
		{Pattern: []string{"gh", "release"}, Decision: CommandDecisionAsk, RiskLevel: CommandRiskHigh, Category: CommandCategoryNetwork, NetworkPolicy: "inherit", Justification: "release publishing command"},
	}
}

func tokensHavePrefix(tokens []string, prefix []string) bool {
	if len(prefix) == 0 || len(tokens) < len(prefix) {
		return false
	}
	for i, value := range prefix {
		if tokens[i] != value {
			return false
		}
	}
	return true
}

func policyDecisionRank(decision string) int {
	switch decision {
	case CommandDecisionDeny:
		return 3
	case CommandDecisionAsk:
		return 2
	case CommandDecisionAllow:
		return 1
	default:
		return 0
	}
}

func maxRisk(a string, b string) string {
	if riskRank(b) > riskRank(a) {
		return b
	}
	return a
}

func riskRank(risk string) int {
	switch risk {
	case CommandRiskCritical:
		return 4
	case CommandRiskHigh:
		return 3
	case CommandRiskMedium:
		return 2
	case CommandRiskLow:
		return 1
	default:
		return 0
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
