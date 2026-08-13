# Remove Agent-visible MCP source enumeration tool

## Problem or goal

The `aivo_tools_list_mcp` extension tool lets the primary Agent enumerate installed MCP source summaries even though the default Agent contract is to expose only concrete tools needed by the current task. MCP source inventory belongs to Host management and auxiliary candidate selection, not the primary Agent tool surface.

## Expected behavior

The `aivo.tools` built-in contributes only the bounded conversational MCP registration proposal tool. Enabled MCP tools remain discoverable to the Host auxiliary selector through persisted sanitized metadata and may enter a conversation only through its initial automatic selection, later `tool_resolve` replacement, or explicit manual selection. The primary Agent has no MCP source-enumeration executor and cannot use automatic selection merely to list installed MCP sources.

## Non-goals

No change to MCP persistence, probing, source settings, cached metadata, auxiliary selection, concrete MCP adapters, `tool_resolve`, manual activation, registration confirmation, credentials, permissions, Provider contracts, or renderer layout.

## Impact

Remove one Manifest contribution, one execution branch, and its catalog/search expectations. Preserve `aivo_tools_register_mcp` and all Host-owned MCP metadata and lifecycle services. Existing installed sources remain eligible for exact task-driven tool selection without depending on an Agent-visible list tool.

## Implementation constraints

The auxiliary selector must continue to receive sanitized concrete MCP tool summaries independently of `aivo_tools_list_mcp`. Provider declarations and Tool Snapshots must continue to exclude unselected MCP tools. No source name, tool schema, credential, or configuration is newly exposed to the primary Agent.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `MCP-LIST-DOC-001` | `REQ-EXTENSION-001`, `REQ-EXTENSION-002` | Primary requirements and traceability remove conversational source listing while retaining registration and automatic concrete-tool selection | `AT-EXTENSION-001`, `AT-EXTENSION-002` | Completed |
| `MCP-LIST-CORE-001` | `REQ-EXTENSION-002` | Manifest and built-in runtime no longer register or execute `aivo_tools_list_mcp` | `AT-EXTENSION-002` | Completed |
| `MCP-LIST-SELECTION-001` | `REQ-EXTENSION-001`, `REQ-SESSION-001` | MCP tools remain auxiliary candidates and only selected tools reach Provider declarations | `AT-EXTENSION-001`, `AT-SESSION-001` | Completed |
| `MCP-LIST-QA-001` | all | Focused and repository gates pass | all | Completed |

## Acceptance and evidence

- Global and workspace catalogs omit `aivo_tools_list_mcp` and retain `aivo_tools_register_mcp`.
- Asking for a concrete MCP capability can still select that exact MCP tool without the deleted source-list tool.
- Unselected MCP tools remain absent from the primary Provider request and Tool Snapshot.
- Conversational MCP registration proposal and native confirmation remain unchanged.
- `pnpm test:core`, `pnpm lint`, `pnpm build`, `pnpm docs:check`, and `git diff --check` pass.

Verification evidence recorded on 2026-08-10: the `aivo.tools` Manifest was advanced to version `1.1.0` with only `aivo_tools_register_mcp`; global and workspace catalog tests prove the removed name is absent; focused MCP selection coverage proves a cached concrete MCP tool remains an automatic candidate and that only the selected MCP tool reaches Provider assembly without the deleted executor. `pnpm test:core`, the focused desktop search test, `pnpm lint`, `pnpm build`, `pnpm docs:check`, and `git diff --check` passed. Lint and build retained only the repository's existing Fast Refresh, large-barrel, and chunk-size warnings. No interactive acceptance is required because no renderer control or layout changed.

## Security and data lifecycle

The change removes a model-visible inventory surface and adds no persisted state. Host-only cached MCP metadata remains bounded and sanitized for selection. Credential ownership, process/network authorization, cancellation, and source teardown are unchanged.

## Compatibility and migration

No schema migration is required. Existing sessions that persisted `aivo_tools_list_mcp` as a manual or automatic name retain inert metadata, but current registry assembly omits the absent registration. Downgrade may restore the tool contribution.

## Bug root cause (type=bug only)

N/A.
