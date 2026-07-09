package domain

import (
	"errors"
	"strings"
)

func NormalizeSessionType(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", SessionTypeGeneric:
		return SessionTypeGeneric, nil
	case SessionTypeCoding:
		return SessionTypeCoding, nil
	default:
		return "", errors.New("invalid session type")
	}
}

func NormalizeSessionStatus(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", SessionStatusActive:
		return SessionStatusActive, nil
	case SessionStatusArchived:
		return SessionStatusArchived, nil
	case SessionStatusDeleted:
		return SessionStatusDeleted, nil
	default:
		return "", errors.New("invalid session status")
	}
}

func NormalizeSessionSource(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", SessionSourceDesktop:
		return SessionSourceDesktop, nil
	case SessionSourceWeb:
		return strings.TrimSpace(value), nil
	case legacyDesktopSessionSource:
		return SessionSourceDesktop, nil
	default:
		return "", errors.New("invalid session source")
	}
}

func ValidateTurnStatus(value string) error {
	switch value {
	case TurnStatusRunning, TurnStatusCompleted, TurnStatusFailed, TurnStatusCancelled:
		return nil
	default:
		return errors.New("invalid turn status")
	}
}

func ValidateEventType(value string) error {
	switch value {
	case EventTypeUserMessage, EventTypeAssistantMessage, EventTypeToolCall, EventTypeToolResult, EventTypeFileRead, EventTypeFileWrite, EventTypeFilePatch, EventTypeShellCommand, EventTypeShellOutput, EventTypeGitDiff, EventTypePlanUpdate, EventTypeSummary, EventTypeCheckpoint, EventTypeError, EventTypeSystemNote:
		return nil
	default:
		return errors.New("invalid event type")
	}
}

func NormalizeEventRole(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", EventRoleSystem:
		return strings.TrimSpace(value), nil
	case EventRoleUser, EventRoleAssistant, EventRoleTool:
		return strings.TrimSpace(value), nil
	default:
		return "", errors.New("invalid event role")
	}
}

func NormalizeEventVisibility(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", EventVisibilityNormal:
		return EventVisibilityNormal, nil
	case EventVisibilityHidden, EventVisibilityInternal, EventVisibilityRedacted:
		return strings.TrimSpace(value), nil
	default:
		return "", errors.New("invalid event visibility")
	}
}

func NormalizeToolCallStatus(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", ToolCallStatusRunning:
		return ToolCallStatusRunning, nil
	case ToolCallStatusSuccess, ToolCallStatusFailed, ToolCallStatusPending, ToolCallStatusInterrupted:
		return strings.TrimSpace(value), nil
	default:
		return "", errors.New("invalid tool call status")
	}
}

func NormalizeExecutionStatus(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", ExecutionStatusIdle:
		return ExecutionStatusIdle, nil
	case ExecutionStatusRunning, ExecutionStatusInterrupted, ExecutionStatusFailed, ExecutionStatusCompacting:
		return strings.TrimSpace(value), nil
	default:
		return "", errors.New("invalid execution status")
	}
}

func NormalizeInputDelivery(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", InputDeliveryImmediate:
		return InputDeliveryImmediate, nil
	case InputDeliverySteer, InputDeliveryQueue:
		return strings.TrimSpace(value), nil
	default:
		return "", errors.New("invalid input delivery")
	}
}
