package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"aivo/core/domain"
)

const CodexViewImageToolName = "view_image"

type CodexViewImageTool struct {
	service *Service
}

func NewCodexViewImageTool(service *Service) *CodexViewImageTool {
	return &CodexViewImageTool{service: service}
}

func (t *CodexViewImageTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:              CodexViewImageToolName,
		Description:       "View a local image file from the active workspace when visual inspection is needed. Use this for images already available on disk.",
		Capability:        "filesystem.read.image",
		RiskLevel:         "low",
		Category:          "filesystem",
		Toolsets:          []string{"safe", "coding"},
		RequiresWorkspace: true,
		ActivationPolicy:  providerAccountActivationPolicy,
		InputSchema: map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"path": map[string]any{"type": "string", "minLength": 1, "description": workspaceRelativePathDescription},
			},
			"required": []string{"path"},
		},
	}
}

func (t *CodexViewImageTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	if t == nil || t.service == nil || execCtx.ActiveModel == nil {
		return toolFailure(execCtx.ToolCallID, CodexViewImageToolName, "codex_image_view_unavailable", "Codex local image inspection is unavailable")
	}
	var input struct {
		Path string `json:"path"`
	}
	if err := decodeStrictToolArgs(args, &input); err != nil {
		return toolFailure(execCtx.ToolCallID, CodexViewImageToolName, "invalid_arguments", "invalid image inspection arguments")
	}
	input.Path = strings.TrimSpace(input.Path)
	if input.Path == "" {
		return toolFailure(execCtx.ToolCallID, CodexViewImageToolName, "invalid_arguments", "path is required")
	}

	cfg, err := t.service.AppConfig(ctx)
	if err != nil {
		return toolErrorWithCallID(CodexViewImageToolName, execCtx.ToolCallID, err)
	}
	if nativeToolDisabled(normalizeNativeToolsRuntimeConfig(cfg.NativeTools), CodexViewImageToolName) {
		return toolFailure(execCtx.ToolCallID, CodexViewImageToolName, "codex_image_view_disabled", "Codex local image inspection is disabled")
	}
	route, err := t.service.ResolveModelRoute(ctx, cfg, execCtx.ActiveModel)
	if err != nil || !capabilitiesForProviderAccount(route).LocalImageView {
		return toolFailure(execCtx.ToolCallID, CodexViewImageToolName, "codex_account_required", "active model is not an authenticated ChatGPT Codex route")
	}
	if model, known := t.service.modelInfoForRoute(ctx, route); known && len(model.Modalities) > 0 && !containsString(model.Modalities, "image") {
		return toolFailure(execCtx.ToolCallID, CodexViewImageToolName, "model_image_unsupported", "selected Codex model does not accept image input")
	}

	root := toolWorkspaceRoot("", execCtx)
	if strings.TrimSpace(root) == "" {
		return toolFailure(execCtx.ToolCallID, CodexViewImageToolName, "workspace_required", "an active workspace is required to view an image")
	}
	if isSensitiveRelPath(input.Path) {
		return toolFailure(execCtx.ToolCallID, CodexViewImageToolName, "sensitive_file", "refusing to read sensitive file")
	}
	target, err := safeJoin(root, input.Path)
	if err != nil {
		return toolFailure(execCtx.ToolCallID, CodexViewImageToolName, "invalid_path", err.Error())
	}
	info, err := os.Stat(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return toolFailure(execCtx.ToolCallID, CodexViewImageToolName, "not_found", "image file was not found")
		}
		return toolFailure(execCtx.ToolCallID, CodexViewImageToolName, "read_failed", "unable to read image file")
	}
	if !info.Mode().IsRegular() {
		return toolFailure(execCtx.ToolCallID, CodexViewImageToolName, "not_file", "image path is not a regular file")
	}
	if !looksLikeSupportedImage(target) {
		return toolFailure(execCtx.ToolCallID, CodexViewImageToolName, "unsupported_image", "image format is not supported")
	}
	result := readPrimitiveImage(ctx, input.Path, target, info)
	result.Name = CodexViewImageToolName
	result.CallID = execCtx.ToolCallID
	if result.OK {
		result.Content = fmt.Sprintf("Viewed workspace image %s", input.Path)
		result.ModelContent = result.Content
	}
	return result
}
