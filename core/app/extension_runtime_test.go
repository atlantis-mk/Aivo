package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"aivo/core/domain"
)

func TestLoadExtensionManifestV2IsDeclarativeAndHashesSchemas(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "schemas", "echo.json"), `{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`, 0o600)
	writeTestExtensionManifest(t, root, map[string]any{
		"schemaVersion": 2, "id": "com.example_echo", "name": "Echo", "version": "1.0.0", "apiVersion": "2",
		"runtime":     map[string]any{"type": "process", "command": "bin/echo-extension", "transport": "stdio"},
		"contributes": map[string]any{"tools": []any{map[string]any{"name": "example_echo", "description": "Echo text", "schema": "schemas/echo.json", "activation": "auto"}}},
	})
	writeTestFile(t, filepath.Join(root, "bin", "echo-extension"), "not executed during discovery", 0o700)
	loaded, err := LoadExtensionManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Integrity == "" || loaded.ToolSchemas["example_echo"]["type"] != "object" {
		t.Fatalf("loaded = %#v", loaded)
	}
	firstIntegrity := loaded.Integrity
	writeTestFile(t, filepath.Join(root, "schemas", "echo.json"), `{"type":"object","properties":{"value":{"type":"number"}}}`, 0o600)
	changed, err := LoadExtensionManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Integrity == firstIntegrity {
		t.Fatal("schema change did not change package integrity")
	}
}

func TestLoadExtensionManifestSupportsExplicitToolGroups(t *testing.T) {
	root := t.TempDir()
	writeTestExtensionManifest(t, root, map[string]any{
		"schemaVersion": 2, "id": "com.example.calendar", "name": "Calendar", "version": "1.0.0", "apiVersion": "2",
		"runtime": map[string]any{"type": "builtin"},
		"contributes": map[string]any{
			"tools": []any{
				map[string]any{"name": "calendar_list", "description": "List events", "schema": map[string]any{"type": "object"}},
				map[string]any{"name": "calendar_update", "description": "Update events", "schema": map[string]any{"type": "object"}},
				map[string]any{"name": "calendar_health", "description": "Check health", "schema": map[string]any{"type": "object"}},
			},
			"toolGroups": []any{map[string]any{
				"id": "events", "name": "Calendar Events", "description": "Read and update calendar events",
				"tools": []any{"calendar_list", "calendar_update"},
			}},
		},
	})
	loaded, err := LoadExtensionManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	groups := extensionToolSelectionGroups(loaded.Manifest)
	wantID := generatedToolName("extension", "com.example.calendar", "events")
	if len(groups) != 2 || groups["calendar_list"].ID != wantID || groups["calendar_list"].Name != "Calendar Events" || groups["calendar_update"].ID != wantID {
		t.Fatalf("selection groups = %#v", groups)
	}
	if _, grouped := groups["calendar_health"]; grouped {
		t.Fatalf("ungrouped tool was assigned a group: %#v", groups)
	}
}

func TestLoadExtensionManifestRejectsInvalidToolGroups(t *testing.T) {
	for name, toolGroups := range map[string][]any{
		"undeclared member": {map[string]any{"id": "events", "name": "Events", "tools": []any{"calendar_missing"}}},
		"duplicate membership": {
			map[string]any{"id": "events", "name": "Events", "tools": []any{"calendar_list"}},
			map[string]any{"id": "updates", "name": "Updates", "tools": []any{"calendar_list"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeTestExtensionManifest(t, root, map[string]any{
				"schemaVersion": 2, "id": "com.example.calendar", "name": "Calendar", "version": "1.0.0", "apiVersion": "2",
				"runtime": map[string]any{"type": "builtin"},
				"contributes": map[string]any{
					"tools":      []any{map[string]any{"name": "calendar_list", "description": "List events", "schema": map[string]any{"type": "object"}}},
					"toolGroups": toolGroups,
				},
			})
			if _, err := LoadExtensionManifest(root); err == nil {
				t.Fatalf("accepted invalid tool groups %#v", toolGroups)
			}
		})
	}
}

func TestLoadExtensionManifestRejectsReservedNamesAndEscapingAssets(t *testing.T) {
	root := t.TempDir()
	writeTestExtensionManifest(t, root, map[string]any{
		"schemaVersion": 2, "id": "com.example.bad", "name": "Bad", "version": "1", "apiVersion": "2",
		"runtime":     map[string]any{"type": "builtin"},
		"contributes": map[string]any{"tools": []any{map[string]any{"name": "read", "schema": map[string]any{"type": "object"}}}},
	})
	if _, err := LoadExtensionManifest(root); err == nil {
		t.Fatal("reserved core tool was accepted")
	}
	writeTestExtensionManifest(t, root, map[string]any{
		"schemaVersion": 2, "id": "com.example.bad", "name": "Bad", "version": "1", "apiVersion": "2",
		"runtime":     map[string]any{"type": "builtin"},
		"contributes": map[string]any{"contexts": []any{map[string]any{"id": "bad", "kind": "skill", "path": "../secret"}}},
	})
	if _, err := LoadExtensionManifest(root); err == nil {
		t.Fatal("escaping context asset was accepted")
	}
}

func TestLoadExtensionManifestValidatesDynamicServiceTransport(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "bin", "service"), "service", 0o700)
	base := map[string]any{
		"schemaVersion": 2, "id": "com.example.dynamic", "name": "Dynamic", "version": "1", "apiVersion": "2",
		"runtime": map[string]any{"type": "service", "command": "bin/service", "transport": "dynamic-http"},
	}
	writeTestExtensionManifest(t, root, base)
	if _, err := LoadExtensionManifest(root); err != nil {
		t.Fatalf("dynamic service manifest failed: %v", err)
	}

	base["runtime"] = map[string]any{"type": "service", "command": "bin/service", "transport": "dynamic-http", "url": "http://127.0.0.1:45000"}
	writeTestExtensionManifest(t, root, base)
	if _, err := LoadExtensionManifest(root); err == nil || !strings.Contains(err.Error(), "must omit runtime.url") {
		t.Fatalf("dynamic service URL error = %v", err)
	}

	base["runtime"] = map[string]any{"type": "service", "command": "bin/service", "transport": "unknown"}
	writeTestExtensionManifest(t, root, base)
	if _, err := LoadExtensionManifest(root); err == nil || !strings.Contains(err.Error(), "http or dynamic-http") {
		t.Fatalf("unknown service transport error = %v", err)
	}
}

