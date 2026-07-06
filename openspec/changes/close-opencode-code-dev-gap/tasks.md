## 1. Runtime Recovery and Context

- [x] 1.1 Add session execution state, queued input, steering input, interrupt, resume, compaction, and event cursor domain types.
- [x] 1.2 Add additive SQLite persistence and app service methods for execution state, pending inputs, event cursor listing, interrupted tool-call recovery, and compaction results.
- [x] 1.3 Update session message submission so `delivery=steer` is promoted at the next safe provider-turn boundary and `delivery=queue` waits until the current continuation is idle.
- [x] 1.4 Add explicit `InterruptSessionExecution`, `ResumeSessionExecution`, `CompactSessionContext`, and `ListSessionEventsAfterCursor` bridge methods and typed frontend services.
- [x] 1.5 Mark running tool calls as interrupted on startup or resume preparation instead of silently replaying side effects.

## 2. Code Intelligence

- [x] 2.1 Add language server status, diagnostic, symbol, definition, and reference domain types plus service interfaces.
- [x] 2.2 Implement a bounded LSP manager for Go and TypeScript/JavaScript workspaces with lifecycle, timeout, and unavailable-state handling.
- [x] 2.3 Add model-facing tools `lsp_diagnostics`, `lsp_definition`, and `lsp_references`; keep `lsp_symbol_search` with LSP-first and scan-fallback behavior.
- [x] 2.4 Show code-intelligence unavailable and fallback states in tool results without failing unrelated task execution.

## 3. Tool and Permission Hardening

- [x] 3.1 Strengthen write, edit, and patch tools with preflight summaries, stale hash checks, external path rejection, partial failure reports, and structured file change results.
- [x] 3.2 Strengthen shell, test, and diagnostics tools with unified command policy, cwd persistence, retained output, timeouts, background process status, and failure summaries.
- [x] 3.3 Extend permission metadata and saved approval matching for file writes, patches, shell/test, network, stdin, env keys, external roots, plugin tools, and MCP tools.
- [x] 3.4 Ensure turn-level diff, revert, and restore operations cover successful write/edit/patch results and report unsupported changes clearly.
- [x] 3.5 Add app-layer tests for permission allow/ask/deny, remembered approvals, stale approvals, external roots, network, stdin, env keys, and secret redaction.

## 4. Plugin and MCP Productionization

- [x] 4.1 Ensure built-in, plugin, and MCP tools all enter one catalog with source, sourceID, registrationID, riskLevel, and toolsets populated.
- [x] 4.2 Reject stale plugin/MCP tool calls when the advertised registration no longer matches the effective registry.
- [x] 4.3 Add MCP and plugin diagnostics to tool detail/settings surfaces, including startup, probe, auth, execution, timeout, and schema errors.
- [x] 4.4 Add explicit UI and service actions to insert MCP prompts/resources into a session; do not auto-inject them.
- [x] 4.5 Add fixture plugin and MCP tests for registration, execution, failure normalization, OAuth or auth errors where applicable, and recovery after restart.

## 5. Workbench Code Delivery

- [x] 5.1 Complete recent project, Git metadata, non-Git, inaccessible path, and selected-project blocked states.
- [x] 5.2 Complete task composer, structured plan review, approve, decline, cancel, resume, and completed review states.
- [x] 5.3 Ensure timeline renders tool calls, command output, permission prompts, file changes, retained output, diffs, diagnostics, verification results, and failure summaries.
- [x] 5.4 Show resume recap with last command, changed files, latest checkpoint, open todos, known issues, and next suggested action.
- [x] 5.5 Keep frontend bridge usage behind typed services and preserve shadcn/TanStack Router boundaries.

## 6. Replacement Acceptance

- [x] 6.1 Add a documented acceptance matrix covering small fix, multi-file feature, debug-to-build flow, interrupted resume, MCP/plugin participation, LSP lookup, and safety denial/approval.
- [x] 6.2 Add automated Go tests for runtime recovery, compaction, tool continuation, tool safety, permissions, MCP/plugin behavior, and LSP fallback.
- [x] 6.3 Run `pnpm test:core`, `pnpm lint`, and `pnpm build`; document any warnings or unrun checks.
- [ ] 6.4 Record manual acceptance results with project, prompt, provider, permissions selected, commands run, files changed, and outcome.
- [x] 6.5 Mark replacement readiness only when every acceptance scenario has a passing result and known limitations are documented.
