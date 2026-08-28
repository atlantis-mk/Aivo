package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"aivo/core/domain"
)

const (
	CodexImagegenToolName             = "imagegen"
	codexImagegenNamespace            = "image_gen"
	codexImagegenNamespaceDescription = "Tools in the image_gen namespace."
	codexImageModel                   = "gpt-image-2"
	codexImageMaxEditImages           = 5
	codexImageMaxGeneratedBytes       = 32 * 1024 * 1024
	codexImageMaxReferenceBytes       = 20 * 1024 * 1024
	codexImageMaxReferenceTotal       = 50 * 1024 * 1024
	codexImageMaxResponseBytes        = 48 * 1024 * 1024
	chatGPTCodexImageGenerationURL    = "https://chatgpt.com/backend-api/codex/images/generations"
	chatGPTCodexImageEditURL          = "https://chatgpt.com/backend-api/codex/images/edits"
)

const codexImagegenDescription = `The ` + "`" + `image_gen.imagegen` + "`" + ` tool enables image generation from descriptions and editing of existing images based on specific instructions. Use it when:

- The user requests an image based on a scene description, such as a diagram, portrait, comic, meme, or any other visual.
- The user wants to modify an attached or previously generated image with specific changes, including adding or removing elements, altering colors, improving quality/resolution, or transforming the style (e.g., cartoon, oil painting).

Guidelines:
- imagegen needs a few minutes to finish. In code-mode, use the first-line @exec directive to give the initial call 120 seconds and the same yield for any waits that follow. Once it finishes, return the image with generatedImage(result).
- Omit both ` + "`" + `referenced_image_paths` + "`" + ` and ` + "`" + `num_last_images_to_include` + "`" + ` when generating a brand new image.
- For edits, use ` + "`" + `referenced_image_paths` + "`" + ` when every target image has a local file path.
- If you have not seen a local image yet, use ` + "`" + `view_image` + "`" + ` to inspect it before editing.
- Use ` + "`" + `num_last_images_to_include` + "`" + ` only when at least one target image has no local file path.
- Set ` + "`" + `num_last_images_to_include` + "`" + ` to the smallest number of recent conversation images that includes every target image, up to 5.
- Never provide both ` + "`" + `referenced_image_paths` + "`" + ` and ` + "`" + `num_last_images_to_include` + "`" + `.
- If neither mechanism can include every target image, ask the user to attach the missing images again.
- Directly generate the image without reconfirmation or clarification unless required images must be attached again.
- Always use this tool for image editing unless the user explicitly requests otherwise. Do not use the ` + "`" + `python` + "`" + ` tool for image editing unless specifically instructed.
`

const codexAbsolutePathSchemaDescription = "A path that is guaranteed to be absolute and normalized (though it is not guaranteed to be canonicalized or exist on the filesystem).\n\nIMPORTANT: When deserializing an `AbsolutePathBuf`, a base path must be set using [AbsolutePathBufGuard::new]. If no base path is set, the deserialization will fail unless the path being deserialized is already absolute."

var codexImageHTTPClient = &http.Client{Timeout: 3 * time.Minute}

type CodexImageGenerationTool struct {
	service *Service
}

func NewCodexImageGenerationTool(service *Service) *CodexImageGenerationTool {
	return &CodexImageGenerationTool{service: service}
}

func (t *CodexImageGenerationTool) Spec() domain.ToolSpec {
	strict := false
	return domain.ToolSpec{
		Name:                 CodexImagegenToolName,
		Namespace:            codexImagegenNamespace,
		NamespaceDescription: codexImagegenNamespaceDescription,
		Description:          codexImagegenDescription,
		Strict:               &strict,
		Capability:           "image.generate", RiskLevel: "medium", Category: "image_generation", Toolsets: []string{"safe", "coding"},
		RequiresNetwork: true, RequiresWorkspace: true, TouchesSecrets: true, ActivationPolicy: providerAccountActivationPolicy,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"prompt": map[string]any{"type": "string"},
				"referenced_image_paths": map[string]any{
					"type":  []string{"array", "null"},
					"items": map[string]any{"type": "string", "description": codexAbsolutePathSchemaDescription},
				},
				"num_last_images_to_include": map[string]any{"type": []string{"integer", "null"}},
			},
			"required":             []string{"prompt"},
			"additionalProperties": false,
		},
	}
}

