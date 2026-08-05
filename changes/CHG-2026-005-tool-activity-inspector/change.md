# Add a contextual tool activity inspector

## Problem or goal

Conversation tool activity is currently represented by repeated disclosure headings and interleaved execution-description text inside the message flow. Replace that sequence with a compact wrapping flow of per-call tool-name labels and let the user inspect the selected activity in a contextual right-side panel without restoring the removed persistent activity workspace.

## Expected behavior

`REQ-TOOL-002` requires each visible non-delegated tool call to use a shadcn Badge containing its actual tool name. Badges are packed from left to right with a compact gap and wrap onto following lines; assistant execution-description fragments are omitted from this activity region. Activating any Badge or the surrounding region opens one right-side inspector for the complete activity, with a width transition that reduces the chat canvas width. A newly received live tool call automatically opens the inspector, while loading existing history does not. Manual close suppresses subsequent automatic opening only for the current conversation; manual activation still works, and a different or newly created conversation resets suppression. The inspector flattens every tool group in that activity into a single timeline ordered by call time, and each timeline row opens an overlapping detail card inside the same inspector. The user can return to the timeline or close the inspector. Delegate/subagent cards retain their current behavior.

## Non-goals

No persistent auxiliary panel, resizable handle, top-bar panel trigger, bottom terminal panel, live terminal management, file-revert control, backend tool behavior, tool contract, permission flow, or data model change. This Work does not restore the multi-tab activity surface removed by `CHG-2026-004-chat-canvas-only-shell`.

## Impact

Only the Electron renderer conversation composition and responsive layout are affected. Electron main/preload, Go domain/application/persistence/transport, API/RPC/IPC, schemas, credentials, providers, plugins/MCP, LSP, processes, worktrees, packaging, dependencies, and platform scope are unchanged.

## Implementation constraints

Compose the UI from existing shadcn Badge, Button, Card, Item, ScrollArea, Separator, and Skeleton primitives without modifying `apps/desktop/src/components/ui`. Use the configured Hugeicons library and semantic theme tokens. Keep the inspector contextual and closed by default, preserve keyboard activation and visible focus, update selected tool data while a run progresses, avoid horizontal overflow at narrow widths, and fall back to an overlay-width presentation when the chat canvas cannot support a full pushed-aside layout.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `TOOL-INSPECTOR-DOC-001` | `REQ-TOOL-002` | Scope, Requirement, Work, and traceability agree | `AT-TOOL-002` | Completed |
| `TOOL-INSPECTOR-BADGE-001` | `REQ-TOOL-002` | Tool calls render in one accessible wrapping activity trigger without execution descriptions | `AT-TOOL-002` | Completed |
| `TOOL-INSPECTOR-PANEL-001` | `REQ-TOOL-002`, `NFR-UI-001` | Animated right inspector, timeline, and stacked detail view | `AT-TOOL-002` | Completed |
| `TOOL-INSPECTOR-QA-001` | `NFR-UI-001`, `NFR-UI-002` | Lint/build/docs plus wide and narrow interaction screenshots | `AT-TOOL-002` | Pending |

## Acceptance and evidence

- Each visible non-delegated tool call shows one Badge with its actual tool name and status affordance; badges flow left to right with compact spacing and wrap automatically inside one activity trigger, while the old disclosure list, counts, and interleaved execution-description text are not rendered in the conversation.
- Mouse and keyboard activation anywhere in the Badge region, including directly on a label, open the complete activity in the contextual inspector; opening and closing visibly animate and preserve the chat's scroll, composer, and interaction docks.
- A newly received live tool call automatically opens its activity. Existing history does not auto-open; manual close blocks later automatic opening for that conversation without blocking manual activation, and changing to a different or new conversation restores automatic opening.
- The inspector flattens every group in the activity and orders all tool calls by `timeCreated`, using the associated assistant invocation description as the row title only when present and the tool name as supporting text, with running, approval, success, or failure state shown through semantic shadcn variants.
- Selecting a timeline row opens its arguments and safe result summary in a card layered over the timeline; arguments and results each use an independent shadcn ScrollArea, while Back and Close restore predictable focus and state.
- The detail card uses its existing title style for the associated invocation description when present and its existing supporting-description style for the tool name; when no invocation description exists, the title line is omitted.
- Repeated selection, live tool updates, missing descriptions/results, long commands, failure, and pending approval remain readable. Cancellation, timeout, teardown, persistence, migration, rollback, provider, and security behavior are N/A because execution services are unchanged.
- Wide and narrow captures show no hidden persistent controls or horizontal viewport overflow. `pnpm docs:check`, `pnpm lint`, and `pnpm build` pass without new warnings.

Implementation evidence recorded on 2026-08-01: each non-delegated tool call renders an individual shadcn Badge containing its actual tool name inside one full-width activity Button, so the surrounding region and every label share a single activation target. Consecutive badges use a compact left-to-right wrapping flow, and assistant execution-description fragments are omitted from the conversation while remaining associated with their corresponding tool groups for the inspector. The contextual inspector composes shadcn Card, ScrollArea, ItemGroup, Badge, Button, and Separator primitives with the configured Hugeicons set; it pushes the chat canvas at wide widths, overlays at narrow widths, flattens the selected activity into one `timeCreated`-ordered status timeline with invocation descriptions, and opens an in-panel stacked detail card for arguments and safe visible results. Per user direction, final visual and interaction acceptance is user-owned, so `TOOL-INSPECTOR-QA-001` remains Pending and this Work stays `Implementing`.

## Security and data lifecycle

The renderer displays only tool-call fields already available to the existing conversation timeline. It adds no logging, persistence, clipboard, network, credential, prompt, retained-output, crash, backup, or privileged-service behavior. Result detail is limited to existing safe summaries/content already rendered in the conversation.

## Compatibility and migration

No schema, data, settings, API/RPC/IPC, dependency, upgrade, or downgrade migration. Rollback removes the inspector composition and restores the existing inline disclosure presentation.

## Bug root cause (type=bug only)

N/A.
