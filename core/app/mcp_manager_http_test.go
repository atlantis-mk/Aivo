package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aivo/core/domain"
)

func TestMCPStreamableHTTPProbeAndPrompt(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	t.Setenv("AIVO_TEST_MCP_TOKEN", "test-token")
	requests := 0
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization = %q, want bearer token from env", got)
		}
		if r.Header.Get("MCP-Protocol-Version") == "" {
			t.Fatalf("missing MCP-Protocol-Version header")
		}
		var request struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": mcpHelperResult(request.Method)})
	}))
	defer httpServer.Close()

	server := domain.MCPServerConfig{
		ID:        "http_mcp",
		Name:      "http_mcp",
		Transport: domain.MCPTransportStreamableHTTP,
		URL:       httpServer.URL,
		AuthType:  domain.MCPAuthBearer, BearerTokenEnv: "AIVO_TEST_MCP_TOKEN",
	}
	if _, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: server}); err != nil {
		t.Fatal(err)
	}
	probe, err := service.ProbeMCPServer(ctx, domain.MCPProbeInput{ServerID: server.ID})
	if err != nil || !probe.OK {
		t.Fatalf("probe = %#v err = %v, want ok", probe, err)
	}
	if len(probe.Tools) != 1 || probe.Tools[0].Name != "echo" || len(probe.ResourceTemplates) != 1 {
		t.Fatalf("probe = %#v, want helper capabilities", probe)
	}
	prompt, err := service.GetMCPPrompt(ctx, domain.MCPPromptGetInput{ServerID: server.ID, Name: "review", Arguments: map[string]string{"path": "README.md"}})
	if err != nil {
		t.Fatal(err)
	}
	if prompt.Content != "user: Review README.md" {
		t.Fatalf("prompt = %#v, want normalized prompt content", prompt)
	}
	if requests < 6 {
		t.Fatalf("requests = %d, want probe and prompt calls", requests)
	}
}

func TestMCPSSEProbeParsesEventStreamResponses(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": mcpHelperResult(request.Method)})
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: message\n"))
		_, _ = w.Write([]byte("data: " + string(raw) + "\n\n"))
	}))
	defer httpServer.Close()

	result, err := probeMCPServer(context.Background(), domain.MCPServerConfig{
		ID:        "sse_mcp",
		Name:      "sse_mcp",
		Transport: domain.MCPTransportSSE,
		URL:       httpServer.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Tools) != 1 || result.Tools[0].Name != "echo" || len(result.Prompts) != 1 {
		t.Fatalf("result = %#v, want helper capabilities from SSE", result)
	}
}

func TestMCPHTTPRespondsToRootsListRequest(t *testing.T) {
	root := t.TempDir()
	rootsResponse := make(chan []any, 1)
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if method, _ := request["method"].(string); method == "" {
			result, _ := request["result"].(map[string]any)
			roots, _ := result["roots"].([]any)
			rootsResponse <- roots
			w.WriteHeader(http.StatusAccepted)
			return
		}
		requestID, _ := request["id"].(string)
		method, _ := request["method"].(string)
		if method == "initialize" {
			params, _ := request["params"].(map[string]any)
			capabilities, _ := params["capabilities"].(map[string]any)
			if _, ok := capabilities["roots"].(map[string]any); !ok {
				t.Fatalf("initialize params = %#v, want roots capability", params)
			}
			rootRequest, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": "roots-request", "method": "roots/list"})
			initResponse, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": requestID, "result": mcpHelperResult(method)})
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: message\n"))
			_, _ = w.Write([]byte("data: " + string(rootRequest) + "\n\n"))
			_, _ = w.Write([]byte("event: message\n"))
			_, _ = w.Write([]byte("data: " + string(initResponse) + "\n\n"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": requestID, "result": mcpHelperResult(method)})
	}))
	defer httpServer.Close()

	result, err := probeMCPServer(context.Background(), domain.MCPServerConfig{
		ID:        "http_roots_mcp",
		Name:      "http_roots_mcp",
		Transport: domain.MCPTransportStreamableHTTP,
		URL:       httpServer.URL,
		Roots:     []string{root},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Tools) != 1 || result.Tools[0].Name != "echo" {
		t.Fatalf("result = %#v, want helper capabilities after HTTP roots/list", result)
	}
	select {
	case roots := <-rootsResponse:
		if len(roots) != 1 {
			t.Fatalf("roots response = %#v, want one root", roots)
		}
		item, _ := roots[0].(map[string]any)
		uri, _ := item["uri"].(string)
		if !strings.HasPrefix(uri, "file://") {
			t.Fatalf("roots response = %#v, want file URI", roots)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for HTTP roots/list response")
	}
}

func TestMCPHTTPRefreshesToolsAfterListChangedNotification(t *testing.T) {
	toolsListCalls := 0
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		switch request.Method {
		case "tools/list":
			toolsListCalls++
			name := "before"
			if toolsListCalls > 1 {
				name = "after"
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": mcpToolsChangedHelperTools(name)})
		case "prompts/list":
			response, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": mcpHelperResult(request.Method)})
			notification, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "notifications/tools/list_changed"})
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: message\n"))
			_, _ = w.Write([]byte("data: " + string(notification) + "\n\n"))
			_, _ = w.Write([]byte("event: message\n"))
			_, _ = w.Write([]byte("data: " + string(response) + "\n\n"))
		default:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": mcpHelperResult(request.Method)})
		}
	}))
	defer httpServer.Close()

	result, err := probeMCPServer(context.Background(), domain.MCPServerConfig{
		ID:        "changed_http_mcp",
		Name:      "changed_http_mcp",
		Transport: domain.MCPTransportStreamableHTTP,
		URL:       httpServer.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if toolsListCalls != 2 {
		t.Fatalf("toolsListCalls = %d, want refresh after list_changed notification", toolsListCalls)
	}
	if len(result.Tools) != 1 || result.Tools[0].Name != "after" {
		t.Fatalf("tools = %#v, want refreshed after tool", result.Tools)
	}
	if len(result.Prompts) != 1 {
		t.Fatalf("prompts = %#v, want prompt response after notification event", result.Prompts)
	}
}
