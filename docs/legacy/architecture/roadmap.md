# Agent Runtime Roadmap

## Phase 1: Architecture Skeleton And Read-only Tools

Implemented scope:

- provider-neutral messages, chat requests/responses, tool specs, tool calls, tool results, streaming event type
- agent loop with max steps, ordinary chat compatibility, tool call execution, tool result feedback, streaming final text, and max step failure
- registry and read-only tool runtime
- `read_file`, `list_files`, `search_files`
- safe relative paths, symlink escape prevention, ignored directories, sensitive file detection, binary rejection, output truncation
- structured tool call logging and SQLite tool call records
- tests for registry, tools, safe path, symlink escape, streamed tool calls, no-tool chat, and max steps

## Phase 2: Permission Engine And Patch

Add allow/ask/deny, pending approval UI, saved approvals, path policies, `apply_patch`, `git_status`, `git_diff`, and deterministic rejection for denied writes.

## Phase 3: Bash, Test Runner, Sandbox, And Terminal

Add command detector, production bash tool, run_tests, env allowlist, command timeout, process cleanup, local sandbox interface, Docker backend interface, and an OpenCode-style conversation bottom PTY terminal panel. See [Phase 3 implementation plan](phase-3-bash-test-sandbox.md).

## Phase 4: Agent Modes And Toolsets

Add assistant, planner, builder, reviewer, researcher, and scheduler_worker. Each mode defines prompt responsibilities, toolsets, default permissions, file write access, command access, network access, and background-task access.

Toolsets: safe, coding, shell, personal, web, git, lsp, mcp, and admin.

## Phase 5: Todo And Session Search

Add todo tools and session search.

## Phase 6: Plugins And MCP

Add plugin manifests, subprocess JSON protocol, plugin hooks, plugin sandboxing, MCP server discovery, tool schema import, MCP namespacing, MCP permissions, result normalization, and progressive tool loading through tool_search/tool_describe/tool_call.

## Phase 7: Scheduler And Personal Assistant Layer

Add one-time tasks, recurring tasks, condition watches, worker execution context, notification channels, and scheduled-job permission scopes.

## Phase 8: Remote And Cloud Backends

Add desktop/cloud coordination, SSH backend, cloud workers, remote workspace sync, long-running task recovery, and multi-workspace execution.
