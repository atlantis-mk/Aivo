package domain

import (
	"context"
	"encoding/json"
)

type ToolSpec struct {
	Name                 string              `json:"name"`
	Description          string              `json:"description"`
	InputSchema          map[string]any      `json:"inputSchema"`
	Hosted               *HostedToolSpec     `json:"hosted,omitempty"`
	Kind                 string              `json:"kind,omitempty"`
	Format               *ToolFormat         `json:"format,omitempty"`
	Strict               *bool               `json:"strict,omitempty"`
	Namespace            string              `json:"namespace,omitempty"`
	NamespaceDescription string              `json:"namespaceDescription,omitempty"`
	Capability           string              `json:"capability,omitempty"`
	RiskLevel            string              `json:"riskLevel,omitempty"`
	Category             string              `json:"category,omitempty"`
	Toolsets             []string            `json:"toolsets,omitempty"`
	RequiresWorkspace    bool                `json:"requiresWorkspace,omitempty"`
	RequiresNetwork      bool                `json:"requiresNetwork,omitempty"`
	TouchesSecrets       bool                `json:"touchesSecrets,omitempty"`
	ActivationPolicy     string              `json:"activationPolicy,omitempty"`
	SelectionGroup       *ToolSelectionGroup `json:"selectionGroup,omitempty"`
	ImplementationHash   string              `json:"-"`
}

type ToolSelectionGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type HostedToolSpec struct {
	Type               string                 `json:"type"`
	ExternalWebAccess  *bool                  `json:"externalWebAccess,omitempty"`
	IndexedWebAccess   *bool                  `json:"indexedWebAccess,omitempty"`
	SearchContextSize  string                 `json:"searchContextSize,omitempty"`
	SearchContentTypes []string               `json:"searchContentTypes,omitempty"`
	AllowedDomains     []string               `json:"allowedDomains,omitempty"`
	UserLocation       *WebSearchUserLocation `json:"userLocation,omitempty"`
	MaxUses            int                    `json:"maxUses,omitempty"`
	VectorStoreIDs     []string               `json:"vectorStoreIds,omitempty"`
	FileIDs            []string               `json:"fileIds,omitempty"`
	ContainerID        string                 `json:"containerId,omitempty"`
	ServerURL          string                 `json:"serverUrl,omitempty"`
	ServerLabel        string                 `json:"serverLabel,omitempty"`
	AllowedTools       []string               `json:"allowedTools,omitempty"`
}

type WebSearchUserLocation struct {
	Type     string `json:"type,omitempty"`
	Country  string `json:"country,omitempty"`
	Region   string `json:"region,omitempty"`
	City     string `json:"city,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

type ToolFormat struct {
	Type       string `json:"type"`
	Syntax     string `json:"syntax,omitempty"`
	Definition string `json:"definition,omitempty"`
}

const (
	ToolKindJSON     = "json"
	ToolKindFreeform = "freeform"
)

type ChatToolCall struct {
	ID        string          `json:"id"`
	Namespace string          `json:"namespace,omitempty"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ChatMessage struct {
	Role        string              `json:"role"`
	Text        string              `json:"text"`
	Attachments []MessageAttachment `json:"attachments,omitempty"`
	ToolCalls   []ChatToolCall      `json:"toolCalls,omitempty"`
	ToolCallID  string              `json:"toolCallId,omitempty"`
	Name        string              `json:"name,omitempty"`
}

type Message = ChatMessage

type MessageAttachment struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	MIMEType string `json:"mimeType,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Data     string `json:"data,omitempty"`
	Text     string `json:"text,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

type ToolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Retry   bool   `json:"retry,omitempty"`
}

type ToolResult struct {
	CallID              string              `json:"call_id"`
	Name                string              `json:"name"`
	OK                  bool                `json:"ok"`
	Content             string              `json:"content,omitempty"`
	ModelContent        string              `json:"modelContent,omitempty"`
	ModelAttachments    []MessageAttachment `json:"-"`
	Structured          map[string]any      `json:"structured,omitempty"`
	Details             *ToolResultDetails  `json:"-"`
	RetainedOutputRefs  []string            `json:"retainedOutputRefs,omitempty"`
	Files               []ToolResultFile    `json:"files,omitempty"`
	Error               string              `json:"error,omitempty"`
	ToolError           *ToolError          `json:"toolError,omitempty"`
	Truncated           bool                `json:"truncated,omitempty"`
	OriginalSize        int                 `json:"originalSize,omitempty"`
	PendingApprovalID   string              `json:"pendingApprovalId,omitempty"`
	PermissionDecision  string              `json:"permissionDecision,omitempty"`
	PermissionRequested bool                `json:"permissionRequested,omitempty"`
}

