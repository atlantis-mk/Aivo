package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"aivo/core/domain"
)

type reservedPrimitiveTestTool struct{ name string }

type unavailablePrimitiveEnvironment struct{ calls []string }

func (e *unavailablePrimitiveEnvironment) Identity() string { return "remote-test-v1" }
func (e *unavailablePrimitiveEnvironment) ExecutePrimitive(_ context.Context, operation string, _ json.RawMessage, _ domain.ToolExecutionContext) domain.ToolResult {
	e.calls = append(e.calls, operation)
	return primitiveError(operation, "environment_unavailable", fmt.Errorf("remote environment is unavailable"))
}

func (t *reservedPrimitiveTestTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{Name: t.name, InputSchema: map[string]any{"type": "object"}}
}
func (t *reservedPrimitiveTestTool) Execute(context.Context, json.RawMessage, domain.ToolExecutionContext) domain.ToolResult {
	return domain.ToolResult{Name: t.name, OK: true}
}

func TestCodingRegistryKeepsDefaultPrimitivesInStableOrder(t *testing.T) {
	registry, err := NewCodingToolRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	specs := registry.Specs()
	names := make([]string, 0, len(specs))
	for _, spec := range specs {
		names = append(names, spec.Name)
	}
	want := []string{"read", ExecCommandToolName, WriteStdinToolName, "edit", "write", "grep", "find", "ls"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("tools = %v, want %v", names, want)
	}
	assembly := AssembleToolSpecs(registry, registry.Specs())
	defaultNames := make([]string, 0, len(assembly.Specs))
	for _, spec := range assembly.Specs {
		defaultNames = append(defaultNames, spec.Name)
	}
	if wantDefaults := []string{"read", ExecCommandToolName, WriteStdinToolName, "edit", "write"}; !reflect.DeepEqual(defaultNames, wantDefaults) {
		t.Fatalf("default tools = %v, want %v", defaultNames, wantDefaults)
	}
	for _, removed := range []string{"read_file", "edit_file", "write_file", "search_files", "glob", "list_files", "apply_patch", ResourceResolveName} {
		if _, ok := registry.Get(removed); ok {
			t.Fatalf("legacy tool %s is executable", removed)
		}
	}
}

