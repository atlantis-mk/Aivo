package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"aivo/core/domain"
)

type bashInput struct {
	Command        string            `json:"command"`
	Timeout        int               `json:"timeout"`
	CWD            string            `json:"cwd"`
	TimeoutSeconds int               `json:"timeoutSeconds"`
	Network        string            `json:"network"`
	Mode           string            `json:"mode"`
	Stdin          string            `json:"stdin"`
	Env            map[string]string `json:"env"`
	Justification  string            `json:"justification"`
}

func parseBashArgs(args json.RawMessage) (bashInput, error) {
	var input bashInput
	if err := decodeStrictToolArgs(args, &input); err != nil {
		return input, errors.New("invalid bash arguments")
	}
	input.Command = strings.TrimSpace(input.Command)
	input.CWD = strings.TrimSpace(input.CWD)
	input.Network = strings.TrimSpace(input.Network)
	input.Mode = normalizeSandboxMode(input.Mode)
	input.Justification = strings.TrimSpace(input.Justification)
	if input.Command == "" {
		return input, errors.New("command is required")
	}
	if input.Network != "" && input.Network != "deny" && input.Network != "inherit" {
		return input, errors.New("network must be deny or inherit")
	}
	if input.Mode != "foreground" && input.Mode != "background" && input.Mode != "pty" {
		return input, errors.New("mode must be foreground, background, or pty")
	}
	for key := range input.Env {
		if isSecretEnvName(key) {
			return input, fmt.Errorf("env override %s is denied because it looks secret-bearing", key)
		}
	}
	return input, nil
}

func parsePrimitiveBashArgs(args json.RawMessage) (bashInput, error) {
	var input struct {
		Command string `json:"command"`
		Timeout int    `json:"timeout"`
	}
	if err := decodeStrictToolArgs(args, &input); err != nil {
		return bashInput{}, errors.New("invalid bash arguments")
	}
	input.Command = strings.TrimSpace(input.Command)
	if input.Command == "" {
		return bashInput{}, errors.New("command is required")
	}
	if input.Timeout < 0 || input.Timeout > int(maxCommandTimeout.Seconds()) {
		return bashInput{}, fmt.Errorf("timeout must be between 1 and %d seconds", int(maxCommandTimeout.Seconds()))
	}
	return bashInput{Command: input.Command, Timeout: input.Timeout, TimeoutSeconds: input.Timeout, Mode: "foreground"}, nil
}

func shouldUsePersistentAgentShell(input bashInput, execCtx domain.ToolExecutionContext) bool {
	return strings.TrimSpace(execCtx.SessionID) != "" &&
		input.Mode == "foreground" &&
		input.Stdin == "" &&
		len(input.Env) == 0
}

type runTestsInput struct {
	Target         string `json:"target"`
	Kind           string `json:"kind"`
	Filter         string `json:"filter"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
}

func parseRunTestsArgs(args json.RawMessage) (runTestsInput, error) {
	var input runTestsInput
	if len(args) > 0 {
		if err := json.Unmarshal(args, &input); err != nil {
			return input, errors.New("invalid run_tests arguments")
		}
	}
	input.Target = firstNonEmpty(strings.TrimSpace(input.Target), "all")
	input.Kind = firstNonEmpty(strings.TrimSpace(input.Kind), "auto")
	input.Filter = strings.TrimSpace(input.Filter)
	switch input.Target {
	case "core", "desktop", "all":
	default:
		return input, fmt.Errorf("unsupported run_tests target %q", input.Target)
	}
	switch input.Kind {
	case "test", "lint", "build", "auto":
	default:
		return input, fmt.Errorf("unsupported run_tests kind %q", input.Kind)
	}
	return input, nil
}

func runTestsCommands(input runTestsInput) ([]string, error) {
	target := input.Target
	kind := input.Kind
	if kind == "auto" {
		switch target {
		case "desktop":
			kind = "lint"
		default:
			kind = "test"
		}
	}
	if strings.TrimSpace(input.Filter) != "" {
		return nil, errors.New("run_tests filter is not supported by the initial command mapping")
	}
	switch target + ":" + kind {
	case "core:test", "all:test":
		return []string{"npm run test:core"}, nil
	case "desktop:lint":
		return []string{"npm run lint"}, nil
	case "desktop:build", "all:build":
		return []string{"npm run build"}, nil
	}
	return nil, fmt.Errorf("unsupported run_tests combination target=%s kind=%s", target, kind)
}
