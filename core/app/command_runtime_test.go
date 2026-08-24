package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestExpandCommandTemplateUsesNamedPositionalAndArgumentsPlaceholders(t *testing.T) {
	prompt, err := expandCommandTemplate(domain.CommandTemplateDefinition{
		Template:  "Review {{path}} with $focus ($1 $2): $ARGUMENTS",
		Arguments: []domain.CommandArgument{{Name: "path", Required: true}, {Name: "focus", Default: "correctness"}},
	}, map[string]string{"path": "README.md"})
	if err != nil {
		t.Fatal(err)
	}
	if prompt != "Review README.md with correctness (README.md correctness): README.md correctness" {
		t.Fatalf("prompt = %q", prompt)
	}
}

func TestExpandCommandTemplateRejectsMissingUnknownAndUnresolvedArguments(t *testing.T) {
	definition := domain.CommandTemplateDefinition{Template: "Review {{path}}", Arguments: []domain.CommandArgument{{Name: "path", Required: true}}}
	if _, err := expandCommandTemplate(definition, nil); err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("missing argument error = %v", err)
	}
	if _, err := expandCommandTemplate(definition, map[string]string{"path": "x", "extra": "y"}); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("unknown argument error = %v", err)
	}
	if _, err := expandCommandTemplate(domain.CommandTemplateDefinition{Template: "{{missing}}"}, nil); err == nil || !strings.Contains(err.Error(), "unresolved") {
		t.Fatalf("unresolved argument error = %v", err)
	}
}

func TestExpandCommandTemplateAcceptsRawArguments(t *testing.T) {
	prompt, err := expandCommandTemplate(domain.CommandTemplateDefinition{Template: "Explain $ARGUMENTS"}, map[string]string{"ARGUMENTS": "one two"})
	if err != nil {
		t.Fatal(err)
	}
	if prompt != "Explain one two" {
		t.Fatalf("prompt = %q", prompt)
	}
}

func TestCommandCatalogIncludesBuiltinsAndMarkdownCommands(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	writeRuntimeConfigTestFile(t, filepath.Join(root, ".aivo", "commands", "team", "audit.md"), `---
description: Team audit
subtask: true
---
Audit $ARGUMENTS.`)
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	entries, err := service.ListCommandCatalog(context.Background(), domain.CommandCatalogInput{ProjectPath: root})
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]domain.CommandCatalogEntry{}
	for _, entry := range entries {
		found[entry.ID] = entry
	}
	if found["builtin:init"].Name != "init" || !found["builtin:review"].Subtask || !found["config:team/audit"].Subtask {
		t.Fatalf("catalog entries = %#v", entries)
	}
}

func TestBuiltinReviewCommandRunsInForkedSubtaskAndReturnsResult(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []map[string]any `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body.Messages) == 0 || !strings.Contains(body.Messages[len(body.Messages)-1]["content"].(string), "Review HEAD~1") {
			t.Errorf("messages = %#v", body.Messages)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"No blocking findings."}}]}`))
	}))
	defer server.Close()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "custom-api", Type: "openai-compatible", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}
	parent, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: root})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.InvokeCommand(ctx, domain.InvokeCommandInput{SessionID: parent.ID, ProjectPath: root, CommandID: "builtin:review", Arguments: map[string]string{"ARGUMENTS": "HEAD~1"}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Subtask || result.ChildSessionID == "" || result.AgentRunID == "" || result.Response != "No blocking findings." {
		t.Fatalf("result = %#v", result)
	}
	child, err := service.GetRuntimeSession(ctx, result.ChildSessionID)
	if err != nil || child.ParentSessionID != parent.ID || child.AgentMode != domain.AgentModeAssistant {
		t.Fatalf("child = %#v err = %v", child, err)
	}
	events, err := service.ListEvents(ctx, parent.ID, false, 50)
	if err != nil || !sessionEventContains(events, domain.EventTypeAssistantMessage, "No blocking findings.") {
		t.Fatalf("parent events = %#v err = %v", events, err)
	}
}