func TestPrimitivePathContractGuidesWorkspaceRelativePaths(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("alpha\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tools := []domain.Tool{NewReadTool(root), NewEditTool(root), NewWriteTool(root)}
	for _, tool := range tools {
		properties, _ := tool.Spec().InputSchema["properties"].(map[string]any)
		pathSchema, _ := properties["path"].(map[string]any)
		description, _ := pathSchema["description"].(string)
		if !strings.Contains(description, "relative to the active workspace root") || !strings.Contains(description, `"notes.txt"`) {
			t.Fatalf("%s path description = %q, want explicit relative-path rule and example", tool.Spec().Name, description)
		}
	}

	absolute := filepath.Join(root, "notes.txt")
	calls := []struct {
		name string
		tool domain.Tool
		args json.RawMessage
	}{
		{name: "read", tool: NewReadTool(root), args: mustJSON(map[string]any{"path": absolute})},
		{name: "edit", tool: NewEditTool(root), args: mustJSON(map[string]any{"path": absolute, "edits": []map[string]string{{"oldText": "alpha", "newText": "beta"}}})},
		{name: "write", tool: NewWriteTool(root), args: mustJSON(map[string]any{"path": absolute, "content": "beta\n"})},
	}
	for _, call := range calls {
		result := call.tool.Execute(context.Background(), call.args, domain.ToolExecutionContext{})
		if result.OK || result.ToolError == nil || result.ToolError.Code != "invalid_path" || !strings.Contains(result.ToolError.Message, "remove the workspace-root prefix") {
			t.Fatalf("%s absolute-path result = %#v, want actionable invalid_path", call.name, result)
		}
	}
	if reason := deniedPathReason([]string{absolute}); reason != workspaceRelativePathErrorMessage {
		t.Fatalf("absolute-path permission denial = %q, want %q", reason, workspaceRelativePathErrorMessage)
	}

	traversal := NewEditTool(root).Execute(context.Background(), json.RawMessage(`{"path":"../notes.txt","edits":[{"oldText":"alpha","newText":"beta"}]}`), domain.ToolExecutionContext{})
	if traversal.OK || traversal.ToolError == nil || !strings.Contains(traversal.ToolError.Message, "escapes workspace root") {
		t.Fatalf("traversal result = %#v, want workspace escape error", traversal)
	}

	request := SandboxRequest{SessionID: "relative-path-contract", ToolCallID: "retained-read"}
	ref, err := retainSandboxOutput(request, "stdout", "retained output\n")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupRetainedOutputSession(request.SessionID)
	retained := NewReadTool(root).Execute(context.Background(), mustJSON(map[string]any{"path": ref}), domain.ToolExecutionContext{})
	if !retained.OK || !strings.Contains(retained.ModelContent, "retained output") {
		t.Fatalf("retained-output read = %#v, want absolute Host ref to remain valid", retained)
	}
}

func TestReservedPrimitiveCannotBeOverridden(t *testing.T) {
	registry, err := NewCodingToolRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterScoped(&reservedPrimitiveTestTool{name: "read"}, domain.ToolSourceExtension, "bad", "1"); err == nil {
		t.Fatal("extension overrode reserved read tool")
	}
}

func TestAlternateEnvironmentAtomicallyOwnsAllFourPrimitivesWithoutLocalFallback(t *testing.T) {
	root := t.TempDir()
	environment := &unavailablePrimitiveEnvironment{}
	registry, err := NewCodingToolRegistryWithExecutionEnvironment(root, nil, environment)
	if err != nil {
		t.Fatal(err)
	}
	localRegistry, err := NewCodingToolRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	if AssembleToolSpecs(registry, registry.Specs()).Snapshot.Revision == AssembleToolSpecs(localRegistry, localRegistry.Specs()).Snapshot.Revision {
		t.Fatal("environment switch did not create a new tool snapshot identity")
	}
	arguments := map[string]json.RawMessage{
		"read":              json.RawMessage(`{"path":"remote.txt"}`),
		ExecCommandToolName: json.RawMessage(`{"cmd":"printf remote"}`),
		WriteStdinToolName:  json.RawMessage(`{"process_ref":"agent-pty:test","chars":""}`),
		"edit":              json.RawMessage(`{"path":"remote.txt","edits":[{"oldText":"a","newText":"b"}]}`),
		"write":             json.RawMessage(`{"path":"remote.txt","content":"remote"}`),
		"grep":              json.RawMessage(`{"query":"remote"}`),
		"find":              json.RawMessage(`{"pattern":"**/*.txt"}`),
		"ls":                json.RawMessage(`{}`),
	}
	for _, name := range []string{"read", ExecCommandToolName, WriteStdinToolName, "edit", "write", "grep", "find", "ls"} {
		tool, ok := registry.Get(name)
		if !ok {
			t.Fatalf("missing %s", name)
		}
		result := tool.Execute(context.Background(), arguments[name], domain.ToolExecutionContext{})
		if result.OK || result.ToolError == nil || result.ToolError.Code != "environment_unavailable" {
			t.Fatalf("%s result = %#v, want explicit environment loss", name, result)
		}
	}
	if !reflect.DeepEqual(environment.calls, []string{"read", ExecCommandToolName, WriteStdinToolName, "edit", "write", "grep", "find", "ls"}) {
		t.Fatalf("environment calls = %v", environment.calls)
	}
	if _, err := os.Stat(filepath.Join(root, "remote.txt")); !os.IsNotExist(err) {
		t.Fatalf("alternate environment silently fell back to local filesystem: %v", err)
	}
}

func TestReadPrimitiveTextPaginationAndBinaryRefusal(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("one\ntwo\nthree\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tool := NewReadTool(root)
	result := tool.Execute(context.Background(), json.RawMessage(`{"path":"notes.txt","offset":2,"limit":1}`), domain.ToolExecutionContext{})
	if !result.OK || !strings.Contains(result.ModelContent, "2|two") || result.Structured["nextOffset"] != 3 {
		t.Fatalf("result = %#v", result)
	}
	if err := os.WriteFile(filepath.Join(root, "binary.dat"), []byte{0, 1, 2}, 0o600); err != nil {
		t.Fatal(err)
	}
	result = tool.Execute(context.Background(), json.RawMessage(`{"path":"binary.dat"}`), domain.ToolExecutionContext{})
	if result.OK || result.ToolError == nil || result.ToolError.Code != "unsupported_binary" {
		t.Fatalf("binary result = %#v", result)
	}
}

func TestReadPrimitiveReturnsBoundedImageAttachment(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "pixel.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	img.Set(1, 1, color.RGBA{R: 255, A: 255})
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	result := NewReadTool(root).Execute(context.Background(), json.RawMessage(`{"path":"pixel.png"}`), domain.ToolExecutionContext{})
	if !result.OK || len(result.ModelAttachments) != 1 || result.ModelAttachments[0].MIMEType != "image/png" {
		t.Fatalf("result = %#v", result)
	}
	if result.Structured["width"] != 4 || result.Structured["height"] != 3 {
		t.Fatalf("details = %#v", result.Structured)
	}
}

func TestEditPrimitiveAppliesExactBatchAtomically(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	if err := os.WriteFile(path, []byte("alpha beta gamma"), 0o600); err != nil {
		t.Fatal(err)
	}
	tool := NewEditTool(root)
	result := tool.Execute(context.Background(), json.RawMessage(`{"path":"file.txt","edits":[{"oldText":"alpha","newText":"A"},{"oldText":"gamma","newText":"G"}]}`), domain.ToolExecutionContext{})
	if !result.OK {
		t.Fatalf("result = %#v", result)
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "A beta G" {
		t.Fatalf("content = %q", raw)
	}

	result = tool.Execute(context.Background(), json.RawMessage(`{"path":"file.txt","edits":[{"oldText":"A beta","newText":"x"},{"oldText":"beta G","newText":"y"}]}`), domain.ToolExecutionContext{})
	if result.OK || result.ToolError.Code != "overlapping_edits" {
		t.Fatalf("overlap result = %#v", result)
	}
	raw, _ = os.ReadFile(path)
	if string(raw) != "A beta G" {
		t.Fatalf("overlap changed file to %q", raw)
	}
}

func TestEditPrimitiveRejectsAmbiguousMatchWithoutChangingFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	if err := os.WriteFile(path, []byte("same same"), 0o600); err != nil {
		t.Fatal(err)
	}
	result := NewEditTool(root).Execute(context.Background(), json.RawMessage(`{"path":"file.txt","edits":[{"oldText":"same","newText":"x"}]}`), domain.ToolExecutionContext{})
	if result.OK || result.ToolError.Code != "multiple_matches" {
		t.Fatalf("result = %#v", result)
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "same same" {
		t.Fatalf("file changed to %q", raw)
	}
}

func TestWritePrimitiveCreatesParentsAndSerializesMutations(t *testing.T) {
	root := t.TempDir()
	tool := NewWriteTool(root)
	path := filepath.Join("nested", "file.txt")
	var wait sync.WaitGroup
	for _, content := range []string{"first", "second"} {
		content := content
		wait.Add(1)
		go func() {
			defer wait.Done()
			result := tool.Execute(context.Background(), mustJSON(map[string]any{"path": path, "content": content}), domain.ToolExecutionContext{})
			if !result.OK {
				t.Errorf("write result = %#v", result)
			}
		}()
	}
	wait.Wait()
	raw, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "first" && string(raw) != "second" {
		t.Fatalf("partial write = %q", raw)
	}
	matches, err := filepath.Glob(filepath.Join(root, "nested", ".aivo-write-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary files = %v, err = %v", matches, err)
	}
}

func TestEditAndWriteRejectFilesChangedAfterApproval(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	if err := os.WriteFile(path, []byte("approved"), 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("approved"))
	storePreparedExpectedHash("edit-call", "file.txt", fmt.Sprintf("%x", digest))
	storePreparedExpectedHash("write-call", "file.txt", fmt.Sprintf("%x", digest))
	if err := os.WriteFile(path, []byte("external"), 0o600); err != nil {
		t.Fatal(err)
	}

	edit := NewEditTool(root).Execute(context.Background(), json.RawMessage(`{"path":"file.txt","edits":[{"oldText":"external","newText":"edited"}]}`), domain.ToolExecutionContext{ToolCallID: "edit-call"})
	if edit.OK || edit.ToolError == nil || edit.ToolError.Code != "file_changed" {
		t.Fatalf("edit result = %#v, want file_changed", edit)
	}
	write := NewWriteTool(root).Execute(context.Background(), json.RawMessage(`{"path":"file.txt","content":"written"}`), domain.ToolExecutionContext{ToolCallID: "write-call"})
	if write.OK || write.ToolError == nil || write.ToolError.Code != "file_changed" {
		t.Fatalf("write result = %#v, want file_changed", write)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "external" {
		t.Fatalf("content = %q, err = %v; external change was not preserved", raw, err)
	}
}

func TestToolSnapshotFreezesRegistrationAndSchemaIdentities(t *testing.T) {
	registry, err := NewCodingToolRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	assembly := AssembleToolSpecs(registry, registry.Specs())
	if len(assembly.Snapshot.Tools) != 5 || assembly.Snapshot.Revision == "" {
		t.Fatalf("snapshot = %#v", assembly.Snapshot)
	}
	for _, entry := range assembly.Snapshot.Tools {
		if entry.RegistrationID == "" || entry.SchemaHash == "" || entry.ActivationSource != "core" {
			t.Fatalf("entry = %#v", entry)
		}
	}
	second := AssembleToolSpecs(registry, registry.Specs())
	if second.Snapshot.Revision != assembly.Snapshot.Revision {
		t.Fatalf("revisions differ: %s != %s", second.Snapshot.Revision, assembly.Snapshot.Revision)
	}
}

func TestToolRuntimeLogsArgumentSizeWithoutRawArguments(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(testTool{name: "safe_log"}); err != nil {
		t.Fatal(err)
	}
	var captured bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&captured)
	t.Cleanup(func() { log.SetOutput(previous) })
	secret := "never-log-this-secret"
	result := NewToolRuntime(registry, t.TempDir()).Execute(context.Background(), domain.ChatToolCall{ID: "safe-call", Name: "safe_log", Arguments: mustJSON(map[string]any{"value": secret})})
	if !result.OK {
		t.Fatalf("result = %#v", result)
	}
	if strings.Contains(captured.String(), secret) || !strings.Contains(captured.String(), "argument_bytes=") {
		t.Fatalf("unsafe tool log = %q", captured.String())
	}
}

func TestToolRuntimeDoesNotLogToolErrorMessages(t *testing.T) {
	registry := NewRegistry()
	secret := "secret-returned-by-extension"
	if err := registry.Register(testTool{name: "unsafe_error", content: secret}); err != nil {
		t.Fatal(err)
	}
	var captured bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&captured)
	t.Cleanup(func() { log.SetOutput(previous) })
	result := NewToolRuntime(registry, t.TempDir()).finish(domain.ChatToolCall{ID: "error-call"}, time.Now(), toolFailure("error-call", "unsafe_error", "external_failure", secret), false)
	if result.OK || !strings.Contains(result.Error, secret) {
		t.Fatalf("result = %#v", result)
	}
	if strings.Contains(captured.String(), secret) || !strings.Contains(captured.String(), "error_code=external_failure") {
		t.Fatalf("unsafe tool error log = %q", captured.String())
	}
}

func mustJSON(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}
