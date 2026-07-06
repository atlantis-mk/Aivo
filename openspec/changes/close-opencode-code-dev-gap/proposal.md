## Why

Aivo already has a usable desktop coding loop, but it is not yet reliable or complete enough to replace opencode for day-to-day code development. This change closes the desktop code-development gap by hardening sessions, tools, permissions, code intelligence, plugin/MCP integration, and workbench review surfaces while preserving Aivo's local-first Electron + Go architecture.

## What Changes

- Add a replacement acceptance baseline focused on desktop code development only: repository understanding, planning, editing, command/test execution, diff review, session recovery, MCP/plugin participation, and permission handling.
- Strengthen coding session execution with queued and steering input, explicit interrupt/resume, context compaction, provider-turn safety, and recovery of interrupted tool calls.
- Expand code intelligence from regex-style symbol search to language-server-backed diagnostics, symbols, definitions, and references with safe fallback behavior.
- Harden write, patch, shell, diagnostics, and test tools with stronger preflight checks, stale-file detection, partial failure reporting, retained output, command policy, and turn-level revert/restore.
- Productionize permission, plugin, and MCP behavior so all built-in, plugin, and MCP tools share catalog identity, authorization, stale-registration checks, diagnostics, and safe context insertion behavior.
- Complete the workbench code-delivery surfaces for project state, plan approval, tool timeline, permission prompts, diff/output review, verification results, and resume recap.
- Exclude CLI, TUI, SDK, GitHub Action, enterprise collaboration, and non-code productivity workflows from this replacement target.

## Capabilities

### New Capabilities

- `coding-agent-runtime`: Durable and recoverable code-agent execution for desktop coding sessions.
- `code-intelligence`: Language-server-backed code diagnostics, symbols, definitions, and references with fallback behavior.
- `tool-permissions`: Unified authorization and safety behavior for file, shell, network, plugin, MCP, and external-directory tool actions.
- `plugin-mcp-runtime`: Production-ready plugin and MCP tool registration, execution, diagnostics, and explicit context insertion.
- `workbench-code-delivery`: Desktop workbench experience for project selection, planning, execution review, diffs, logs, permissions, and resume.
- `replacement-acceptance`: Task-matrix acceptance criteria that define when Aivo can replace opencode for desktop code development.

### Modified Capabilities

- `session-runtime`: Extend durable sessions with interrupt/resume execution, queued and steering input, compaction, event cursors, and interrupted tool-call recovery.
- `coding-session-continuity`: Extend coding continuity with reliable resume recap, last command, changed files, open todos, checkpoints, and recovery-oriented next actions.
- `local-project-context`: Require project metadata to support replacement-level recent project, Git, non-Git, and inaccessible-path states.
- `code-task-workflow`: Require plan approval, observable tool execution, verification, cancellation, and restart recovery to be sufficient for real coding tasks.
- `confirmation-gates`: Extend sensitive-action confirmation coverage to plugin/MCP tools, network, stdin/env, external directories, and saved approvals.

## Impact

- Backend domain and app services gain execution state, event cursor, compaction, interrupt/resume, code intelligence, and expanded permission metadata request/response shapes.
- Tool runtime keeps the current registry pattern but requires stable source/sourceID/registrationID for built-in, plugin, and MCP tools and rejects stale advertised tools.
- Persistence gains additive SQLite tables or columns for session execution state, queued inputs, event cursors or sequence metadata, LSP server status, and acceptance run results as needed.
- Frontend services and workbench UI gain typed clients and review surfaces for execution state, code intelligence results, permission decisions, MCP/plugin context insertion, and replacement acceptance runs.
- Tests expand around runtime recovery, tool safety, permissions, MCP/plugin behavior, LSP fallback, and real task-matrix scenarios.
