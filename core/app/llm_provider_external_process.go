package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"aivo/core/domain"
)

const externalProviderOutputMaxBytes = 4 << 20

func callExternalProcessProvider(
	ctx context.Context,
	definition ProviderDefinition,
	model domain.ModelRef,
	messages []llmChatMessage,
	tools []domain.ToolSpec,
	reasoningEffort string,
	serviceTier string,
	onDelta func(string),
) (domain.ChatResponse, error) {
	if strings.TrimSpace(definition.Command) == "" {
		return domain.ChatResponse{}, errors.New("external provider command is not configured")
	}
	payload, err := json.Marshal(map[string]any{
		"model": model.ModelID, "messages": messages, "tools": tools,
		"reasoningEffort": reasoningEffort, "serviceTier": serviceTier,
	})
	if err != nil {
		return domain.ChatResponse{}, err
	}
	command := exec.CommandContext(ctx, definition.Command, definition.Args...)
	command.Stdin = bytes.NewReader(payload)
	command.Env = SanitizedEnvironment(".", defaultEnvAllowlist(), nil, nil)
	var stdout limitedProviderBuffer
	var stderr limitedProviderBuffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return domain.ChatResponse{}, fmt.Errorf("external provider failed: %s: %w", bounded(strings.TrimSpace(stderr.String()), 2000), err)
	}
	if stdout.overflow {
		return domain.ChatResponse{}, fmt.Errorf("external provider output exceeds %d bytes", externalProviderOutputMaxBytes)
	}
	var response domain.ChatResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return domain.ChatResponse{}, fmt.Errorf("decode external provider response: %w", err)
	}
	if strings.TrimSpace(response.Text) == "" && len(response.ToolCalls) == 0 {
		return domain.ChatResponse{}, errors.New("external provider returned an empty response")
	}
	if onDelta != nil && response.Text != "" {
		onDelta(response.Text)
	}
	return response, nil
}

type limitedProviderBuffer struct {
	buffer   bytes.Buffer
	overflow bool
}

func (b *limitedProviderBuffer) Write(input []byte) (int, error) {
	original := len(input)
	remaining := externalProviderOutputMaxBytes - b.buffer.Len()
	if remaining <= 0 {
		b.overflow = true
		return original, nil
	}
	if len(input) > remaining {
		b.overflow = true
		input = input[:remaining]
	}
	_, _ = b.buffer.Write(input)
	return original, nil
}

func (b *limitedProviderBuffer) Bytes() []byte  { return b.buffer.Bytes() }
func (b *limitedProviderBuffer) String() string { return b.buffer.String() }