type codexImagegenInput struct {
	Prompt                 string   `json:"prompt"`
	ReferencedImagePaths   []string `json:"referenced_image_paths"`
	NumLastImagesToInclude int      `json:"num_last_images_to_include"`
}

type codexImageResponse struct {
	Created    uint64 `json:"created"`
	Background string `json:"background"`
	Quality    string `json:"quality"`
	Size       string `json:"size"`
	Data       []struct {
		Base64 string `json:"b64_json"`
	} `json:"data"`
}

func (t *CodexImageGenerationTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	if t == nil || t.service == nil || execCtx.ActiveModel == nil {
		return toolFailure(execCtx.ToolCallID, CodexImagegenToolName, "codex_image_unavailable", "Codex image generation route is unavailable")
	}
	var input codexImagegenInput
	if err := json.Unmarshal(args, &input); err != nil {
		return toolFailure(execCtx.ToolCallID, CodexImagegenToolName, "invalid_arguments", "invalid image generation arguments")
	}
	input.Prompt = strings.TrimSpace(input.Prompt)
	if input.Prompt == "" {
		return toolFailure(execCtx.ToolCallID, CodexImagegenToolName, "invalid_arguments", "prompt is required")
	}
	if len(input.ReferencedImagePaths) > codexImageMaxEditImages {
		return toolFailure(execCtx.ToolCallID, CodexImagegenToolName, "invalid_arguments", "referenced_image_paths must contain at most five paths")
	}
	if len(input.ReferencedImagePaths) > 0 && input.NumLastImagesToInclude > 0 {
		return toolFailure(execCtx.ToolCallID, CodexImagegenToolName, "invalid_arguments", "provide only one of referenced_image_paths or num_last_images_to_include")
	}
	if input.NumLastImagesToInclude < 0 || input.NumLastImagesToInclude > codexImageMaxEditImages {
		return toolFailure(execCtx.ToolCallID, CodexImagegenToolName, "invalid_arguments", "num_last_images_to_include must be between one and five")
	}

	cfg, err := t.service.AppConfig(ctx)
	if err != nil {
		return toolErrorWithCallID(CodexImagegenToolName, execCtx.ToolCallID, err)
	}
	if nativeToolDisabled(normalizeNativeToolsRuntimeConfig(cfg.NativeTools), CodexImagegenToolName) {
		return toolFailure(execCtx.ToolCallID, CodexImagegenToolName, "codex_image_disabled", "Codex image generation is disabled")
	}
	route, err := t.service.ResolveModelRoute(ctx, cfg, execCtx.ActiveModel)
	if err != nil || !capabilitiesForProviderAccount(route).ImageGeneration {
		return toolFailure(execCtx.ToolCallID, CodexImagegenToolName, "codex_account_required", "active model is not an authenticated ChatGPT Codex route")
	}
	access, accountID, err := t.service.validOpenAIAccessToken(ctx, route.Credential)
	if err != nil {
		return toolErrorWithCallID(CodexImagegenToolName, execCtx.ToolCallID, err)
	}

	images, err := codexImageEditInputs(input, execCtx)
	if err != nil {
		return toolFailure(execCtx.ToolCallID, CodexImagegenToolName, "invalid_image_reference", err.Error())
	}
	body := map[string]any{
		"prompt": input.Prompt, "background": "auto", "model": codexImageModel, "quality": "auto", "size": "auto",
	}
	endpoint := chatGPTCodexImageGenerationURL
	operation := "generation"
	if len(images) > 0 {
		body["images"] = images
		endpoint = chatGPTCodexImageEditURL
		operation = "edit"
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return toolErrorWithCallID(CodexImagegenToolName, execCtx.ToolCallID, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return toolErrorWithCallID(CodexImagegenToolName, execCtx.ToolCallID, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("User-Agent", openAIUserAgent)
	if turnID := strings.TrimSpace(execCtx.TurnID); turnID != "" {
		req.Header.Set("x-codex-image-turn-id", turnID)
	}
	if accountID != "" {
		req.Header.Set("ChatGPT-Account-Id", accountID)
	}
	response, err := codexImageHTTPClient.Do(req)
	if err != nil {
		return toolErrorWithCallID(CodexImagegenToolName, execCtx.ToolCallID, err)
	}
	defer response.Body.Close()
	responseRaw, truncated, err := readBoundedBody(response.Body, codexImageMaxResponseBytes)
	if err != nil {
		return toolErrorWithCallID(CodexImagegenToolName, execCtx.ToolCallID, err)
	}
	if truncated {
		return toolFailure(execCtx.ToolCallID, CodexImagegenToolName, "image_response_too_large", "Codex image response exceeded the safe limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return codexImageProviderFailure(execCtx.ToolCallID, response, responseRaw)
	}
	var payload codexImageResponse
	if err := json.Unmarshal(responseRaw, &payload); err != nil || len(payload.Data) == 0 || strings.TrimSpace(payload.Data[0].Base64) == "" {
		return toolFailure(execCtx.ToolCallID, CodexImagegenToolName, "invalid_image_response", "Codex image response did not contain an image")
	}
	imageBytes, err := decodeBoundedGeneratedImage(payload.Data[0].Base64)
	if err != nil {
		return toolFailure(execCtx.ToolCallID, CodexImagegenToolName, "invalid_image_response", err.Error())
	}
	path, relPath, err := saveCodexGeneratedImage(execCtx.WorkspaceRoot, imageBytes)
	if err != nil {
		return toolFailure(execCtx.ToolCallID, CodexImagegenToolName, "image_save_failed", err.Error())
	}
	encoded := base64.StdEncoding.EncodeToString(imageBytes)
	content := fmt.Sprintf("Generated image saved to %s", path)
	return domain.ToolResult{
		CallID: execCtx.ToolCallID, Name: CodexImagegenToolName, OK: true, Content: content, ModelContent: content,
		ModelAttachments: []domain.MessageAttachment{{Name: filepath.Base(path), MIMEType: "image/png", Kind: "image", Data: encoded, Size: int64(len(imageBytes))}},
		Structured:       map[string]any{"provider": "codex", "operation": operation, "model": codexImageModel, "path": path, "relativePath": relPath, "size": payload.Size, "quality": payload.Quality, "background": payload.Background},
		Files:            []domain.ToolResultFile{{Path: relPath, FullPath: path, Type: "generated"}},
	}
}

func codexImageEditInputs(input codexImagegenInput, execCtx domain.ToolExecutionContext) ([]map[string]any, error) {
	if len(input.ReferencedImagePaths) == 0 && input.NumLastImagesToInclude == 0 {
		return nil, nil
	}
	urls := make([]string, 0, codexImageMaxEditImages)
	if len(input.ReferencedImagePaths) > 0 {
		for _, path := range input.ReferencedImagePaths {
			url, err := codexWorkspaceImageDataURL(execCtx.WorkspaceRoot, path)
			if err != nil {
				return nil, err
			}
			urls = append(urls, url)
		}
	} else {
		count := input.NumLastImagesToInclude
		if count < 1 || count > codexImageMaxEditImages {
			return nil, errors.New("num_last_images_to_include must be between one and five")
		}
		if len(execCtx.RecentImages) < count {
			return nil, fmt.Errorf("requested the last %d conversation images, but only %d were available", count, len(execCtx.RecentImages))
		}
		for _, attachment := range execCtx.RecentImages[len(execCtx.RecentImages)-count:] {
			url, err := codexAttachmentDataURL(attachment)
			if err != nil {
				return nil, err
			}
			urls = append(urls, url)
		}
	}
	items := make([]map[string]any, 0, len(urls))
	totalEncodedBytes := 0
	for _, url := range urls {
		totalEncodedBytes += len(url)
		if totalEncodedBytes > base64.StdEncoding.EncodedLen(codexImageMaxReferenceTotal)+1024*len(urls) {
			return nil, errors.New("referenced images exceed the combined safe limit")
		}
		items = append(items, map[string]any{"image_url": url})
	}
	return items, nil
}

func codexWorkspaceImageDataURL(workspaceRoot, relPath string) (string, error) {
	path, err := safeJoin(workspaceRoot, relPath)
	if err != nil {
		return "", fmt.Errorf("unable to read referenced image %q: %w", relPath, err)
	}
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("unable to read referenced image %q", relPath)
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, codexImageMaxReferenceBytes+1))
	if err != nil || len(raw) == 0 || len(raw) > codexImageMaxReferenceBytes {
		return "", fmt.Errorf("referenced image %q is empty or exceeds the safe limit", relPath)
	}
	mimeType := normalizeAttachmentMIME(http.DetectContentType(raw[:min(512, len(raw))]))
	if !isImageAttachmentMIME(mimeType) {
		return "", fmt.Errorf("referenced file %q is not a supported image", relPath)
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}

func codexAttachmentDataURL(attachment domain.MessageAttachment) (string, error) {
	encoded, embeddedMIME, err := attachmentBase64Payload(strings.TrimSpace(attachment.Data))
	if err != nil {
		return "", errors.New("conversation image data is invalid")
	}
	mimeType := normalizeAttachmentMIME(firstNonEmpty(embeddedMIME, attachment.MIMEType))
	if !isImageAttachmentMIME(mimeType) {
		return "", errors.New("conversation attachment is not a supported image")
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(raw) == 0 || len(raw) > codexImageMaxReferenceBytes {
		return "", errors.New("conversation image is empty or exceeds the safe limit")
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}

func decodeBoundedGeneratedImage(encoded string) ([]byte, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" || base64.StdEncoding.DecodedLen(len(encoded)) > codexImageMaxGeneratedBytes {
		return nil, errors.New("generated image exceeds the safe limit")
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(raw) == 0 || len(raw) > codexImageMaxGeneratedBytes {
		return nil, errors.New("generated image payload is invalid or exceeds the safe limit")
	}
	if mimeType := normalizeAttachmentMIME(http.DetectContentType(raw[:min(512, len(raw))])); mimeType != "image/png" {
		return nil, errors.New("generated image payload is not a PNG")
	}
	return raw, nil
}

func saveCodexGeneratedImage(workspaceRoot string, raw []byte) (string, string, error) {
	workspaceRoot = strings.TrimSpace(workspaceRoot)
	if workspaceRoot == "" {
		return "", "", errors.New("image generation requires an active workspace")
	}
	dir, err := safeJoin(workspaceRoot, "generated_images")
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", err
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", "", errors.New("generated_images must be a real workspace directory")
	}
	name := "image_" + uuid.NewString() + ".png"
	path := filepath.Join(dir, name)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", "", err
	}
	if _, err = file.Write(raw); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		_ = os.Remove(path)
		return "", "", err
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return "", "", closeErr
	}
	return path, filepath.ToSlash(filepath.Join("generated_images", name)), nil
}

func codexImageProviderFailure(callID string, response *http.Response, raw []byte) domain.ToolResult {
	message := bounded(providerHTTPError(response.StatusCode, response.Status, string(raw)).Error(), 1000)
	code := "provider_error"
	var payload struct {
		Error struct {
			Message string `json:"message"`
			Code    string `json:"code"`
			LimitID string `json:"limit_id"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &payload) == nil {
		if strings.TrimSpace(payload.Error.Message) != "" {
			message = bounded(payload.Error.Message, 1000)
		}
		if payload.Error.LimitID == "image_gen" || strings.Contains(strings.ToLower(payload.Error.Code), "usage") {
			code = "image_generation_usage_limit"
		}
	}
	return toolFailure(callID, CodexImagegenToolName, code, message)
}
