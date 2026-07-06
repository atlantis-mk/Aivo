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

	"aivo/core/domain"
)

const (
	browserNamespace            = "browser"
	browserNamespaceDescription = "Tools for operating Aivo's visible built-in browser. Use these only when the task requires a real interactive browser page, such as testing UI behavior, inspecting client-rendered pages, clicking controls, filling forms, reading console output, or capturing a screenshot. Prefer web_search and web_fetch for ordinary web research that does not require a live browser."
	defaultBrowserBridgeURL     = "http://127.0.0.1:43118"
)

type BrowserTool struct {
	name        string
	operation   string
	description string
	capability  string
	riskLevel   string
	schema      map[string]any
	client      *http.Client
	bridgeURL   string
}

type browserBridgeResponse struct {
	OK         bool           `json:"ok"`
	Content    string         `json:"content,omitempty"`
	Structured map[string]any `json:"structured,omitempty"`
	Error      string         `json:"error,omitempty"`
}

func NewBrowserTools() []domain.Tool {
	return []domain.Tool{
		newBrowserTool("browser_state", "state", "Return the current visible built-in browser state for a tab.", "browser.state", "low", browserTabSchema()),
		newBrowserTool("browser_navigate", "navigate", "Open the built-in browser tab and navigate it to an http(s) URL, then return the resulting page state.", "browser.navigate", "medium", browserNavigateSchema()),
		newBrowserTool("browser_snapshot", "snapshot", "Read the current browser page text and visible interactive elements from the live DOM.", "browser.snapshot", "low", browserSnapshotSchema()),
		newBrowserTool("browser_click", "click", "Click a visible element in the live browser by CSS selector, visible text, or snapshot index.", "browser.click", "medium", browserElementActionSchema(false)),
		newBrowserTool("browser_fill", "fill", "Fill a visible input, textarea, select-like, or contenteditable element in the live browser.", "browser.fill", "medium", browserElementActionSchema(true)),
		newBrowserTool("browser_press_key", "press_key", "Send a keyboard key event to the focused live browser page.", "browser.key", "medium", browserPressKeySchema()),
		newBrowserTool("browser_evaluate", "evaluate", "Execute JavaScript in the live browser page and return the JSON-serializable result. Use after safer browser tools are insufficient.", "browser.evaluate", "high", browserEvaluateSchema()),
		newBrowserTool("browser_screenshot", "screenshot", "Capture a PNG screenshot of the visible built-in browser tab and save it under the workspace screenshot directory by default.", "browser.screenshot", "low", browserScreenshotSchema()),
		newBrowserTool("browser_console_messages", "console", "Return recent console messages from the built-in browser tab.", "browser.console", "low", browserLogSchema()),
		newBrowserTool("browser_network_requests", "network", "Return recent network requests from the built-in browser tab.", "browser.network", "low", browserLogSchema()),
	}
}

func newBrowserTool(name string, operation string, description string, capability string, riskLevel string, schema map[string]any) *BrowserTool {
	return &BrowserTool{
		name: name, operation: operation, description: description, capability: capability, riskLevel: riskLevel,
		schema: schema, client: &http.Client{Timeout: 35 * time.Second}, bridgeURL: browserBridgeURL(),
	}
}

