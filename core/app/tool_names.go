package app

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"aivo/core/domain"
)

const toolNameMaxLength = 64

func providerSafeToolName(name string) bool {
	if name == "" || len(name) > toolNameMaxLength {
		return false
	}
	for index := 0; index < len(name); index++ {
		current := name[index]
		if (current >= 'a' && current <= 'z') || (current >= 'A' && current <= 'Z') || (current >= '0' && current <= '9') || current == '_' || current == '-' {
			continue
		}
		return false
	}
	return true
}

func validateCanonicalToolName(name string) error {
	if !providerSafeToolName(name) {
		return fmt.Errorf("tool name %q must match ^[A-Za-z0-9_-]+$ and be at most %d bytes", name, toolNameMaxLength)
	}
	return nil
}

func validateProviderToolIdentities(specs []domain.ToolSpec, messages []llmChatMessage) error {
	for _, spec := range specs {
		if spec.Hosted == nil {
			if err := validateCanonicalToolName(spec.Name); err != nil {
				return err
			}
		}
	}
	for _, message := range messages {
		if message.Name != "" {
			if err := validateCanonicalToolName(message.Name); err != nil {
				return fmt.Errorf("historical tool result: %w", err)
			}
		}
		for _, call := range message.ToolCalls {
			if err := validateCanonicalToolName(call.Name); err != nil {
				return fmt.Errorf("historical tool call: %w", err)
			}
		}
	}
	return nil
}

func generatedToolName(parts ...string) string {
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		if component := sanitizeMCPExtensionNameComponent(part); component != "" {
			normalized = append(normalized, component)
		}
	}
	name := strings.Join(normalized, "_")
	if name == "" {
		name = "tool"
	}
	if len(name) <= toolNameMaxLength {
		return name
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	suffix := "_" + hex.EncodeToString(digest[:])[:12]
	prefix := strings.TrimRight(name[:toolNameMaxLength-len(suffix)], "_-")
	if prefix == "" {
		prefix = "tool"
	}
	return prefix + suffix
}