func TestLoadExtensionManifestV2ValidatesRuntimeMessagingPermission(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "bin", "service"), "service", 0o700)
	manifest := map[string]any{
		"schemaVersion": 2, "id": "com.example.messaging", "name": "Messaging", "version": "1", "apiVersion": "2",
		"runtime":     map[string]any{"type": "service", "command": "bin/service", "transport": "dynamic-http"},
		"permissions": []any{"runtime.messaging"},
	}
	writeTestExtensionManifest(t, root, manifest)
	loaded, err := LoadExtensionManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded.Manifest.Permissions, []string{"runtime.messaging"}) {
		t.Fatalf("permissions = %#v", loaded.Manifest.Permissions)
	}

	for name, testCase := range map[string]struct {
		schema int
		api    string
		values []any
		want   string
	}{
		"v1 unsupported": {1, "1", nil, "supported 2/2 pair"},
		"mixed versions": {2, "1", nil, "supported 2/2 pair"},
		"unknown":        {2, "2", []any{"tabs"}, "invalid or duplicate extension permission"},
		"duplicate":      {2, "2", []any{"runtime.messaging", "runtime.messaging"}, "invalid or duplicate extension permission"},
	} {
		t.Run(name, func(t *testing.T) {
			manifest["schemaVersion"] = testCase.schema
			manifest["apiVersion"] = testCase.api
			manifest["permissions"] = testCase.values
			writeTestExtensionManifest(t, root, manifest)
			if _, err := LoadExtensionManifest(root); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("invalid manifest error = %v; want %q", err, testCase.want)
			}
		})
	}
}

func TestExtensionToolViewRefPrefersToolDetailAndUsesPageFallback(t *testing.T) {
	manifest := domain.ExtensionManifest{
		ID: "com.example.views",
		Contributes: domain.ExtensionContributions{Views: []domain.ExtensionViewContribution{
			{ID: "page", Title: "Page", Surfaces: []string{"page"}, Tools: []string{"example_tool"}},
			{ID: "detail", Title: "Detail", Surfaces: []string{"page", "tool-detail"}, Tools: []string{"example_tool"}},
			{ID: "other", Title: "Other", Surfaces: []string{"tool-detail"}, Tools: []string{"other_tool"}},
		}},
	}

	ref := extensionToolViewRef(manifest, "example_tool")
	if ref == nil || ref.ExtensionID != manifest.ID || ref.ViewID != "detail" || ref.Surface != "tool-detail" || ref.Title != "Detail" {
		t.Fatalf("tool detail ref = %#v", ref)
	}

	manifest.Contributes.Views = manifest.Contributes.Views[:1]
	ref = extensionToolViewRef(manifest, "example_tool")
	if ref == nil || ref.ViewID != "page" || ref.Surface != "page" {
		t.Fatalf("page fallback ref = %#v", ref)
	}
	if ref := extensionToolViewRef(manifest, "missing_tool"); ref != nil {
		t.Fatalf("missing tool ref = %#v; want nil", ref)
	}
}

type builtinExtensionTestClient struct {
	events     *[]string
	executeErr error
}

func (c *builtinExtensionTestClient) Initialize(context.Context, domain.ExtensionManifest) error {
	*c.events = append(*c.events, "initialize")
	return nil
}
func (c *builtinExtensionTestClient) Execute(_ context.Context, name string, _ json.RawMessage, _ domain.ToolExecutionContext) (domain.ToolResult, error) {
	*c.events = append(*c.events, "execute:"+name)
	if c.executeErr != nil {
		return domain.ToolResult{Name: name}, c.executeErr
	}
	return domain.ToolResult{Name: name, OK: true, Content: "ok"}, nil
}
func (c *builtinExtensionTestClient) UIEvent(context.Context, string, string, any) (any, error) {
	return map[string]any{"ok": true}, nil
}
func (c *builtinExtensionTestClient) Shutdown(context.Context) error {
	*c.events = append(*c.events, "shutdown")
	return nil
}

type versionedBuiltinTestClient struct{ version string }

type policyBuiltinTestClient struct{}

