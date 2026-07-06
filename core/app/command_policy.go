package app

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
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
	normalized := normalizeCommandText(rawCommand)
	detection := CommandDetection{
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

	paths, external := commandPathHints(tokens, cwd, workspaceRoot)
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
	detection.ApprovalKey = commandApprovalKey(workspaceRoot, cwd, normalized, tokens, toolName, "local", "default", detection.NetworkHint, detection.Category, detection.RiskLevel, detection.Capabilities)
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
	if toolName == "bash" && detection.HasMetacharacters && evaluation.Decision == CommandDecisionAllow {
		evaluation.Decision = CommandDecisionAsk
		evaluation.RiskLevel = maxRisk(evaluation.RiskLevel, CommandRiskHigh)
		evaluation.Justification = "shell metacharacters require explicit approval"
	}
	if detection.Category == CommandCategoryNetwork && evaluation.NetworkPolicy == "deny" && toolName == "bash" {
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

func classifyCommand(detection *CommandDetection) {
	if len(detection.Argv) == 0 {
		return
	}
	switch detection.Argv[0] {
	case "git":
		if len(detection.Argv) > 1 {
			switch detection.Argv[1] {
			case "status", "diff", "log":
				detection.Category = CommandCategoryRead
				detection.RiskLevel = CommandRiskLow
				detection.NetworkHint = "deny"
				detection.Reason = "read-only git command"
			case "push":
				detection.Category = CommandCategoryNetwork
				detection.RiskLevel = CommandRiskHigh
				detection.NetworkHint = "inherit"
				detection.Reason = "network write command"
			}
		}
	case "pwd", "ls", "find":
		detection.Category = CommandCategoryRead
		detection.RiskLevel = CommandRiskLow
		detection.NetworkHint = "deny"
		detection.Reason = "read-only command"
	case "go":
		if len(detection.Argv) > 1 && detection.Argv[1] == "test" {
			detection.Category = CommandCategoryTest
			detection.RiskLevel = CommandRiskMedium
			detection.NetworkHint = "deny"
			detection.Reason = "known test command"
		} else if len(detection.Argv) > 1 && detection.Argv[1] == "get" || len(detection.Argv) > 2 && detection.Argv[1] == "mod" && detection.Argv[2] == "tidy" {
			detection.Category = CommandCategoryWrite
			detection.RiskLevel = CommandRiskHigh
			detection.NetworkHint = "inherit"
			detection.Reason = "dependency mutation command"
		}
	case "gofmt", "rustfmt", "shfmt":
		detection.Category = CommandCategoryWrite
		detection.RiskLevel = CommandRiskMedium
		detection.NetworkHint = "deny"
		detection.Reason = "formatter writes source files"
	case "npx":
		if len(detection.Argv) > 3 && detection.Argv[1] == "--no-install" && detection.Argv[2] == "prettier" && detection.Argv[3] == "--write" {
			detection.Category = CommandCategoryWrite
			detection.RiskLevel = CommandRiskMedium
			detection.NetworkHint = "deny"
			detection.Reason = "project-local formatter writes source files"
		} else if len(detection.Argv) > 3 && detection.Argv[1] == "--no-install" && detection.Argv[2] == "eslint" && detection.Argv[3] == "--fix" {
			detection.Category = CommandCategoryWrite
			detection.RiskLevel = CommandRiskMedium
			detection.NetworkHint = "deny"
			detection.Reason = "project-local lint fixer writes source files"
		}
	case "python":
		if len(detection.Argv) > 2 && detection.Argv[1] == "-m" && detection.Argv[2] == "black" {
			detection.Category = CommandCategoryWrite
			detection.RiskLevel = CommandRiskMedium
			detection.NetworkHint = "deny"
			detection.Reason = "formatter writes source files"
		}
	case "npm":
		if len(detection.Argv) > 2 && detection.Argv[1] == "run" {
			switch detection.Argv[2] {
			case "test:core", "lint":
				detection.Category = CommandCategoryTest
				detection.RiskLevel = CommandRiskMedium
				detection.NetworkHint = "deny"
				detection.Reason = "known test or lint command"
			case "build", "diagnostics":
				detection.Category = CommandCategoryBuild
				detection.RiskLevel = CommandRiskMedium
				detection.NetworkHint = "deny"
				detection.Reason = "known build or diagnostics command"
			}
		} else if len(detection.Argv) > 1 && detection.Argv[1] == "install" {
			detection.Category = CommandCategoryWrite
			detection.RiskLevel = CommandRiskHigh
			detection.NetworkHint = "inherit"
			detection.Reason = "dependency mutation command"
		} else if len(detection.Argv) > 1 && detection.Argv[1] == "publish" {
			detection.Category = CommandCategoryNetwork
			detection.RiskLevel = CommandRiskHigh
			detection.NetworkHint = "inherit"
			detection.Reason = "package publishing command"
		}
	case "curl", "wget", "ssh", "scp", "rsync":
		detection.Category = CommandCategoryNetwork
		detection.RiskLevel = CommandRiskHigh
		detection.NetworkHint = "inherit"
		detection.Reason = "network command"
	}
	if detection.HasMetacharacters {
		detection.RiskLevel = maxRisk(detection.RiskLevel, CommandRiskHigh)
		detection.Reason = firstNonEmpty(detection.Reason, "shell metacharacters present")
	}
}

func normalizeCommandText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func shellTokenize(command string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	var quote rune
	escaped := false
	for _, r := range command {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			continue
		}
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(r)
	}
	if escaped {
		current.WriteRune('\\')
	}
	if quote != 0 {
		return nil, errors.New("unterminated shell quote")
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens, nil
}

func commandHasMetacharacters(command string) bool {
	for _, marker := range []string{"|", ">", "<", "$(", "`", "&&", "||", ";", "&"} {
		if strings.Contains(command, marker) {
			return true
		}
	}
	return false
}

func hardlineCommandDenyReason(tokens []string, command string) string {
	if len(tokens) == 0 {
		return ""
	}
	if tokens[0] == "sudo" {
		if commandContainsSudoPasswordPipe(command, tokens) {
			return "sudo password piping is blocked"
		}
		return ""
	}
	if commandContainsSudoPasswordPipe(command, tokens) {
		return "sudo password piping is blocked"
	}
	if strings.Contains(command, ":(){ :|:& };:") {
		return "fork bombs are blocked"
	}
	if len(tokens) >= 3 && tokens[0] == "rm" && tokens[1] == "-rf" && (tokens[2] == "/" || tokens[2] == "." || tokens[2] == "./") {
		return "destructive rm command is blocked"
	}
	if len(tokens) >= 3 && tokens[0] == "chmod" && tokens[1] == "-R" && tokens[2] == "777" {
		return "broad world-writable chmod is blocked"
	}
	if strings.HasPrefix(tokens[0], "mkfs") || tokens[0] == "diskutil" {
		return "disk formatting commands are blocked"
	}
	for _, token := range tokens[1:] {
		clean := filepath.ToSlash(filepath.Clean(token))
		if clean == ".git" || strings.HasPrefix(clean, ".git/") || strings.Contains(clean, "/.git/") {
			return "commands targeting .git internals are blocked"
		}
	}
	return ""
}

func isSudoCommand(tokens []string) bool {
	if len(tokens) == 0 {
		return false
	}
	switch tokens[0] {
	case "sudo", "su", "doas":
		return true
	default:
		return false
	}
}

func commandContainsSudoPasswordPipe(command string, tokens []string) bool {
	if strings.Contains(command, "sudo -S") || strings.Contains(command, "| sudo") {
		return true
	}
	for _, token := range tokens {
		if token == "-S" || token == "--stdin" {
			return true
		}
	}
	return false
}

func appendUniqueStrings(values []string, next ...string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values)+len(next))
	for _, value := range append(values, next...) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func commandPathHints(tokens []string, cwd string, workspaceRoot string) ([]string, []string) {
	seenPaths := map[string]bool{}
	seenExternal := map[string]bool{}
	root, _ := filepath.Abs(strings.TrimSpace(workspaceRoot))
	base := cwd
	if strings.TrimSpace(base) == "" {
		base = root
	}
	for _, token := range tokens[1:] {
		if !looksLikePathToken(token) {
			continue
		}
		path := token
		if strings.Contains(path, "=") {
			continue
		}
		var abs string
		if filepath.IsAbs(path) {
			abs = filepath.Clean(path)
		} else {
			abs = filepath.Clean(filepath.Join(base, path))
		}
		if pathHasPrefix(abs, root) {
			rel, err := filepath.Rel(root, abs)
			if err == nil && rel != "." {
				seenPaths[filepath.ToSlash(rel)] = true
			}
			continue
		}
		if filepath.IsAbs(path) || strings.HasPrefix(filepath.Clean(path), "..") {
			seenExternal[filepath.ToSlash(path)] = true
		}
	}
	return sortedMapKeys(seenPaths), sortedMapKeys(seenExternal)
}

func looksLikePathToken(token string) bool {
	if strings.TrimSpace(token) == "" || strings.HasPrefix(token, "-") {
		return false
	}
	return strings.Contains(token, "/") || strings.HasPrefix(token, ".")
}

func commandApprovalKey(workspaceRoot string, cwd string, command string, argv []string, toolName string, backend string, sandboxProfile string, networkPolicy string, category string, riskLevel string, capabilities []string) string {
	caps := append([]string(nil), capabilities...)
	sort.Strings(caps)
	parts := []string{
		"workspace=" + normalizeStoredPathForKey(workspaceRoot),
		"cwd=" + normalizeStoredPathForKey(cwd),
		"command=" + command,
		"argv=" + strings.Join(argv, "\x00"),
		"tool=" + toolName,
		"backend=" + backend,
		"sandbox=" + firstNonEmpty(sandboxProfile, "default"),
		"network=" + networkPolicy,
		"category=" + category,
		"risk=" + riskLevel,
		"capabilities=" + strings.Join(caps, ","),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return "shell:" + hex.EncodeToString(sum[:])
}

func normalizeStoredPathForKey(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(value))
	}
	return filepath.ToSlash(filepath.Clean(abs))
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

func sortedMapKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

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