func browserBridgeURL() string {
	if value := strings.TrimSpace(os.Getenv("AIVO_BROWSER_BRIDGE_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	return defaultBrowserBridgeURL
}

func (t *BrowserTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:                 t.name,
		Description:          t.description,
		InputSchema:          t.schema,
		Namespace:            browserNamespace,
		NamespaceDescription: browserNamespaceDescription,
		Capability:           t.capability,
		RiskLevel:            t.riskLevel,
		Category:             "browser",
		Toolsets:             []string{"browser"},
		RequiresNetwork:      true,
	}
}

func (t *BrowserTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	if t == nil {
		return toolError("browser", errors.New("browser tool is not configured"))
	}
	if len(args) == 0 {
		args = json.RawMessage(`{}`)
	}
	var decoded map[string]any
	if err := json.Unmarshal(args, &decoded); err != nil {
		return toolError(t.name, err)
	}
	saveScreenshot := t.operation == "screenshot" && browserScreenshotShouldSave(decoded)
	bridgeArgs := decoded
	if t.operation == "screenshot" {
		bridgeArgs = cloneBrowserBridgeArgs(decoded)
		delete(bridgeArgs, "save")
	}
	response, err := t.callBridge(ctx, bridgeArgs)
	if err != nil {
		return toolError(t.name, err)
	}
	content := strings.TrimSpace(response.Content)
	structured := response.Structured
	if structured == nil {
		structured = map[string]any{}
	}
	files := []domain.ToolResultFile(nil)
	if t.operation == "screenshot" {
		if response.OK && saveScreenshot {
			file, err := saveBrowserScreenshot(ctx, execCtx.WorkspaceRoot, structured)
			if err != nil {
				return toolError(t.name, err)
			}
			files = []domain.ToolResultFile{file}
			structured["saved"] = true
			structured["savedPath"] = file.Path
			structured["savedFullPath"] = file.FullPath
			structured["files"] = files
		} else {
			structured["saved"] = false
		}
		content = browserScreenshotSummary(structured)
	}
	if content == "" && response.Error != "" {
		content = response.Error
	}
	return domain.ToolResult{
		Name:         t.name,
		OK:           response.OK,
		Content:      content,
		ModelContent: content,
		Structured:   structured,
		Files:        files,
		Error:        response.Error,
	}
}

func cloneBrowserBridgeArgs(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func browserScreenshotShouldSave(args map[string]any) bool {
	if value, ok := args["save"]; ok {
		if typed, ok := value.(bool); ok {
			return typed
		}
	}
	return true
}

func saveBrowserScreenshot(ctx context.Context, workspaceRoot string, structured map[string]any) (domain.ToolResultFile, error) {
	if err := ctx.Err(); err != nil {
		return domain.ToolResultFile{}, err
	}
	root := strings.TrimSpace(workspaceRoot)
	if root == "" {
		return domain.ToolResultFile{}, errors.New("workspace root is required to save browser screenshots")
	}
	dataURL, _ := structured["dataUrl"].(string)
	if strings.TrimSpace(dataURL) == "" {
		return domain.ToolResultFile{}, errors.New("browser screenshot response did not include image data")
	}
	imageBytes, err := decodeBrowserScreenshotDataURL(dataURL)
	if err != nil {
		return domain.ToolResultFile{}, err
	}
	if err := os.MkdirAll(filepath.Join(root, "screenshot"), 0o700); err != nil {
		return domain.ToolResultFile{}, err
	}
	relPath, absPath, err := nextBrowserScreenshotPath(root)
	if err != nil {
		return domain.ToolResultFile{}, err
	}
	if err := os.WriteFile(absPath, imageBytes, 0o600); err != nil {
		return domain.ToolResultFile{}, err
	}
	return domain.ToolResultFile{
		Path:     relPath,
		FullPath: filepath.ToSlash(absPath),
		Type:     "add",
	}, nil
}

func decodeBrowserScreenshotDataURL(dataURL string) ([]byte, error) {
	trimmed := strings.TrimSpace(dataURL)
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(trimmed, prefix) {
		return nil, errors.New("browser screenshot response is not a PNG data URL")
	}
	imageBytes, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(trimmed, prefix))
	if err != nil {
		return nil, fmt.Errorf("invalid browser screenshot image data: %w", err)
	}
	if len(imageBytes) == 0 {
		return nil, errors.New("browser screenshot image data is empty")
	}
	return imageBytes, nil
}

func nextBrowserScreenshotPath(workspaceRoot string) (string, string, error) {
	stamp := time.Now().Format("20060102-150405.000000000")
	stamp = strings.ReplaceAll(stamp, ".", "-")
	for i := 0; i < 100; i++ {
		name := "browser-" + stamp
		if i > 0 {
			name = fmt.Sprintf("%s-%02d", name, i)
		}
		relPath := filepath.ToSlash(filepath.Join("screenshot", name+".png"))
		absPath, err := safeTargetForWrite(workspaceRoot, relPath)
		if err != nil {
			return "", "", err
		}
		if _, err := os.Stat(absPath); errors.Is(err, os.ErrNotExist) {
			return relPath, absPath, nil
		} else if err != nil {
			return "", "", err
		}
	}
	return "", "", errors.New("could not allocate browser screenshot filename")
}

func (t *BrowserTool) callBridge(ctx context.Context, args map[string]any) (browserBridgeResponse, error) {
	payload, err := json.Marshal(map[string]any{
		"tool": t.operation,
		"args": args,
	})
	if err != nil {
		return browserBridgeResponse{}, err
	}
	bridgeURL := strings.TrimRight(firstNonEmpty(t.bridgeURL, browserBridgeURL()), "/") + "/browser-tool"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, bridgeURL, bytes.NewReader(payload))
	if err != nil {
		return browserBridgeResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := t.client
	if client == nil {
		client = &http.Client{Timeout: 35 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return browserBridgeResponse{}, fmt.Errorf("built-in browser bridge is unavailable at %s: %w", bridgeURL, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return browserBridgeResponse{}, err
	}
	var out browserBridgeResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return browserBridgeResponse{}, fmt.Errorf("invalid browser bridge response: %w", err)
	}
	if resp.StatusCode >= 500 {
		return out, errors.New(firstNonEmpty(out.Error, http.StatusText(resp.StatusCode)))
	}
	return out, nil
}

func browserScreenshotSummary(structured map[string]any) string {
	url, _ := structured["url"].(string)
	title, _ := structured["title"].(string)
	mimeType, _ := structured["mimeType"].(string)
	savedPath, _ := structured["savedPath"].(string)
	bytesValue := 0
	if raw, ok := structured["bytes"].(float64); ok {
		bytesValue = int(raw)
	}
	parts := []string{"Screenshot captured from the built-in browser."}
	if url != "" {
		parts = append(parts, "URL: "+url)
	}
	if title != "" {
		parts = append(parts, "Title: "+title)
	}
	if mimeType != "" {
		parts = append(parts, "Type: "+mimeType)
	}
	if savedPath != "" {
		parts = append(parts, "Saved: "+savedPath)
	}
	if bytesValue > 0 {
		parts = append(parts, fmt.Sprintf("Data URL bytes: %d", bytesValue))
	}
	return strings.Join(parts, "\n")
}

func browserTabSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tabId": map[string]any{"type": "string", "description": "Optional browser tab id. Defaults to builtin-browser."},
		},
		"additionalProperties": false,
	}
}

func browserScreenshotSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tabId": map[string]any{"type": "string", "description": "Optional browser tab id. Defaults to builtin-browser."},
			"save":  map[string]any{"type": "boolean", "description": "Whether to save the PNG under workspace screenshot/. Defaults to true; set false only when the user explicitly asks not to save."},
		},
		"additionalProperties": false,
	}
}

func browserNavigateSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tabId":     map[string]any{"type": "string", "description": "Optional browser tab id. Defaults to builtin-browser."},
			"url":       map[string]any{"type": "string", "description": "Absolute http(s) URL, or a host that can be normalized to https."},
			"timeoutMs": map[string]any{"type": "integer", "minimum": 250, "maximum": 30000, "description": "Maximum load wait time in milliseconds. Defaults to 10000."},
		},
		"required":             []string{"url"},
		"additionalProperties": false,
	}
}

func browserSnapshotSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tabId":        map[string]any{"type": "string", "description": "Optional browser tab id. Defaults to builtin-browser."},
			"maxTextChars": map[string]any{"type": "integer", "minimum": 1000, "maximum": 50000, "description": "Maximum page text characters. Defaults to 12000."},
			"maxElements":  map[string]any{"type": "integer", "minimum": 1, "maximum": 200, "description": "Maximum visible interactive elements. Defaults to 80."},
			"timeoutMs":    map[string]any{"type": "integer", "minimum": 250, "maximum": 15000, "description": "Maximum wait time for current load to settle. Defaults to 3000."},
		},
		"additionalProperties": false,
	}
}

func browserElementActionSchema(withValue bool) map[string]any {
	properties := map[string]any{
		"tabId":    map[string]any{"type": "string", "description": "Optional browser tab id. Defaults to builtin-browser."},
		"selector": map[string]any{"type": "string", "description": "CSS selector for the target element."},
		"text":     map[string]any{"type": "string", "description": "Visible label/text fallback for the target element."},
		"index":    map[string]any{"type": "integer", "minimum": 0, "description": "Index from browser_snapshot interactive elements."},
	}
	required := []string{}
	if withValue {
		properties["value"] = map[string]any{"type": "string", "description": "Text to enter into the target element."}
		required = append(required, "value")
	}
	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             required,
		"additionalProperties": false,
	}
}

func browserPressKeySchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tabId": map[string]any{"type": "string", "description": "Optional browser tab id. Defaults to builtin-browser."},
			"key":   map[string]any{"type": "string", "description": "Electron keyCode, such as Enter, Escape, Tab, ArrowDown, or a printable character."},
		},
		"required":             []string{"key"},
		"additionalProperties": false,
	}
}

func browserEvaluateSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tabId":       map[string]any{"type": "string", "description": "Optional browser tab id. Defaults to builtin-browser."},
			"script":      map[string]any{"type": "string", "description": "JavaScript expression or async function body to execute in the page."},
			"userGesture": map[string]any{"type": "boolean", "description": "Whether to execute with a user gesture flag."},
		},
		"required":             []string{"script"},
		"additionalProperties": false,
	}
}

func browserLogSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tabId": map[string]any{"type": "string", "description": "Optional browser tab id. Defaults to builtin-browser."},
			"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 200, "description": "Maximum recent entries. Defaults to 50."},
		},
		"additionalProperties": false,
	}
}