func (*policyBuiltinTestClient) Initialize(context.Context, domain.ExtensionManifest) error {
	return nil
}
func (*policyBuiltinTestClient) Execute(_ context.Context, name string, _ json.RawMessage, _ domain.ToolExecutionContext) (domain.ToolResult, error) {
	structured := map[string]any{}
	if name == "policy.example.guard.pre_tool_call" {
		structured = map[string]any{"block": true, "message": "blocked by test policy"}
	}
	return domain.ToolResult{Name: name, OK: true, Structured: structured}, nil
}
func (*policyBuiltinTestClient) UIEvent(context.Context, string, string, any) (any, error) {
	return nil, nil
}
func (*policyBuiltinTestClient) Shutdown(context.Context) error { return nil }

func (c *versionedBuiltinTestClient) Initialize(_ context.Context, manifest domain.ExtensionManifest) error {
	if manifest.Version == "broken" {
		return fmt.Errorf("broken update")
	}
	c.version = manifest.Version
	return nil
}
func (c *versionedBuiltinTestClient) Execute(_ context.Context, name string, _ json.RawMessage, _ domain.ToolExecutionContext) (domain.ToolResult, error) {
	return domain.ToolResult{Name: name, OK: true, Content: c.version}, nil
}
func (c *versionedBuiltinTestClient) UIEvent(context.Context, string, string, any) (any, error) {
	return nil, nil
}
func (c *versionedBuiltinTestClient) Shutdown(context.Context) error { return nil }

func TestExtensionSupervisorSeparatesDiscoveryTrustActivationAndExecution(t *testing.T) {
	root := t.TempDir()
	writeTestExtensionManifest(t, root, map[string]any{
		"schemaVersion": 2, "id": "com.example.builtin", "name": "Builtin", "version": "1.0.0", "apiVersion": "2",
		"runtime": map[string]any{"type": "builtin"},
		"contributes": map[string]any{
			"tools":       []any{map[string]any{"name": "example_echo", "description": "Echo", "schema": map[string]any{"type": "object"}, "activation": "manual"}},
			"views":       []any{map[string]any{"id": "echo-detail", "title": "Echo detail", "type": "web", "route": "/ui", "surfaces": []string{"tool-detail"}, "tools": []string{"example_echo"}}},
			"environment": map[string]any{"id": "example.environment"},
		},
	})
	events := []string{}
	supervisor := NewExtensionSupervisor()
	var client *builtinExtensionTestClient
	supervisor.RegisterBuiltin("com.example.builtin", func() extensionRuntimeClient {
		client = &builtinExtensionTestClient{events: &events}
		return client
	})
	status, err := supervisor.Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != domain.ExtensionStateValidated || len(events) != 0 {
		t.Fatalf("discovery status = %#v, events = %v", status, events)
	}
	status, err = supervisor.Enable(context.Background(), status.ID)
	if err != nil || status.State != domain.ExtensionStateReady || !reflect.DeepEqual(events, []string{"initialize"}) {
		t.Fatalf("enable status = %#v, events = %v, err = %v", status, events, err)
	}
	environment, err := supervisor.ExecutionEnvironment(status.ID)
	if err != nil {
		t.Fatal(err)
	}
	environmentRegistry, err := NewCodingToolRegistryWithExecutionEnvironment(t.TempDir(), nil, environment)
	if err != nil {
		t.Fatal(err)
	}
	environmentResult := NewToolRuntime(environmentRegistry, t.TempDir()).Execute(context.Background(), domain.ChatToolCall{Name: "read", Arguments: json.RawMessage(`{"path":"remote.txt"}`)})
	if !environmentResult.OK || !containsString(events, "execute:environment.read") {
		t.Fatalf("environment result = %#v, events = %v", environmentResult, events)
	}
	registry, err := NewCodingToolRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := supervisor.RegisterReadyTools(status.ID, registry); err != nil {
		t.Fatal(err)
	}
	assembly := AssembleToolSpecsWithActivated(registry, registry.Specs(), map[string]bool{"example_echo": true})
	if len(assembly.Specs) != 6 || assembly.Specs[5].Name != "example_echo" {
		t.Fatalf("activated specs = %#v", assembly.Specs)
	}
	result := supervisor.Execute(context.Background(), status.ID, "example_echo", json.RawMessage(`{}`), domain.ToolExecutionContext{ToolSnapshot: &assembly.Snapshot})
	if !result.OK || result.Details == nil || result.Details.View == nil || result.Details.View.ViewID != "echo-detail" || !reflect.DeepEqual(events, []string{"initialize", "execute:environment.read", "execute:example_echo"}) {
		t.Fatalf("result = %#v, events = %v", result, events)
	}
	modelJSON, err := json.Marshal(result)
	if err != nil || strings.Contains(string(modelJSON), "echo-detail") {
		t.Fatalf("model result exposed Host-only view metadata: %s, err = %v", modelJSON, err)
	}
	client.executeErr = errors.New("extension failure")
	failed := supervisor.Execute(context.Background(), status.ID, "example_echo", json.RawMessage(`{}`), domain.ToolExecutionContext{ToolSnapshot: &assembly.Snapshot})
	if failed.OK || failed.Details == nil || failed.Details.View == nil || failed.Details.View.ViewID != "echo-detail" {
		t.Fatalf("failed result lost Host-only view metadata: %#v", failed)
	}
	status, err = supervisor.Stop(context.Background(), status.ID)
	if err != nil || status.State != domain.ExtensionStateStopped || events[len(events)-1] != "shutdown" {
		t.Fatalf("stop status = %#v, events = %v, err = %v", status, events, err)
	}
}

