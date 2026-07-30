package domain

import (
	"errors"
	"strings"
)

const (
	AgentModeCode            = "code"
	AgentModeAssistant       = "assistant"
	AgentModeBuild           = "build"
	AgentModeExplore         = "explore"
	AgentModePlan            = "plan"
	AgentModePlanner         = "planner"
	AgentModeReview          = "review"
	AgentModeDebug           = "debug"
	AgentModeSummary         = "summary"
	AgentModeTitle           = "title"
	AgentModeSchedulerWorker = "scheduler_worker"

	AgentRunStatusRunning   = "running"
	AgentRunStatusCompleted = "completed"
	AgentRunStatusFailed    = "failed"
	AgentRunStatusCancelled = "cancelled"

	TodoStatusPending    = "pending"
	TodoStatusInProgress = "in_progress"
	TodoStatusCompleted  = "completed"
	TodoStatusCancelled  = "cancelled"

	ScheduledJobStatusActive    = "active"
	ScheduledJobStatusPaused    = "paused"
	ScheduledJobStatusRunning   = "running"
	ScheduledJobStatusCompleted = "completed"
	ScheduledJobStatusFailed    = "failed"
	ScheduledJobStatusCancelled = "cancelled"
)

type AgentModeDefinition struct {
	ID                   string         `json:"id"`
	DisplayName          string         `json:"displayName"`
	Description          string         `json:"description"`
	Prompt               string         `json:"prompt"`
	Toolsets             []string       `json:"toolsets"`
	DefaultPermissions   []string       `json:"defaultPermissions,omitempty"`
	FileWriteAccess      bool           `json:"fileWriteAccess"`
	CommandAccess        bool           `json:"commandAccess"`
	NetworkAccess        bool           `json:"networkAccess"`
	BackgroundTaskAccess bool           `json:"backgroundTaskAccess"`
	Hidden               bool           `json:"hidden,omitempty"`
	Model                *ModelRef      `json:"model,omitempty"`
	Temperature          *float64       `json:"temperature,omitempty"`
	TopP                 *float64       `json:"topP,omitempty"`
	MaxSteps             int            `json:"maxSteps,omitempty"`
	PermissionScope      string         `json:"permissionScope,omitempty"`
	Mode                 string         `json:"mode,omitempty"`
	Variant              string         `json:"variant,omitempty"`
	Options              map[string]any `json:"options,omitempty"`
	Revision             string         `json:"revision,omitempty"`
}

type SetSessionAgentModeInput struct {
	SessionID string `json:"sessionId"`
	Mode      string `json:"mode"`
}

