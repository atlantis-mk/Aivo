## Why

Aivo needs durable continuity before the code-to-delivery workflow can support serious long-running work. The current assistant session shape is close to chat history, but the product direction requires sessions to be recoverable, searchable, auditable, checkpointed, and extensible beyond coding.

## What Changes

- Introduce a generic Session Runtime where sessions are first-class resources composed of turns, events, tool calls, summaries, checkpoints, and domain context.
- Persist complete session event history in SQLite while keeping coding-specific state in a separate coding context model.
- Add service methods for create, list, read, update, archive, soft delete, continue, fork, event append, turn lifecycle, summary, checkpoint, coding context, and context building.
- Expand Aivo-bridge-facing assistant APIs and frontend service/UI surfaces so users can list sessions, inspect session details, resume work, view summaries/checkpoints/coding context, and continue by recent session or project path.
- Add deterministic fallback title/summary generation and a provider-independent interface for later LLM-generated titles, summaries, and compaction.
- Add visibility and redaction behavior so hidden, internal, and redacted events are excluded from normal UI and model context by default.
- Preserve compatibility with current assistant session and message usage during migration.

## Capabilities

### New Capabilities

- `session-runtime`: Generic durable Session Core covering session records, turn lifecycle, event stream, tool calls, summaries, checkpoints, archive/delete, fork, search filters, and context building.
- `coding-session-continuity`: Coding-specific session behavior covering project-based resume, coding context, git metadata, checkpoint recap, and frontend resume surfaces.

### Modified Capabilities

- None.

## Impact

- Backend domain models gain generic session, turn, event, tool call, summary, checkpoint, context builder, and coding context types.
- App service layer gains UI-independent SessionService use cases and deterministic fallback summarization/title behavior.
- SQLite persistence gains migrations/tables for sessions expansion, turns, session events, tool calls, session summaries, session checkpoints, and coding contexts, plus indexes for filtering and basic search.
- Electron app bindings expose session runtime operations through typed request/response shapes.
- Frontend service and workbench UI gain session list/detail/resume/checkpoint/summary/context views without scattering raw Aivo bridge calls through components.
- Tests cover service behavior, persistence behavior, context builder visibility filtering, fork lineage, checkpoint creation, and compatibility with existing assistant flows.
