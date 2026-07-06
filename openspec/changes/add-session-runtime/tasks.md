## 1. Domain Model Setup

- [x] 1.1 Add generic session, turn, event, tool call, summary, checkpoint, coding context, resume recap, and context builder structs in `domain`.
- [x] 1.2 Add enum-like constants and validation helpers for session type, session status, source, turn status, event type, event role, event visibility, and tool call status.
- [x] 1.3 Add request/response structs for create, list/filter, update, append event, start/complete/fail/cancel turn, create summary, create checkpoint, fork, resume, and build context operations.

## 2. Persistence Migration

- [x] 2.1 Extend the existing SQLite migration path with additive columns for generic `sessions` metadata while preserving current assistant session compatibility.
- [x] 2.2 Add SQLite tables and indexes for `turns`, `session_events`, `tool_calls`, `session_summaries`, `session_checkpoints`, and `coding_contexts`.
- [x] 2.3 Add repository methods for session CRUD, filters, latest by project, events, turns, tool calls, summaries, checkpoints, coding context, and fork support.
- [x] 2.4 Add persistence tests covering empty database migration, legacy `sessions/messages` compatibility, indexes/filters, and JSON payload round trips.

## 3. Session Service Core

- [x] 3.1 Implement UI-independent SessionService use cases for create, list, get, update, archive, soft delete, continue last, and latest by project.
- [x] 3.2 Implement append event with type/visibility validation, safe redaction defaults, session timestamp updates, and normal-visibility list behavior.
- [x] 3.3 Implement turn lifecycle methods that preserve already-written events when turns fail or are cancelled.
- [x] 3.4 Implement tool call persistence helpers with result summaries and redacted/internal visibility handling.
- [x] 3.5 Implement summary creation, latest summary lookup, deterministic fallback summary/title generation, and compact entry point.
- [x] 3.6 Implement checkpoint creation/list/latest behavior and checkpoint event append.
- [x] 3.7 Implement fork session with lineage, copied coding context, copied or referenced latest summary, and independent future events.

## 4. Context Builder and Coding Continuity

- [x] 4.1 Implement coding context creation/update using project path normalization and best-effort git branch, commit sha, repo URL, changed file, language stack, package manager, cwd, and permission capture.
- [x] 4.2 Update existing assistant create/submit/list flows to write through Session Core while preserving current Electron response shapes.
- [x] 4.3 Implement resume recap by id, continue last, and continue by project path with safe user-facing fields only.
- [x] 4.4 Implement buildSessionContext with ordered sections for system prompt snapshot, user preference placeholder, goal, latest summary, latest checkpoint, coding context, recent normal events, bounded tool/command summaries, optional search results, and current user input.
- [x] 4.5 Add deterministic budget handling and truncation metadata for context builder output.

## 5. Electron and Frontend Integration

- [x] 5.1 Add Aivo bridge methods for session create/list/get/update/archive/delete, latest session, latest by project, events, turns, fork, summaries, checkpoints, coding context, resume recap, and context building.
- [x] 5.2 Add typed frontend service clients under `frontend/src/services` instead of calling generated Aivo bridge handlers directly from arbitrary components.
- [x] 5.3 Add or update workbench/session UI for session list, session detail, resume button, safe event timeline, latest summary, checkpoint list, coding context, and resume recap.
- [x] 5.4 Ensure UI defaults exclude hidden, internal, and redacted events and clearly distinguishes archive from delete.

## 6. Testing and Verification

- [x] 6.1 Add app service tests for session lifecycle, event append, visibility filtering, turn failure preservation, summary, checkpoint, fork lineage, and context builder output.
- [x] 6.2 Add coding continuity tests for create coding session, submit message fallback, latest by project path, resume recap, and checkpoint without git metadata.
- [x] 6.3 Add frontend tests for session list/detail/resume states if the project test setup is available. No component test setup is currently configured; covered by frontend typecheck/lint.
- [x] 6.4 Run `go test ./...` and fix failures.
- [x] 6.5 Run `cd frontend && pnpm typecheck` or the configured equivalent and fix failures.
- [x] 6.6 Run `cd frontend && pnpm lint` or report clearly if lint is not configured.
