# Show resource-resolution disclosures in the conversation

## Problem or goal

The Host now resolves optional resources when the primary Agent explicitly calls `resource_resolve`, and users need a bounded way to inspect what that call found or activated. The former automatic resource-search system note is removed rather than retained as a compatibility surface.

## Expected behavior

For `REQ-SESSION-001`, `REQ-TOOL-001`, and `REQ-TOOL-002`, `resource_resolve` returns bounded typed resource summaries and an inspect/conversation lifetime through its normal tool result. MCP and extension summaries contain the exact source ID once plus the number of matching or activated concrete tools, never one item per grouped member. The desktop presents those summaries through the ordinary tool-activity/result surface and no longer parses a dedicated system-note payload.

## Non-goals

This Work does not expose the eligible candidate catalog, group descriptions, schemas, auxiliary prompt/response/reasoning, fallback provenance, Skill or extension context, credentials, endpoints, or source configuration. It does not change selection, replacement, authorization, execution, Tool Snapshot, or inspector auto-open behavior.

## Impact

Go application orchestration returns resource summaries from `resource_resolve` without creating an auxiliary-selection system event. The renderer relies on the existing tool-call timeline and inspector to present the result. Electron main/preload, RPC methods, SQLite schemas/migrations, providers, extension processes, dependencies, packaging, and platform scope are unchanged.

## Implementation constraints

Core owns `resource_resolve` result construction and derives summaries from Host-validated resource matches or concrete use selections. It emits no initial-selection system note before the first primary Provider request. The renderer accepts only the four resource kinds in `resource_resolve` result summaries, rejects duplicate or malformed identities, and presents tool-call output through the ordinary safe result surfaces.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `TASK-036-01` | `REQ-SESSION-001`, `REQ-TOOL-001` | `resource_resolve` exposes only bounded inspect/use resource summaries and no first-request resource note is created | `AT-SESSION-001`, `AT-TOOL-001`, `CT-SECURITY-001` | Completed |
| `TASK-036-02` | `REQ-TOOL-002`, `NFR-UI-001` | Tool-activity presentation shows resource-resolution output without a dedicated resource disclosure | `AT-TOOL-002` | Completed |
| `TASK-036-03` | `REQ-TOOL-002` | Former automatic-resource system-note parser and renderer are removed | `AT-TOOL-002` | Completed |
| `TASK-036-04` | `REQ-SESSION-001`, `REQ-TOOL-002` | Cancellation no longer needs special handling for automatic resource-search notes | `AT-SESSION-001`, `AT-TOOL-002` | Completed |

## Acceptance and evidence

- A `resource_resolve` call shows a resource-resolution result through the tool activity; no dedicated system-note disclosure is created or parsed.
- Inspect results identify non-persistent inventory; use results identify conversation-persistent activation.
- An empty result shows no activated resources; subsequent model steps and later turns do not create an initial resource-search record.
- Candidate groups, descriptions, schemas, resolver output, prompt/context, and credentials are absent from the payload and UI.
- Ordinary system notes continue using their existing safe text presentation.
- Focus, wrapping, long names, and narrow widths remain readable. Cancellation has no special resource-search cleanup path. Timeout, teardown, migration, rollback, and target-OS packaging are N/A because no long-lived resource or schema changes.
- Required verification: focused Go and renderer tests, `pnpm docs:check`, `pnpm test:core`, `pnpm lint`, and `pnpm build`; wide/narrow visual acceptance remains user-owned.

Implementation evidence on 2026-08-21 covered the earlier automatic-note design. This revision removes that event surface and keeps resource summaries inside `resource_resolve` tool results.

Verification on 2026-08-21 covered the earlier automatic-note design. The removed parser and tests no longer apply.

Live-state refinement evidence on 2026-08-21 covered the earlier automatic-note design and is superseded by the explicit `resource_resolve` result path.

Live-state verification on 2026-08-21 covered the earlier automatic-note design and is superseded by the explicit `resource_resolve` result path.

Default-collapse refinement on 2026-08-24 covered the earlier automatic-note design and no longer applies after removal.

## Security and data lifecycle

Resource-resolution output contains only bounded status and, for completion, typed source/tool/skill IDs, safe display names, member counts, and inspect/conversation lifetime. Concrete MCP/extension member names remain internal to session state and Tool Snapshots. The visible result contains no secrets, prompts, user content, descriptions, schemas, configuration, paths, raw provider output, error detail, or authorization state.

## Compatibility and migration

No schema, data migration, RPC method, setting, dependency, or irreversible operation changes. The former dedicated system-note payload is not supported. Older clients render any remaining persisted rows as ordinary system notes through the generic path.

## Bug root cause (type=bug only)

N/A.
