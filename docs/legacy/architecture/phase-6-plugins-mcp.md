# Phase 6 Plugins And MCP

Phase 6 adds product-grade local extension points for Aivo:

- local plugin manifests and subprocess JSONL tools
- MCP server configuration and stdio tool import
- plugin/MCP permission integration
- progressive tool loading with `tool_search`, `tool_describe`, and `tool_call`
- desktop management UI under Settings -> Plugins & MCP

## Plugin Manifest

A plugin root may contain either `.aivo-plugin/plugin.json` or `aivo.plugin.json`.
All relative paths must remain inside the plugin root.

```json
{
  "id": "example-tools",
  "name": "example-tools",
  "version": "1.0.0",
  "displayName": "Example Tools",
  "description": "Small local extension",
  "entrypoint": {
    "command": "./plugin-server",
    "args": [],
    "env": {}
  },
  "hooks": ["pre_tool_call", "post_tool_call"],
  "tools": [
    {
      "name": "example_echo",
      "description": "Echo text",
      "capability": "plugin.read",
      "riskLevel": "low",
      "inputSchema": {
        "type": "object",
        "properties": {
          "text": { "type": "string" }
        },
        "required": ["text"]
      }
    }
  ]
}
```

## Plugin JSONL Protocol

The host starts the entrypoint as a subprocess and sends line-delimited JSON:

```json
{"id":"...","method":"initialize","params":{"pluginId":"example-tools","manifest":{}}}
{"id":"...","method":"tool.call","params":{"name":"example_echo","arguments":{"text":"hi"}}}
{"id":"...","method":"hook.invoke","params":{"hook":"post_tool_call","payload":{}}}
```

The plugin responds:

```json
{"id":"...","result":{"tools":[],"hooks":[]}}
{"id":"...","result":{"ok":true,"content":"hi"}}
```

Errors are returned as:

```json
{"id":"...","error":{"code":"failed","message":"details"}}
```

## MCP Configuration

MCP servers can be created in the desktop UI or declared by plugin manifests.
Phase 6 v1 enables stdio transport. HTTP/SSE config is accepted as a forward-compatible shape, but probing returns a clear unsupported-transport diagnostic until those transports are enabled.

```json
{
  "id": "filesystem",
  "name": "filesystem",
  "transport": "stdio",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "."],
  "enabled": true
}
```

## Permissions

Plugin tools default to `category: plugin` and `capability: plugin.read`.
MCP tools default to `category: mcp` and `capability: mcp.read`.
Capabilities containing `.write` or `.patch` use the existing write approval path.
Scheduler worker mode denies admin and MCP tools.

## Progressive Loading

Built-in tools remain visible. Plugin and MCP tools are deferrable. When their schemas become large enough, Aivo advertises only:

- `tool_search`
- `tool_describe`
- `tool_call`

`tool_call` invokes the underlying tool through the same `ToolRuntime`, so permissions, hooks, result bounding, and audit logging are shared with direct calls.

## Stability

The registry records a registration identity for every advertised tool. If a plugin or MCP tool changes after a model turn is prepared, execution returns `stale_tool_registration` instead of invoking a different implementation under the same name.
