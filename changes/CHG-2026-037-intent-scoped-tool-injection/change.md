# Move resource inspection to runtime resource_resolve

## Problem or goal

The initial auxiliary selector runs before the primary Agent has inspected the task context, so a question such as “当前有哪些工具可调用” can be mistaken for a task that needs those tools throughout the conversation. Resource selection is more coherent when the primary Agent starts with the stable core surface and calls `resource_resolve` only after it knows it needs to inspect or activate optional capabilities. The Host still owns hidden catalog lookup, exact typed resource validation, grouped expansion, lifetime, and count behavior; the model only chooses whether it is asking to inspect resources or use them.

## Expected behavior

For `REQ-SESSION-001`, `REQ-TOOL-001`, and `REQ-EXTENSION-001`, Core no longer runs automatic resource selection before the first primary Provider request. The primary Agent always receives the required core tools plus `resource_resolve`; explicit composer/tool-picker selections are still applied directly before submission. `resource_resolve` requires `mode:"inspect"` or `mode:"use"`; omitted or invalid modes fail validation. `inspect` returns bounded categorized summaries of every matching eligible grouped-or-individual tool, Skill, or extension-context resource without exposing schemas, activating tools, or persisting automatic names. `use` invokes the Host auxiliary selector over current eligible candidates, expands selected resources locally without a Host-owned concrete-tool count cap, persists selected concrete tool names as the stable automatic set, replaces the filtered visible Skill catalog, and adds selected extension context for the next model step.

## Non-goals

This Work does not add a general Agent-visible source catalog executor, change manual activation, bypass global visibility/readiness/mode filters or call-time permissions, install or enable sources, remove auxiliary group/input sanitization bounds or Provider-owned protocol/context limits, expose grouped concrete members in summaries, or make later source-catalog changes silently enter an existing conversation.

## Impact

Core auxiliary parsing, runtime `resource_resolve` input/outputs, Provider tool assembly, Tool Snapshot activation-source metadata, visible selection/result summaries, renderer lifetime copy, focused tests, primary specifications, one superseding ADR, and traceability change. Electron/preload, external RPC/HTTP contracts, SQLite schema, dependencies, credentials, processes, extension protocols, platform packaging, and release behavior are unchanged.

## Implementation constraints

The Host owns mode validation, complete eligible expansion, auxiliary group/input bounds, lifetime, persistence, and snapshot assembly. `inspect` must be non-persistent and summary-only; `use` must be atomic and conversation-persistent. Invalid auxiliary output falls back to bounded local search. Concrete expansion has no Host-owned count cap and must never silently truncate or convert a larger valid use catalog into an empty result. Cancellation or Provider failure cannot leak request-scoped inspection into another request or conversation.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `TASK-037-01` | REQ-TOOL-001 | Strict `resource_resolve` `inspect`/`use` mode validation, typed resource-ID validation, and uncapped use expansion | AT-TOOL-001, CT-SECURITY-001 | Completed |
| `TASK-037-02` | REQ-SESSION-001 | No first-request optional-resource selection; runtime inspect summaries versus durable use selection | AT-SESSION-001, CT-RELIABILITY-001 | Completed |
| `TASK-037-03` | REQ-EXTENSION-001 | Complete eligible group expansion and persistent `resource_resolve` use replacement | AT-EXTENSION-001 | Completed |
| `TASK-037-04` | all | Documentation and repository gates with recorded evidence | all | Completed |

## Acceptance and evidence

- A new-conversation inventory question first reaches the primary Agent with only the required core surface plus `resource_resolve`; optional tools are not pre-injected.
- `resource_resolve` `mode:"inspect"` returns bounded typed resource summaries for matching eligible tools, Skills, and extension context without persisting automatic names or exposing schemas.
- `resource_resolve` `mode:"use"` remains persistent, replaces rather than accumulates, and expands valid grouped resources without a Host-owned concrete-tool count cap.
- Invalid mode, unknown/duplicate groups, hidden or unavailable members, auxiliary failure, cancellation, and repetition fail closed or use the bounded local fallback without leaking tools across requests; a valid use catalog larger than 64 concrete tools remains complete.
- Focused Go tests, `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, `pnpm build`, and `git diff --check` provide evidence. Wide/narrow visual acceptance remains owned by related `CHG-2026-036` because this Work changes only the lifetime copy inside that same unaccepted disclosure surface; this Work remains `Implementing` with that related acceptance.

Automated evidence on 2026-08-21 covered the earlier automatic-resource design. This revision replaces that behavior with no initial optional-resource injection, explicit runtime inspect summaries, persistent `resource_resolve` use replacement, hidden-member exclusion, standalone-tool retention, and exact MCP server-ID registration coverage.

## Security and data lifecycle

The auxiliary request contains only bounded user intent and sanitized typed MCP/extension/tool/Skill/context IDs with source descriptions; blank MCP descriptions use the Host-assembled current tool-description fallback. `inspect` returns only categorized resource IDs/names/counts and cannot expose executable schemas or activate tools. `use` broadens selected schemas only within immutable Provider requests and Tool Snapshots after Host eligibility checks. Visible summaries never include concrete grouped member names, credentials, endpoints, configuration, auxiliary output, or hidden candidates.

## Compatibility and migration

No schema migration is required. Existing initialized conversations keep their current automatic set. An uninitialized conversation starts with no automatic set until explicit user activation or a successful `resource_resolve` `use` call initializes it. Downgrade treats the absent or empty automatic set as an ordinary stable empty automatic set.

## Bug root cause (type=bug only)

N/A.