func TestExtensionPolicyInterceptsBeforeToolExecution(t *testing.T) {
	root := t.TempDir()
	writeTestExtensionManifest(t, root, map[string]any{
		"schemaVersion": 2, "id": "com.example.policy", "name": "Policy", "version": "1", "apiVersion": "2",
		"runtime":     map[string]any{"type": "builtin"},
		"contributes": map[string]any{"policies": []string{"example.guard"}},
	})
	supervisor := NewExtensionSupervisor()
	supervisor.RegisterBuiltin("com.example.policy", func() extensionRuntimeClient { return &policyBuiltinTestClient{} })
	status, err := supervisor.Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := supervisor.Enable(context.Background(), status.ID); err != nil {
		t.Fatal(err)
	}
	registry := NewRegistry()
	if err := registry.Register(testTool{name: "guarded"}); err != nil {
		t.Fatal(err)
	}
	runtime := NewToolRuntime(registry, t.TempDir())
	runtime.ExtensionHooks = supervisor
	result := runtime.Execute(context.Background(), domain.ChatToolCall{Name: "guarded", Arguments: json.RawMessage(`{}`)})
	if result.OK || result.ToolError == nil || result.ToolError.Code != "policy_denied" || result.Error != "blocked by test policy" {
		t.Fatalf("policy result = %#v", result)
	}
}

func TestExtensionUpdateKeepsFrozenGenerationAndFailedUpdateKeepsCurrent(t *testing.T) {
	root := t.TempDir()
	writeVersion := func(version string) {
		writeTestExtensionManifest(t, root, map[string]any{
			"schemaVersion": 2, "id": "com.example_versioned", "name": "Versioned", "version": version, "apiVersion": "2",
			"runtime":     map[string]any{"type": "builtin"},
			"contributes": map[string]any{"tools": []any{map[string]any{"name": "example_versioned", "description": version, "schema": map[string]any{"type": "object"}}}},
		})
	}
	writeVersion("1.0.0")
	supervisor := NewExtensionSupervisor()
	supervisor.RegisterBuiltin("com.example_versioned", func() extensionRuntimeClient { return &versionedBuiltinTestClient{} })
	status, err := supervisor.Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := supervisor.Enable(context.Background(), status.ID); err != nil {
		t.Fatal(err)
	}
	oldRegistry := NewRegistry()
	if err := supervisor.RegisterReadyTools(status.ID, oldRegistry); err != nil {
		t.Fatal(err)
	}
	oldIdentity, _ := oldRegistry.IdentityFor("example_versioned")

	writeVersion("2.0.0")
	pending, err := supervisor.Discover(root)
	if err != nil || pending.Version != "2.0.0" {
		t.Fatalf("pending = %#v, err = %v", pending, err)
	}
	if _, err := supervisor.Enable(context.Background(), status.ID); err != nil {
		t.Fatal(err)
	}
	newRegistry := NewRegistry()
	if err := supervisor.RegisterReadyTools(status.ID, newRegistry); err != nil {
		t.Fatal(err)
	}
	newIdentity, _ := newRegistry.IdentityFor("example_versioned")
	if oldIdentity.RegistrationID == newIdentity.RegistrationID || oldIdentity.ImplementationHash == newIdentity.ImplementationHash {
		t.Fatalf("old = %#v new = %#v, want immutable implementation identities", oldIdentity, newIdentity)
	}
	oldResult := NewToolRuntime(oldRegistry, t.TempDir()).ExecuteWithContext(context.Background(), domain.ChatToolCall{Name: "example_versioned", Arguments: json.RawMessage(`{}`)}, domain.ToolExecutionContext{ExpectedRegistrations: map[string]domain.ToolRegistrationIdentity{"example_versioned": oldIdentity}})
	newResult := NewToolRuntime(newRegistry, t.TempDir()).ExecuteWithContext(context.Background(), domain.ChatToolCall{Name: "example_versioned", Arguments: json.RawMessage(`{}`)}, domain.ToolExecutionContext{ExpectedRegistrations: map[string]domain.ToolRegistrationIdentity{"example_versioned": newIdentity}})
	if !oldResult.OK || oldResult.Content != "1.0.0" || !newResult.OK || newResult.Content != "2.0.0" {
		t.Fatalf("old result = %#v new result = %#v", oldResult, newResult)
	}

	writeVersion("broken")
	if _, err := supervisor.Discover(root); err != nil {
		t.Fatal(err)
	}
	if _, err := supervisor.Enable(context.Background(), status.ID); err == nil {
		t.Fatal("broken update unexpectedly replaced the ready generation")
	}
	currentRegistry := NewRegistry()
	if err := supervisor.RegisterReadyTools(status.ID, currentRegistry); err != nil {
		t.Fatal(err)
	}
	currentResult := NewToolRuntime(currentRegistry, t.TempDir()).Execute(context.Background(), domain.ChatToolCall{Name: "example_versioned", Arguments: json.RawMessage(`{}`)})
	if !currentResult.OK || currentResult.Content != "2.0.0" {
		t.Fatalf("current result after failed update = %#v", currentResult)
	}
	_, _ = supervisor.Stop(context.Background(), status.ID)
	removedRegistry := NewRegistry()
	if err := supervisor.RegisterAllReadyTools(removedRegistry); err != nil {
		t.Fatal(err)
	}
	if _, ok := removedRegistry.Get("example_versioned"); ok {
		t.Fatal("stopped extension remained executable")
	}
}