type ToolResultDetails struct {
	View *ExtensionToolViewRef `json:"view,omitempty"`
}

type ToolResultFile struct {
	Path         string `json:"path"`
	FullPath     string `json:"fullPath,omitempty"`
	MovePath     string `json:"movePath,omitempty"`
	MoveFullPath string `json:"moveFullPath,omitempty"`
	Type         string `json:"type"`
	Additions    int    `json:"additions"`
	Deletions    int    `json:"deletions"`
	Diff         string `json:"diff,omitempty"`
	BaseHash     string `json:"baseHash,omitempty"`
	CurrentHash  string `json:"currentHash,omitempty"`
	Stale        bool   `json:"stale,omitempty"`
}

type OutputPolicy struct {
	MaxChars         int    `json:"maxChars,omitempty"`
	MaxLines         int    `json:"maxLines,omitempty"`
	TruncationMarker string `json:"truncationMarker,omitempty"`
}

type RetainedOutputReadInput struct {
	Ref    string `json:"ref"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type RetainedOutputReadResult struct {
	Ref        string `json:"ref"`
	Content    string `json:"content"`
	Offset     int    `json:"offset"`
	NextOffset int    `json:"nextOffset"`
	Size       int    `json:"size"`
	Truncated  bool   `json:"truncated,omitempty"`
}

type ToolExecutionContext struct {
	WorkspaceRoot         string                              `json:"workspaceRoot,omitempty"`
	SessionID             string                              `json:"sessionId,omitempty"`
	TurnID                string                              `json:"turnId,omitempty"`
	ToolCallID            string                              `json:"toolCallId,omitempty"`
	AgentMode             string                              `json:"agentMode,omitempty"`
	AllowedToolsets       []string                            `json:"allowedToolsets,omitempty"`
	PermissionScope       string                              `json:"permissionScope,omitempty"`
	OutputPolicy          OutputPolicy                        `json:"outputPolicy,omitempty"`
	ExpectedRegistrations map[string]ToolRegistrationIdentity `json:"expectedRegistrations,omitempty"`
	ToolSnapshot          *ToolSnapshot                       `json:"toolSnapshot,omitempty"`
	BridgeCallDepth       int                                 `json:"bridgeCallDepth,omitempty"`
	ActiveModel           *ModelRef                           `json:"activeModel,omitempty"`
	RecentImages          []MessageAttachment                 `json:"-"`
}

type Tool interface {
	Spec() ToolSpec
	Execute(ctx context.Context, args json.RawMessage, execCtx ToolExecutionContext) ToolResult
}

type ChatRequest struct {
	Messages    []ChatMessage  `json:"messages"`
	Tools       []ToolSpec     `json:"tools,omitempty"`
	Temperature *float64       `json:"temperature,omitempty"`
	TopP        *float64       `json:"topP,omitempty"`
	Options     map[string]any `json:"options,omitempty"`
}

type ChatResponse struct {
	Text      string         `json:"text"`
	ToolCalls []ChatToolCall `json:"toolCalls,omitempty"`
	Usage     *TokenUsage    `json:"usage,omitempty"`
	Sources   []ChatSource   `json:"sources,omitempty"`
}

type ChatSource struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
	RefID string `json:"refId,omitempty"`
}

type TokenUsage struct {
	InputTokens               int  `json:"inputTokens,omitempty"`
	OutputTokens              int  `json:"outputTokens,omitempty"`
	TotalTokens               int  `json:"totalTokens,omitempty"`
	CacheReadTokens           int  `json:"cacheReadTokens,omitempty"`
	CacheWriteTokens          int  `json:"cacheWriteTokens,omitempty"`
	ReasoningTokens           int  `json:"reasoningTokens,omitempty"`
	InputTokensAvailable      bool `json:"inputTokensAvailable,omitempty"`
	OutputTokensAvailable     bool `json:"outputTokensAvailable,omitempty"`
	TotalTokensAvailable      bool `json:"totalTokensAvailable,omitempty"`
	CacheReadTokensAvailable  bool `json:"cacheReadTokensAvailable,omitempty"`
	CacheWriteTokensAvailable bool `json:"cacheWriteTokensAvailable,omitempty"`
	ReasoningTokensAvailable  bool `json:"reasoningTokensAvailable,omitempty"`
	Estimated                 bool `json:"estimated,omitempty"`
}

type StreamingEvent struct {
	Type       string        `json:"type"`
	Delta      string        `json:"delta,omitempty"`
	ToolCall   *ChatToolCall `json:"toolCall,omitempty"`
	ToolResult *ToolResult   `json:"toolResult,omitempty"`
	Error      *ToolError    `json:"error,omitempty"`
}
