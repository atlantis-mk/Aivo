package app

import (
	"bufio"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestMCPProbeHelperProcess(t *testing.T) {
	if !hasArg("mcp-helper") {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var request struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil || request.ID == "" {
			continue
		}
		writeMCPHelperResponse(request.ID, mcpHelperResult(request.Method))
	}
	os.Exit(0)
}

func TestMCPRootsHelperProcess(t *testing.T) {
	if !hasArg("mcp-roots-helper") {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		os.Exit(1)
	}
	var initRequest struct {
		ID     string         `json:"id"`
		Method string         `json:"method"`
		Params map[string]any `json:"params"`
	}
	if err := json.Unmarshal(scanner.Bytes(), &initRequest); err != nil || initRequest.ID == "" || initRequest.Method != "initialize" {
		os.Exit(1)
	}
	rawRootRequest, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": "roots-request", "method": "roots/list"})
	_, _ = os.Stdout.Write(append(rawRootRequest, '\n'))
	if !scanner.Scan() {
		os.Exit(1)
	}
	var rootsResponse struct {
		Result struct {
			Roots []struct {
				URI string `json:"uri"`
			} `json:"roots"`
		} `json:"result"`
	}
	if err := json.Unmarshal(scanner.Bytes(), &rootsResponse); err != nil {
		writeMCPHelperError(initRequest.ID, "invalid roots/list response")
		os.Exit(0)
	}
	if len(rootsResponse.Result.Roots) != 1 || !strings.HasPrefix(rootsResponse.Result.Roots[0].URI, "file://") {
		writeMCPHelperError(initRequest.ID, "unexpected roots/list response: "+string(scanner.Bytes()))
		os.Exit(0)
	}
	capabilities, _ := initRequest.Params["capabilities"].(map[string]any)
	if _, ok := capabilities["roots"].(map[string]any); !ok {
		writeMCPHelperError(initRequest.ID, "initialize did not advertise roots capability")
		os.Exit(0)
	}
	writeMCPHelperResponse(initRequest.ID, mcpHelperResult("initialize"))
	for scanner.Scan() {
		var request struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil || request.ID == "" {
			continue
		}
		writeMCPHelperResponse(request.ID, mcpHelperResult(request.Method))
	}
	os.Exit(0)
}

func TestMCPToolsChangedHelperProcess(t *testing.T) {
	if !hasArg("mcp-tools-changed-helper") {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	toolsListCalls := 0
	for scanner.Scan() {
		var request struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil || request.ID == "" {
			continue
		}
		switch request.Method {
		case "tools/list":
			toolsListCalls++
			name := "before"
			if toolsListCalls > 1 {
				name = "after"
			}
			writeMCPHelperResponse(request.ID, mcpToolsChangedHelperTools(name))
		case "prompts/list":
			rawNotification, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "notifications/tools/list_changed"})
			_, _ = os.Stdout.Write(append(rawNotification, '\n'))
			writeMCPHelperResponse(request.ID, mcpHelperResult(request.Method))
		default:
			writeMCPHelperResponse(request.ID, mcpHelperResult(request.Method))
		}
	}
	os.Exit(0)
}

func TestMCPLongLivedHelperProcess(t *testing.T) {
	if !hasArg("mcp-long-lived-helper") {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	counter := 0
	for scanner.Scan() {
		var request struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil || request.ID == "" {
			continue
		}
		switch request.Method {
		case "tools/list":
			writeMCPHelperResponse(request.ID, map[string]any{"tools": []any{map[string]any{
				"name":        "counter",
				"description": "Increment process-local counter",
				"inputSchema": map[string]any{"type": "object"},
			}}})
		case "tools/call":
			counter++
			writeMCPHelperResponse(request.ID, map[string]any{"content": []any{map[string]any{"type": "text", "text": strconv.Itoa(counter)}}})
		default:
			writeMCPHelperResponse(request.ID, mcpHelperResult(request.Method))
		}
	}
	os.Exit(0)
}

func TestMCPReconnectHelperProcess(t *testing.T) {
	if !hasArg("mcp-reconnect-helper") {
		return
	}
	marker := os.Getenv("AIVO_MCP_RECONNECT_FILE")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var request struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil || request.ID == "" {
			continue
		}
		switch request.Method {
		case "tools/list":
			writeMCPHelperResponse(request.ID, map[string]any{"tools": []any{map[string]any{
				"name":        "recover",
				"description": "Recover after reconnect",
				"inputSchema": map[string]any{"type": "object"},
			}}})
		case "tools/call":
			if marker != "" {
				if _, err := os.Stat(marker); os.IsNotExist(err) {
					_ = os.WriteFile(marker, []byte("failed"), 0o600)
					os.Exit(0)
				}
			}
			writeMCPHelperResponse(request.ID, map[string]any{"content": []any{map[string]any{"type": "text", "text": "recovered"}}})
		default:
			writeMCPHelperResponse(request.ID, mcpHelperResult(request.Method))
		}
	}
	os.Exit(0)
}
