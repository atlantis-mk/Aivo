# Distinguish one-request tool inspection from conversation tool use

## Problem or goal

The initial auxiliary selector currently returns capability groups, so a question such as “当前有哪些工具可调用” can be mistaken for a task that needs those tools throughout the conversation. The Host needs the auxiliary model to distinguish capability inspection from capability use while keeping the selection identity uniform: inspection exposes the complete eligible catalog to the primary model once, while use persists only capabilities expanded from exact typed MCP/extension source IDs. Complete concrete expansion must not be rejected merely because the catalog exceeds a fixed Host-owned count.

## Expected behavior

For `REQ-SESSION-001`, `REQ-TOOL-001`, and `REQ-EXTENSION-001`, the pre-call auxiliary response classifies the user intent as `inspect` or `use` and can return only exact typed MCP/extension source IDs. `inspect` makes all currently eligible concrete tools visible only in the first primary Provider request, persists no automatic names, and initializes the conversation with an empty durable automatic set. `use` expands selected sources locally and persists their concrete names as the stable automatic set. Later `tool_resolve` remains use-only and persistent.

## Non-goals

This Work does not add an Agent-visible source inventory executor, change manual activation, bypass global visibility/readiness/mode filters or call-time permissions, install or enable sources, remove auxiliary group/input sanitization bounds or Provider-owned protocol/context limits, or make later source-catalog changes silently enter an existing conversation.

## Impact

Core auxiliary parsing, initial automatic-set initialization, Provider tool assembly, Tool Snapshot activation-source metadata, visible selection notes, renderer lifetime copy, focused tests, primary specifications, one superseding ADR, and traceability change. Electron/preload, external RPC/HTTP contracts, SQLite schema, dependencies, credentials, processes, extension protocols, platform packaging, and release behavior are unchanged.

## Implementation constraints

The Host owns classification validation, complete eligible expansion, auxiliary group/input bounds, lifetime, persistence, and snapshot assembly. `inspect` must be non-persistent and exactly one Provider request; `use` must be atomic and conversation-persistent. Invalid auxiliary output falls back to bounded local classification/search. Concrete expansion has no Host-owned count cap and must never silently truncate or convert a larger valid catalog into an empty result. Cancellation or Provider failure cannot leak request-scoped activation into another request or conversation.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `TASK-037-01` | REQ-TOOL-001 | Strict auxiliary `inspect`/`use` decision, typed source-ID validation, and uncapped concrete expansion | AT-TOOL-001, CT-SECURITY-001 | Completed |
| `TASK-037-02` | REQ-SESSION-001 | One-request complete inspection injection versus durable use selection | AT-SESSION-001, CT-RELIABILITY-001 | Completed |
| `TASK-037-03` | REQ-EXTENSION-001 | Complete eligible group expansion and persistent `tool_resolve` replacement | AT-EXTENSION-001 | Completed |
| `TASK-037-04` | all | Documentation and repository gates with recorded evidence | all | Completed |

## Acceptance and evidence

- A new-conversation inventory question exposes every eligible concrete tool in its first primary request without a Host-owned concrete-tool count cap, persists an initialized empty automatic set, and omits those tools from a later primary request.
- A new-conversation action request exposes and persists only selected groups; later turns retain those concrete tools without another auxiliary call.
- `tool_resolve` remains persistent, replaces rather than accumulates, and cannot request temporary inspection exposure.
- Invalid classification, unknown/duplicate groups, hidden or unavailable members, auxiliary failure, cancellation, and repetition fail closed or use the bounded local fallback without leaking tools across requests; a valid catalog larger than 64 concrete tools remains complete.
- Focused Go tests, `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, `pnpm build`, and `git diff --check` provide evidence. Wide/narrow visual acceptance remains owned by related `CHG-2026-036` because this Work changes only the lifetime copy inside that same unaccepted disclosure surface; this Work remains `Implementing` with that related acceptance.

Automated evidence on 2026-08-21: strict decision/parser, hidden-member expansion, complete inspection and visible-note parsing beyond the former 64-tool cap, request-only first-snapshot isolation, initialized-empty persistence, later-turn omission, persistent initial use, persistent `tool_resolve` replacement, standalone-tool retention, and exact MCP server-ID registration tests pass. The final `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, `pnpm build`, and `git diff --check` gates pass. Lint/build retain only existing Fast Refresh, large-barrel, and chunk-size warnings. One earlier full-Core run exposed the shell output ordering test's existing timing sensitivity; three isolated repetitions passed and the final full-Core gate passed.

## Security and data lifecycle

The auxiliary request contains only bounded user intent and sanitized typed MCP/extension IDs with source descriptions; blank MCP descriptions use the Host-assembled current tool-description fallback. `inspect` broadens every eligible schema only within one immutable Provider request and Tool Snapshot after Host eligibility checks. The visible selection note contains categorized resource IDs/counts and lifetime, never concrete MCP/extension member names, credentials, endpoints, configuration, auxiliary output, or hidden candidates.

## Compatibility and migration

No schema migration is required. Existing initialized conversations keep their current automatic set. An uninitialized conversation uses the new classification on its next primary request. Downgrade treats an inspect-initialized empty set as an ordinary stable empty automatic set.

## Bug root cause (type=bug only)

N/A.
