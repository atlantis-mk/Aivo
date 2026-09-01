package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

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
