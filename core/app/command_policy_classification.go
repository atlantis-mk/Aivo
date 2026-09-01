package app

import "strings"

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