func TestExtensionRemoveShutsDownRetiredGenerations(t *testing.T) {
	writeStaticVersion := func(root string, version string) LoadedExtension {
		writeTestExtensionManifest(t, root, map[string]any{
			"schemaVersion": 2, "id": "com.example.retired", "name": "Retired", "version": version, "apiVersion": "2",
			"runtime": map[string]any{"type": "static"},
		})
		loaded, err := LoadExtensionManifest(root)
		if err != nil {
			t.Fatal(err)
		}
		return loaded
	}
	first := writeStaticVersion(t.TempDir(), "1.0.0")
	second := writeStaticVersion(t.TempDir(), "2.0.0")
	supervisor := NewExtensionSupervisor()
	if _, err := supervisor.Discover(first.Root); err != nil {
		t.Fatal(err)
	}
	if _, err := supervisor.Enable(context.Background(), first.Manifest.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := supervisor.Discover(second.Root); err != nil {
		t.Fatal(err)
	}
	if _, err := supervisor.Enable(context.Background(), second.Manifest.ID); err != nil {
		t.Fatal(err)
	}
	retired := supervisor.retired[first.Manifest.ID][first.Integrity]
	if retired == nil || retired.client == nil {
		t.Fatal("first generation was not retained during update")
	}
	if err := supervisor.Remove(context.Background(), first.Manifest.ID); err != nil {
		t.Fatal(err)
	}
	if retired.client != nil || supervisor.items[first.Manifest.ID] != nil || supervisor.retired[first.Manifest.ID] != nil {
		t.Fatal("remove did not tear down all extension generations")
	}
}

func TestExecutableExtensionRequiresIntegrityBoundTrust(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "extension"), "#!/bin/sh\nexit 0\n", 0o700)
	writeTestExtensionManifest(t, root, map[string]any{
		"schemaVersion": 2, "id": "com.example.process", "name": "Process", "version": "1.0.0", "apiVersion": "2",
		"runtime": map[string]any{"type": "process", "command": "extension", "transport": "stdio"},
	})
	supervisor := NewExtensionSupervisor()
	status, err := supervisor.Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != domain.ExtensionStateUntrusted || status.Trusted {
		t.Fatalf("status = %#v", status)
	}
	if _, err := supervisor.Enable(context.Background(), status.ID); err == nil {
		t.Fatal("untrusted executable was enabled")
	}
	if _, err := supervisor.Trust(status.ID, "wrong"); err == nil {
		t.Fatal("wrong integrity was trusted")
	}
	trusted, err := supervisor.Trust(status.ID, status.Integrity)
	if err != nil || !trusted.Trusted || trusted.State != domain.ExtensionStateValidated {
		t.Fatalf("trusted = %#v, err = %v", trusted, err)
	}
}

func TestExtensionCredentialBrokerRequiresDeclaredBoundOperationSlot(t *testing.T) {
	store := NewMemorySecretStore()
	if err := store.Put(context.Background(), "secure/github", "token-value"); err != nil {
		t.Fatal(err)
	}
	broker := NewHostCredentialBroker(store)
	if err := broker.Bind("com.example.github", "github", "secure/github"); err != nil {
		t.Fatal(err)
	}
	if _, err := broker.Request(context.Background(), "com.example.github", []string{"github"}, "github", ""); err == nil {
		t.Fatal("ownerless credential lease succeeded")
	}
	if _, err := broker.Request(context.Background(), "com.example.github", []string{"github"}, "other", "op-1"); err == nil {
		t.Fatal("undeclared credential slot succeeded")
	}
	value, err := broker.Request(context.Background(), "com.example.github", []string{"github"}, "github", "op-1")
	if err != nil || value != "token-value" {
		t.Fatalf("value = %q, err = %v", value, err)
	}
	broker.Release("op-1")
}

func TestExtensionProcessProtocolCredentialsCatalogAndLifecycle(t *testing.T) {
	root := t.TempDir()
	script := "#!/bin/sh\nexec " + strconv.Quote(os.Args[0]) + " -test.run=TestExtensionProcessHelper -- extension-process\n"
	writeTestFile(t, filepath.Join(root, "bin", "extension"), script, 0o700)
	writeTestExtensionManifest(t, root, map[string]any{
		"schemaVersion": 2, "id": "com.example.process", "name": "Process", "version": "1.0.0", "apiVersion": "2",
		"runtime":      map[string]any{"type": "process", "command": "bin/extension", "transport": "stdio"},
		"requirements": map[string]any{"credentials": []string{"demo"}},
		"contributes":  map[string]any{"tools": []any{map[string]any{"name": "example_echo", "description": "Echo", "schema": map[string]any{"type": "object"}, "activation": "auto"}}},
	})
	secretStore := NewMemorySecretStore()
	if err := secretStore.Put(context.Background(), "secret/demo", "credential-value"); err != nil {
		t.Fatal(err)
	}
	broker := NewHostCredentialBroker(secretStore)
	if err := broker.Bind("com.example.process", "demo", "secret/demo"); err != nil {
		t.Fatal(err)
	}
	supervisor := NewExtensionSupervisor()
	supervisor.SetCredentialBroker(broker)
	status, err := supervisor.Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := supervisor.Trust(status.ID, status.Integrity); err != nil {
		t.Fatal(err)
	}
	if _, err := supervisor.Enable(context.Background(), status.ID); err != nil {
		t.Fatal(err)
	}
	result := supervisor.Execute(context.Background(), status.ID, "example_echo", json.RawMessage(`{"text":"hello"}`), domain.ToolExecutionContext{ToolCallID: "op-1"})
	if !result.OK || result.Content != "credential accepted" {
		t.Fatalf("result = %#v", result)
	}
	if _, err := supervisor.Stop(context.Background(), status.ID); err != nil {
		t.Fatal(err)
	}
}

