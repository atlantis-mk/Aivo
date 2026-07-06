# Tool Runtime Architecture

## Phase 1 Shape

The first implementation is intentionally read-only:

- `read_file`
- `list_files`
- `search_files`

These tools are registered through `NewReadOnlyToolRegistry`, exposed to models only for coding sessions with a workspace root, and executed through `ToolRuntime`. They use relative workspace paths, reject traversal, reject symlink escapes, skip generated directories, reject sensitive files, reject binary content, and truncate large outputs.

## Core Types

`domain.ToolSpec` defines the model-visible name, description, JSON schema, namespace, capability, risk level, category, toolsets, and workspace/network/secret flags.

`domain.Tool` binds a spec to executable code:

```go
type Tool interface {
    Spec() ToolSpec
    Execute(ctx context.Context, args json.RawMessage, execCtx ToolExecutionContext) ToolResult
}
```

`ToolExecutionContext` carries workspace root, session id, turn id, agent mode, and output policy. Future fields should include approval ids, sandbox profile, and audit ids.

`ToolResult` is structured: call id, name, ok flag, content, display error, `ToolError`, truncation flag, and original size.

`OutputPolicy` starts with max chars/lines and truncation marker. It will grow to stdout/stderr separation, diff limits, redaction, raw artifact references, and summarization hooks.

## Execution Pipeline

```text
receive tool_call
  → normalize name and args
  → lookup tool
  → validate JSON args
  → resolve workspace
  → authorize permission
  → maybe ask user
  → execute with timeout
  → sanitize output
  → truncate output
  → write audit log
  → return tool_result
```

Phase 1 implements normalization, lookup, JSON validation, local workspace safety, timeout, panic recovery, output truncation, structured log lines, SQLite tool call records, and internal tool result events. Permission and approval are reserved for Phase 2.

## Registry

The registry must support built-in tools now and later support self-registering tools, plugin tools, MCP tools, duplicate name detection, provider-specific schema conversion, toolsets, availability checks, health checks, dynamic enable/disable, and progressive loading.

Tool names are globally unique inside the active registry. Future plugin and MCP names should be namespaced, for example `mcp.github.search_issues` or `plugin.docs.render_docx`.

## Built-in Tool Matrix

| Tool | Capability | Risk | Default Permission | Plan | Build | Workspace | Network | Secrets |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| read_file | filesystem.read | low | allow | yes | yes | yes | no | may encounter, blocked by detector |
| list_files | filesystem.list | low | allow | yes | yes | yes | no | no content read |
| search_files | filesystem.search | low | allow | yes | yes | yes | no | skips sensitive paths |
| write_file | filesystem.write | high | ask | no | yes | yes | no | possible |
| apply_patch | filesystem.patch | high | ask | no | yes | yes | no | possible |
| git_status | git.read | low | allow | yes | yes | yes | no | no |
| git_diff | git.read | low | allow | yes | yes | yes | no | possible |
| git_log | git.read | low | allow | yes | yes | yes | no | no |
| bash | shell.exec | critical | ask | no | guarded | yes | optional | possible |
| run_tests | shell.test | medium | ask/allow known | no | yes | yes | optional | possible |
| lsp_diagnostics | lsp.read | low | allow | yes | yes | yes | no | no |
| update_plan | plan.write | low | allow | yes | yes | no | no | no |
| ask_user | user.ask | low | allow | yes | yes | no | no | no |
| web_search | web.search | medium | ask/allow | yes | yes | no | yes | external transfer |
| web_fetch | web.fetch | medium | ask/allow | yes | yes | no | yes | external transfer |
| scheduler_create | scheduler.write | high | ask | no | guarded | no | optional | possible |
| delegate_task | agent.delegate | high | ask | yes | yes | optional | optional | possible |

## Error Handling

Tool errors are structured with stable codes. Phase 1 codes include `runtime_unconfigured`, `invalid_tool_call`, `tool_not_found`, `invalid_arguments`, `timeout`, `tool_panic`, and `tool_error`. Later codes should include `schema_validation_failed`, `permission_denied`, `user_rejected`, `sandbox_failed`, `binary_file_rejected`, `output_too_large`, and `provider_unsupported_tool_calls`.

## Tests

Phase 1 requires registry tests, safe path tests, symlink escape tests, read/list/search tests, truncation tests, no-tool agent loop tests, tool loop tests, streamed tool call tests, and max step tests. Phase 2 adds permission and approval tests. Phase 3 adds sandbox, command, env, and cleanup tests.