type AgentRun struct {
	ID              string            `json:"id"`
	ParentSessionID string            `json:"parentSessionId,omitempty"`
	SessionID       string            `json:"sessionId,omitempty"`
	Mode            string            `json:"mode"`
	Status          string            `json:"status"`
	Prompt          string            `json:"prompt,omitempty"`
	Result          string            `json:"result,omitempty"`
	Error           string            `json:"error,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	TimeCreated     string            `json:"timeCreated"`
	TimeUpdated     string            `json:"timeUpdated"`
	TimeCompleted   string            `json:"timeCompleted,omitempty"`
}

type AgentRunRequest struct {
	ParentSessionID string            `json:"parentSessionId,omitempty"`
	SessionID       string            `json:"sessionId,omitempty"`
	Mode            string            `json:"mode,omitempty"`
	Status          string            `json:"status,omitempty"`
	Prompt          string            `json:"prompt,omitempty"`
	Result          string            `json:"result,omitempty"`
	Error           string            `json:"error,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

type AgentRunListRequest struct {
	SessionID string `json:"sessionId,omitempty"`
	Status    string `json:"status,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type TodoItem struct {
	ID            string            `json:"id"`
	SessionID     string            `json:"sessionId,omitempty"`
	ProjectPath   string            `json:"projectPath,omitempty"`
	Title         string            `json:"title"`
	Status        string            `json:"status"`
	Position      int               `json:"position,omitempty"`
	OwnerMode     string            `json:"ownerMode,omitempty"`
	SourceEventID string            `json:"sourceEventId,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	TimeCreated   string            `json:"timeCreated"`
	TimeUpdated   string            `json:"timeUpdated"`
}

type PlanStepInput struct {
	ID     string `json:"id,omitempty"`
	Step   string `json:"step"`
	Status string `json:"status"`
}

type UpdatePlanInput struct {
	SessionID     string            `json:"sessionId,omitempty"`
	ProjectPath   string            `json:"projectPath,omitempty"`
	Explanation   string            `json:"explanation,omitempty"`
	Plan          []PlanStepInput   `json:"plan"`
	OwnerMode     string            `json:"ownerMode,omitempty"`
	SourceEventID string            `json:"sourceEventId,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type TodoListInput struct {
	SessionID   string `json:"sessionId,omitempty"`
	ProjectPath string `json:"projectPath,omitempty"`
	Status      string `json:"status,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type ScheduledJob struct {
	ID              string            `json:"id"`
	SessionID       string            `json:"sessionId,omitempty"`
	Title           string            `json:"title"`
	Prompt          string            `json:"prompt"`
	Schedule        string            `json:"schedule"`
	WorkerMode      string            `json:"workerMode"`
	Toolsets        []string          `json:"toolsets,omitempty"`
	PermissionScope string            `json:"permissionScope,omitempty"`
	Status          string            `json:"status"`
	NextRunAt       string            `json:"nextRunAt,omitempty"`
	LastRunAt       string            `json:"lastRunAt,omitempty"`
	LastResult      string            `json:"lastResult,omitempty"`
	LastError       string            `json:"lastError,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	TimeCreated     string            `json:"timeCreated"`
	TimeUpdated     string            `json:"timeUpdated"`
}

type ScheduledJobInput struct {
	ID              string            `json:"id,omitempty"`
	SessionID       string            `json:"sessionId,omitempty"`
	Title           string            `json:"title,omitempty"`
	Prompt          string            `json:"prompt,omitempty"`
	Schedule        string            `json:"schedule,omitempty"`
	WorkerMode      string            `json:"workerMode,omitempty"`
	Toolsets        []string          `json:"toolsets,omitempty"`
	PermissionScope string            `json:"permissionScope,omitempty"`
	Status          string            `json:"status,omitempty"`
	NextRunAt       string            `json:"nextRunAt,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

type ScheduledJobListInput struct {
	SessionID string `json:"sessionId,omitempty"`
	Status    string `json:"status,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

func NormalizeAgentMode(value string) (string, error) {
	normalized := strings.TrimSpace(value)
	switch normalized {
	case "", AgentModeAssistant:
		return AgentModeAssistant, nil
	case AgentModeCode, AgentModeBuild, AgentModeExplore, AgentModePlan, AgentModePlanner, AgentModeReview, AgentModeDebug, AgentModeSummary, AgentModeTitle, AgentModeSchedulerWorker:
		return normalized, nil
	default:
		if validAgentModeIdentifier(normalized) {
			return normalized, nil
		}
		return "", errors.New("invalid agent mode")
	}
}

func validAgentModeIdentifier(value string) bool {
	if value == "" || len(value) > 128 || strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") || strings.Contains(value, "//") {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_' || char == '-' || char == '/' {
			continue
		}
		return false
	}
	return true
}

func NormalizeAgentRunStatus(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", AgentRunStatusRunning:
		return AgentRunStatusRunning, nil
	case AgentRunStatusCompleted, AgentRunStatusFailed, AgentRunStatusCancelled:
		return strings.TrimSpace(value), nil
	default:
		return "", errors.New("invalid agent run status")
	}
}

func NormalizeTodoStatus(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "":
		return TodoStatusPending, nil
	case TodoStatusPending, TodoStatusInProgress, TodoStatusCompleted, TodoStatusCancelled:
		return strings.TrimSpace(value), nil
	default:
		return "", errors.New("invalid todo status")
	}
}

func NormalizeScheduledJobStatus(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "":
		return ScheduledJobStatusActive, nil
	case ScheduledJobStatusActive, ScheduledJobStatusPaused, ScheduledJobStatusRunning, ScheduledJobStatusCompleted, ScheduledJobStatusFailed, ScheduledJobStatusCancelled:
		return strings.TrimSpace(value), nil
	default:
		return "", errors.New("invalid scheduled job status")
	}
}