func TestExtensionProcessHelper(t *testing.T) {
	if !hasTestArgument("extension-process") {
		return
	}
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return
		}
		var request struct {
			ID     string         `json:"id"`
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
		}
		if json.Unmarshal(line, &request) != nil {
			continue
		}
		result := map[string]any{}
		switch request.Method {
		case "catalog/list":
			result["tools"] = []any{map[string]any{"name": "example_echo", "schemaHash": toolSchemaHash(domain.ToolSpec{InputSchema: map[string]any{"type": "object"}})}}
		case "tool/execute":
			opID, _ := request.Params["operationId"].(string)
			writeExtensionTestFrame(writer, map[string]any{"jsonrpc": "2.0", "id": "credential-1", "method": "credential/request", "params": map[string]any{"slot": "demo", "operationId": opID}})
			credentialLine, readErr := reader.ReadBytes('\n')
			if readErr != nil || !strings.Contains(string(credentialLine), "credential-value") {
				return
			}
			writeExtensionTestFrame(writer, map[string]any{"jsonrpc": "2.0", "method": "tool/progress", "params": map[string]any{"operationId": opID, "message": "working"}})
			result = map[string]any{"ok": true, "content": "credential accepted"}
		}
		writeExtensionTestFrame(writer, map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
		if request.Method == "extension/shutdown" {
			return
		}
	}
}

func TestDynamicExtensionServiceEndpointHandshakeAndViewRouting(t *testing.T) {
	firstSupervisor, firstStatus := startDynamicTestExtension(t, "com.example.dynamic_one")
	defer func() { _, _ = firstSupervisor.Stop(context.Background(), firstStatus.ID) }()
	secondSupervisor, secondStatus := startDynamicTestExtension(t, "com.example.dynamic_two")
	defer func() { _, _ = secondSupervisor.Stop(context.Background(), secondStatus.ID) }()

	firstView, err := firstSupervisor.ResolveView(context.Background(), firstStatus.ID, "detail")
	if err != nil {
		t.Fatal(err)
	}
	secondView, err := secondSupervisor.ResolveView(context.Background(), secondStatus.ID, "detail")
	if err != nil {
		t.Fatal(err)
	}
	firstURL, firstErr := url.Parse(firstView.BackendURL)
	secondURL, secondErr := url.Parse(secondView.BackendURL)
	if firstErr != nil || secondErr != nil || firstURL.Port() == "" || secondURL.Port() == "" || firstURL.Port() == secondURL.Port() {
		t.Fatalf("dynamic view URLs = %q, %q; parse errors = %v, %v", firstView.BackendURL, secondView.BackendURL, firstErr, secondErr)
	}
	if firstURL.Path != "/ui" || firstView.BackendToken == "" || strings.Contains(firstView.LogicalURL, firstURL.Port()) {
		t.Fatalf("first dynamic view descriptor = %#v", firstView)
	}

	result := firstSupervisor.Execute(context.Background(), firstStatus.ID, "dynamic_service", json.RawMessage(`{"message":"hello"}`), domain.ToolExecutionContext{ToolCallID: "dynamic-op"})
	if !result.OK || result.Content != "dynamic service ok" {
		t.Fatalf("dynamic tool result = %#v", result)
	}
	action, err := firstSupervisor.InvokeViewAction(context.Background(), firstStatus.ID, "detail", "view.refresh", map[string]any{"operationId": "dynamic-op"})
	if err != nil || action.(map[string]any)["ok"] != true {
		t.Fatalf("dynamic action = %#v, err = %v", action, err)
	}
}

func TestDynamicServiceReadinessRejectsInvalidFramesAndOrigins(t *testing.T) {
	valid := `{"protocol":"aivo-extension-service/1","url":"http://127.0.0.1:49152"}` + "\n"
	if endpoint, err := readDynamicServiceEndpoint(context.Background(), io.NopCloser(strings.NewReader(valid))); err != nil || endpoint != "http://127.0.0.1:49152" {
		t.Fatalf("valid endpoint = %q, err = %v", endpoint, err)
	}

	tests := []struct {
		name  string
		frame string
	}{
		{name: "missing newline", frame: strings.TrimSuffix(valid, "\n")},
		{name: "invalid JSON", frame: "not-json\n"},
		{name: "wrong protocol", frame: `{"protocol":"other","url":"http://127.0.0.1:49152"}` + "\n"},
		{name: "public origin", frame: `{"protocol":"aivo-extension-service/1","url":"http://example.com:49152"}` + "\n"},
		{name: "zero port", frame: `{"protocol":"aivo-extension-service/1","url":"http://127.0.0.1:0"}` + "\n"},
		{name: "credentials", frame: `{"protocol":"aivo-extension-service/1","url":"http://user@127.0.0.1:49152"}` + "\n"},
		{name: "path", frame: `{"protocol":"aivo-extension-service/1","url":"http://127.0.0.1:49152/api"}` + "\n"},
		{name: "query", frame: `{"protocol":"aivo-extension-service/1","url":"http://127.0.0.1:49152/?token=x"}` + "\n"},
		{name: "oversized", frame: strings.Repeat("x", extensionServiceHandshakeMaxFrame+1) + "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if endpoint, err := readDynamicServiceEndpoint(context.Background(), io.NopCloser(strings.NewReader(tt.frame))); err == nil {
				t.Fatalf("endpoint = %q, want refusal", endpoint)
			}
		})
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := readDynamicServiceEndpoint(cancelled, io.NopCloser(strings.NewReader(""))); err == nil {
		t.Fatal("cancelled readiness was accepted")
	}

	reader, writer := io.Pipe()
	defer writer.Close()
	if _, err := readDynamicServiceEndpointWithTimeout(context.Background(), reader, 5*time.Millisecond); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("timeout error = %v", err)
	}
}

