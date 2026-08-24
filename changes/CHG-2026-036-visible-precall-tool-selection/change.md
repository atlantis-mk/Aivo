# Show pre-call automatic tool selection in the conversation

## Problem or goal

The Host searches eligible capability groups and initializes a conversation's automatic tool set before its first primary model request, but that action is invisible in the conversation. Users can inspect later model-authored tool calls while being unable to see which concrete tools the Host injected up front.

## Expected behavior

For `REQ-SESSION-001`, `REQ-TOOL-001`, and `REQ-TOOL-002`, the first actual auxiliary tool-selection call creates one normal turn-owned `host_tool_selection` system note in `running` state and pushes it to the active desktop immediately. Successful initialization updates that same note to `completed` with bounded typed resource summaries and request/conversation lifetime; failed initialization updates it to `failed`. MCP and extension summaries contain the exact source ID once plus the number of concrete tools Host injected, never one item per member. Standalone tools and Skills use the only other two resource kinds. The desktop renders the note as a default-collapsed accessible disclosure with a visible state summary, grouped details using the same four resource categories as management, and explicit empty/failure states.

## Non-goals

This Work does not expose the eligible candidate catalog, group descriptions, schemas, auxiliary prompt/response/reasoning, fallback provenance, Skill or extension context, credentials, endpoints, or source configuration. It does not change selection, replacement, authorization, execution, Tool Snapshot, or inspector auto-open behavior.

## Impact

Go application orchestration creates and updates one existing-format session event around the first auxiliary selection and emits its safe normal-system-note DTO over a local created/updated event stream. The renderer merges that event into the active turn and adds a responsive default-collapsed disclosure to the existing conversation timeline. Electron main/preload, RPC methods, SQLite schemas/migrations, providers, extension processes, dependencies, packaging, and platform scope are unchanged; the local Core event-stream contract gains the additive system-event notification.

## Implementation constraints

Core owns event construction and emits `running` only immediately before an actual auxiliary Provider call. It updates the same event ID after Host validation and persistence, derives completed resource summaries from the validated concrete selection, and emits at most one record for initial selection. Event-stream DTOs contain the same persisted safe normal note. The renderer upserts by event ID, accepts only the four resource kinds, rejects duplicate or malformed identities, fails safely to the ordinary system-note presentation, and keeps long IDs and grouped badges wrapping at narrow widths.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `TASK-036-01` | `REQ-SESSION-001`, `REQ-TOOL-001` | One bounded turn-owned event records the successfully committed initial automatic set | `AT-SESSION-001`, `AT-TOOL-001`, `CT-SECURITY-001` | Completed |
| `TASK-036-02` | `REQ-TOOL-002`, `NFR-UI-001` | Default-collapsed responsive disclosure keeps its state summary visible and groups expanded source-level injection details into the four resource categories | `AT-TOOL-002` | Completed |
| `TASK-036-03` | `REQ-TOOL-002` | Malformed/historical note payloads retain safe fallback rendering | `AT-TOOL-002` | Completed |
| `TASK-036-04` | `REQ-SESSION-001`, `REQ-TOOL-002` | One live note transitions from auxiliary loading to completed or failed result | `AT-SESSION-001`, `AT-TOOL-002` | Completed |

## Acceptance and evidence

- A first task that selects tools shows one “前置工具搜索” disclosure before subsequent activity; it starts collapsed with its current state summary visible and, when expanded, displays each MCP/extension source once with its exact ID and injected member count.
- When an auxiliary selector is available, the same disclosure becomes visible with an animated loading state before its Provider call returns, then transitions in place to the categorized resource/lifetime result; no second row is added.
- An initialized empty automatic set shows “未注入额外工具”; a failed initialization creates no success record; subsequent model steps and later turns do not duplicate the initial record.
- Candidate groups, descriptions, schemas, resolver output, prompt/context, and credentials are absent from the payload and UI.
- Historical ordinary system notes and malformed typed payloads continue using their existing safe text presentation.
- Focus, keyboard toggle, wrapping, long names, and narrow widths remain readable. Cancellation, timeout, teardown, migration, rollback, and target-OS packaging are N/A because no long-lived resource or schema changes.
- Required verification: focused Go and renderer tests, `pnpm docs:check`, `pnpm test:core`, `pnpm lint`, and `pnpm build`; wide/narrow visual acceptance remains user-owned.

Implementation evidence on 2026-08-21: Core emits one turn-owned normal `system_note`, derives deterministic typed source summaries from the exact internal selection, and stops before the primary Provider request if the visibility event cannot be stored. Repeated turns retain one event, while a successful empty initialization records an empty resource array. The renderer validates bounded four-kind resource identities/counts, places the disclosure immediately after the user message and before later activity, originally opened it by default, groups wrapping badges by resource kind, and safely renders malformed typed notes through the ordinary note fallback.

Verification on 2026-08-21: the focused Core selection/event tests and `apps/desktop/tests/conversation-system-note-model.test.ts` passed; `pnpm docs:check`, `pnpm scripts:test` (51 desktop tests plus script/extension suites), `pnpm test:core`, `pnpm lint`, and `pnpm build` passed. Lint retained only the repository's existing Fast Refresh warnings and the build retained existing large-barrel/chunk advisories. Wide/narrow visual acceptance remains user-owned, so this Work stays `Implementing` and unsealed.

Live-state refinement evidence on 2026-08-21: immediately before the first actual auxiliary Provider selection call, Core persists and publishes one bounded `running` system note. It updates that same event ID to `completed` only after Host validation and automatic-set persistence, including typed source summaries and request/conversation lifetime, or to a generic `failed` state. The desktop consumes additive local event notifications, upserts the note by event ID, and announces the live state accessibly. No-auxiliary local initialization creates only the completed result.

Live-state verification on 2026-08-21: focused Core coverage asserted the created/running then updated/completed sequence, stable event identity, final persistence, and later-turn non-duplication; the renderer tests covered running, completed, failed, malformed fallback, historical missing-status compatibility, and in-place merge. `pnpm scripts:test` passed with 53 desktop tests plus the script/extension suites; `pnpm docs:check`, `pnpm test:core`, `pnpm lint`, and `pnpm build` passed. Lint retained only existing Fast Refresh warnings and the build retained existing large-barrel/chunk advisories. Wide/narrow visual acceptance remains user-owned, so this Work stays `Implementing` and unsealed.

Default-collapse refinement on 2026-08-24: the desktop now initializes the disclosure as collapsed while keeping the running/completed/failed summary and toggle available in its header. `REQ-TOOL-002`, traceability, and this Work were synchronized to the new default. `pnpm docs:check`, `pnpm lint`, and `pnpm build` passed; lint retained only existing Fast Refresh warnings and the build retained existing large-barrel/chunk advisories. Wide/narrow visual acceptance remains user-owned.

## Security and data lifecycle

The event persists only a bounded running/completed/failed state and, for completion, typed source/tool/skill IDs, safe display names, member counts, and request/conversation lifetime. Concrete MCP/extension member names remain internal to session state and Tool Snapshots. The record contains no secrets, prompts, user content, descriptions, schemas, configuration, paths, raw provider output, error detail, or authorization state.

## Compatibility and migration

No schema, data migration, RPC method, setting, dependency, or irreversible operation changes. The local event stream adds created/updated session-event notifications; clients that ignore them still load the final persisted note normally. Payloads without `status` remain compatible completed history, older clients render bounded content as an ordinary system note, and older history without the payload remains unchanged. Rollback leaves harmless ordinary system-note rows in persisted history.

## Bug root cause (type=bug only)

N/A.
