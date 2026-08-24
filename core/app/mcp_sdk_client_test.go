package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"aivo/core/domain"
)

type sdkEchoInput struct {
	Text string `json:"text"`
}

func TestMCPOfficialSDKStreamableHTTPInterop(t *testing.T) {
	t.Setenv("AIVO_TEST_MCP_TOKEN", "test-token")
	initialized := make(chan struct{}, 1)
	authorized := make(chan struct{}, 1)
	server := mcp.NewServer(&mcp.Implementation{Name: "aivo-test-server", Version: "1.0.0"}, &mcp.ServerOptions{
		InitializedHandler: func(context.Context, *mcp.InitializedRequest) {
			initialized <- struct{}{}
		},
	})
	mcp.AddTool(server, &mcp.Tool{Name: "echo", Description: "Echo text"}, func(_ context.Context, _ *mcp.CallToolRequest, input sdkEchoInput) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: input.Text}}}, nil, nil
	})
	sdkHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{JSONResponse: true})
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("authorization = %q, want configured bearer token", r.Header.Get("Authorization"))
		} else {
			select {
			case authorized <- struct{}{}:
			default:
			}
		}
		sdkHandler.ServeHTTP(w, r)
	}))
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := startMCPSDKClient(ctx, domain.MCPServerConfig{
		ID: "official-sdk", Name: "official-sdk", Transport: domain.MCPTransportStreamableHTTP, URL: httpServer.URL,
		AuthType: domain.MCPAuthBearer, BearerTokenEnv: "AIVO_TEST_MCP_TOKEN",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.close()

	select {
	case <-initialized:
	case <-ctx.Done():
		t.Fatal("official SDK server did not receive notifications/initialized")
	}
	select {
	case <-authorized:
	default:
		t.Fatal("official SDK requests did not pass through Aivo authentication headers")
	}
	if client.session.InitializeResult().ProtocolVersion != "2025-11-25" {
		t.Fatalf("protocol version = %q, want 2025-11-25", client.session.InitializeResult().ProtocolVersion)
	}
	if client.session.ID() == "" {
		t.Fatal("streamable HTTP session ID was not retained")
	}
	result, err := client.call(ctx, "tools/call", map[string]any{"name": "echo", "arguments": map[string]any{"text": "hello"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := textFromMCPToolContent(result); got != "hello" {
		t.Fatalf("tool content = %q, want hello", got)
	}
}

type pagedMCPClient struct {
	calls []string
}

func (c *pagedMCPClient) call(_ context.Context, _ string, params map[string]any) (map[string]any, error) {
	cursor, _ := params["cursor"].(string)
	c.calls = append(c.calls, cursor)
	if cursor == "" {
		return map[string]any{"tools": []any{map[string]any{"name": "first"}}, "nextCursor": "page-2"}, nil
	}
	return map[string]any{"tools": []any{map[string]any{"name": "second"}}}, nil
}

func (*pagedMCPClient) consumeToolsListChanged() bool { return false }
func (*pagedMCPClient) close()                        {}

func TestListAllMCPPagesFollowsNextCursor(t *testing.T) {
	client := &pagedMCPClient{}
	result, err := listAllMCPPages(context.Background(), client, "tools/list", "tools")
	if err != nil {
		t.Fatal(err)
	}
	items, _ := result["tools"].([]any)
	if len(items) != 2 || len(client.calls) != 2 || client.calls[1] != "page-2" {
		t.Fatalf("result = %#v calls = %#v, want two pages", result, client.calls)
	}
}