func TestDynamicExtensionServiceHelperProcess(t *testing.T) {
	if !hasTestArgument("dynamic-extension-service") {
		return
	}
	token := os.Getenv("AIVO_EXTENSION_BEARER_TOKEN")
	if len(token) != 64 {
		os.Exit(2)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		os.Exit(3)
	}
	address := listener.Addr().(*net.TCPAddr)
	_, _ = fmt.Fprintf(os.Stdout, `{"protocol":"%s","url":"http://127.0.0.1:%d"}`+"\n", extensionServiceHandshakeProtocol, address.Port)
	service := &http.Server{}
	service.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var request struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if json.NewDecoder(r.Body).Decode(&request) != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		result := map[string]any{}
		switch request.Method {
		case "catalog/list":
			result["tools"] = []any{map[string]any{"name": "dynamic_service", "schemaHash": toolSchemaHash(domain.ToolSpec{InputSchema: map[string]any{"type": "object"}})}}
		case "tool/execute":
			result = map[string]any{"ok": true, "content": "dynamic service ok"}
		case "ui/event":
			result = map[string]any{"ok": true}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
		if request.Method == "extension/shutdown" {
			go func() {
				time.Sleep(10 * time.Millisecond)
				os.Exit(0)
			}()
		}
	})
	if service.Serve(listener) != nil {
		os.Exit(4)
	}
}

func startDynamicTestExtension(t *testing.T, id string) (*ExtensionSupervisor, domain.ExtensionStatus) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	executableRaw, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	helperName := "dynamic-service-helper" + filepath.Ext(executable)
	helperPath := filepath.Join(root, "bin", helperName)
	if err := os.MkdirAll(filepath.Dir(helperPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(helperPath, executableRaw, 0o700); err != nil {
		t.Fatal(err)
	}
	writeTestExtensionManifest(t, root, map[string]any{
		"schemaVersion": 2, "id": id, "name": "Dynamic service", "version": "1", "apiVersion": "2",
		"runtime": map[string]any{
			"type": "service", "command": filepath.ToSlash(filepath.Join("bin", helperName)), "transport": "dynamic-http",
			"args": []string{"-test.run=TestDynamicExtensionServiceHelperProcess$", "--", "dynamic-extension-service"},
		},
		"contributes": map[string]any{
			"tools": []any{map[string]any{"name": "dynamic_service", "description": "Dynamic", "schema": map[string]any{"type": "object"}}},
			"views": []any{map[string]any{"id": "detail", "title": "Detail", "type": "web", "route": "/ui", "surfaces": []string{"tool-detail"}, "tools": []string{"dynamic_service"}, "actions": []string{"view.refresh"}}},
		},
	})
	supervisor := NewExtensionSupervisor()
	status, err := supervisor.Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if status, err = supervisor.Trust(status.ID, status.Integrity); err != nil {
		t.Fatal(err)
	}
	if status, err = supervisor.Enable(context.Background(), status.ID); err != nil {
		t.Fatal(err)
	}
	return supervisor, status
}

func TestServiceExternalAndStaticExtensionRuntimes(t *testing.T) {
	schemaHash := toolSchemaHash(domain.ToolSpec{InputSchema: map[string]any{"type": "object"}})
	serviceHandler := func(expectedBearer func(string) bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !expectedBearer(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			var request struct {
				ID     string `json:"id"`
				Method string `json:"method"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			result := map[string]any{}
			if request.Method == "catalog/list" {
				result["tools"] = []any{map[string]any{"name": "example_service", "schemaHash": schemaHash}}
			}
			if request.Method == "tool/execute" {
				result = map[string]any{"ok": true, "content": "service ok"}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
		}
	}

	localServer := httptest.NewServer(serviceHandler(func(value string) bool { return len(value) == 64 }))
	defer localServer.Close()
	localRoot := t.TempDir()
	writeTestFile(t, filepath.Join(localRoot, "bin", "service"), "#!/bin/sh\nsleep 30\n", 0o700)
	writeTestExtensionManifest(t, localRoot, map[string]any{
		"schemaVersion": 2, "id": "com.example_service", "name": "Service", "version": "1", "apiVersion": "2",
		"runtime":     map[string]any{"type": "service", "command": "bin/service", "url": localServer.URL},
		"permissions": []any{"runtime.messaging"},
		"contributes": map[string]any{
			"tools": []any{map[string]any{"name": "example_service", "schema": map[string]any{"type": "object"}}},
			"views": []any{map[string]any{"id": "detail", "title": "Detail", "type": "web", "route": "/ui", "surfaces": []string{"tool-detail"}, "tools": []string{"example_service"}, "actions": []string{"view.refresh"}}},
		},
	})
	localSupervisor := NewExtensionSupervisor()
	localSupervisor.idleTimeout = 25 * time.Millisecond
	localStatus, err := localSupervisor.Discover(localRoot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := localSupervisor.Trust(localStatus.ID, localStatus.Integrity); err != nil {
		t.Fatal(err)
	}
	if _, err := localSupervisor.Enable(context.Background(), localStatus.ID); err != nil {
		t.Fatal(err)
	}
	view, err := localSupervisor.ResolveView(context.Background(), localStatus.ID, "detail")
	if err != nil || view.BackendToken == "" || !strings.HasPrefix(view.LogicalURL, "aivo-extension://") || !reflect.DeepEqual(view.Permissions, []string{"runtime.messaging"}) {
		t.Fatalf("view = %#v, err = %v", view, err)
	}
	if err := localSupervisor.OpenView(context.Background(), localStatus.ID); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if status, _ := localSupervisor.Status(localStatus.ID); status.State != domain.ExtensionStateActive {
		t.Fatalf("view did not prevent idle stop: %#v", status)
	}
	if err := localSupervisor.CloseView(context.Background(), localStatus.ID); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if status, _ := localSupervisor.Status(localStatus.ID); status.State != domain.ExtensionStateReady || !status.Enabled {
		t.Fatalf("idle status = %#v, want enabled lazy-ready", status)
	}
	if result := localSupervisor.Execute(context.Background(), localStatus.ID, "example_service", json.RawMessage(`{}`), domain.ToolExecutionContext{ToolCallID: "local-op"}); !result.OK {
		t.Fatalf("lazy restart result = %#v", result)
	}
	_, _ = localSupervisor.Stop(context.Background(), localStatus.ID)

	externalServer := httptest.NewTLSServer(serviceHandler(func(value string) bool { return value == "external-token" }))
	defer externalServer.Close()
	externalRoot := t.TempDir()
	writeTestExtensionManifest(t, externalRoot, map[string]any{
		"schemaVersion": 2, "id": "com.example.external", "name": "External", "version": "1", "apiVersion": "2",
		"runtime":      map[string]any{"type": "external", "url": externalServer.URL},
		"requirements": map[string]any{"credentials": []string{"service"}},
		"contributes": map[string]any{
			"tools": []any{map[string]any{"name": "example_service", "schema": map[string]any{"type": "object"}}},
			"views": []any{map[string]any{"id": "external-view", "title": "External", "type": "web", "route": "/ui", "surfaces": []string{"page"}}},
		},
	})
	store := NewMemorySecretStore()
	_ = store.Put(context.Background(), "external/ref", "external-token")
	broker := NewHostCredentialBroker(store)
	_ = broker.Bind("com.example.external", "service", "external/ref")
	externalSupervisor := NewExtensionSupervisor()
	externalSupervisor.httpClient = externalServer.Client()
	externalSupervisor.SetCredentialBroker(broker)
	externalStatus, err := externalSupervisor.Discover(externalRoot)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = externalSupervisor.Trust(externalStatus.ID, externalStatus.Integrity)
	if _, err := externalSupervisor.Enable(context.Background(), externalStatus.ID); err != nil {
		t.Fatal(err)
	}
	if result := externalSupervisor.Execute(context.Background(), externalStatus.ID, "example_service", json.RawMessage(`{}`), domain.ToolExecutionContext{ToolCallID: "external-op"}); !result.OK {
		t.Fatalf("external result = %#v", result)
	}
	descriptor, err := externalSupervisor.ResolveView(context.Background(), externalStatus.ID, "external-view")
	if err != nil || descriptor.BackendToken != "external-token" {
		t.Fatalf("external view descriptor = %#v, err = %v; want Host-bound proxy token", descriptor, err)
	}
	_, _ = externalSupervisor.Stop(context.Background(), externalStatus.ID)

	staticRoot := t.TempDir()
	writeTestFile(t, filepath.Join(staticRoot, "context", "guide.md"), "safe context", 0o600)
	writeTestExtensionManifest(t, staticRoot, map[string]any{
		"schemaVersion": 2, "id": "com.example.static", "name": "Static", "version": "1", "apiVersion": "2",
		"runtime":     map[string]any{"type": "static"},
		"contributes": map[string]any{"contexts": []any{map[string]any{"id": "guide", "kind": "skill", "path": "context/guide.md"}}},
	})
	staticSupervisor := NewExtensionSupervisor()
	staticStatus, err := staticSupervisor.Discover(staticRoot)
	if err != nil {
		t.Fatal(err)
	}
	if staticStatus, err = staticSupervisor.Enable(context.Background(), staticStatus.ID); err != nil || staticStatus.State != domain.ExtensionStateReady {
		t.Fatalf("static status = %#v, err = %v", staticStatus, err)
	}
	contexts, err := staticSupervisor.ContextResources(staticStatus.ID)
	if err != nil || len(contexts) != 1 || contexts[0].Content != "safe context" || contexts[0].SHA256 == "" {
		t.Fatalf("static contexts = %#v, err = %v", contexts, err)
	}
}

func writeExtensionTestFrame(writer *bufio.Writer, value any) {
	raw, _ := json.Marshal(value)
	_, _ = writer.Write(append(raw, '\n'))
	_ = writer.Flush()
}

func hasTestArgument(value string) bool {
	for _, arg := range os.Args {
		if arg == value {
			return true
		}
	}
	return false
}

func writeTestExtensionManifest(t *testing.T, root string, manifest map[string]any) {
	t.Helper()
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, "aivo.extension.json"), string(raw), 0o600)
}

func writeTestFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}
