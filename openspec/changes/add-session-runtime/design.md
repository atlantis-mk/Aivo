## Context

Aivo is a local-first Electron desktop app with Go backend services, SQLite persistence, and a React/Vite frontend. The current code already has lightweight assistant sessions and messages, but the requested personal assistant direction needs a runtime that behaves more like a durable work thread than a chat transcript.

The design combines three reference ideas without copying their implementations: Codex-style thread/turn/event history, OpenCode-style sessions as API/service resources, and Hermes-Agent-style summaries with safe context restoration.

## Goals / Non-Goals

**Goals:**

- Add a generic Session Core that supports coding now and future personal assistant domains later.
- Persist complete event history, turns, tool calls, summaries, checkpoints, and coding context in SQLite.
- Keep session operations in Go domain/app/infra layers so Electron, future CLI, IDE, or API clients can reuse the same service.
- Build safe bounded resume context without replaying full history.
- Preserve existing assistant session/message compatibility while expanding the schema.
- Add frontend session continuity surfaces through typed service boundaries.

**Non-Goals:**

- Full autonomous agent execution, real LLM summarization, vector search, export/import, hard rollback, or multi-user sync.
- Storing secrets or sensitive tool results in normal visible content.
- Making coding fields part of the generic sessions table.
- Implementing enterprise audit/compliance or collaboration permissions.

## Decisions

### 1. Expand Session Core instead of replacing current assistant sessions

The existing `sessions` and `messages` tables will be migrated forward by adding generic columns to `sessions` and introducing new runtime tables. Existing `messages` can remain as a compatibility view/source during transition, while new assistant flows write to `session_events`.

Alternatives considered:

- Replace `sessions` and `messages` immediately. This is cleaner but risks breaking current Aivo bridge handlers and frontend code.
- Keep only messages and add metadata. This fails the event stream, turn, checkpoint, and fork requirements.

### 2. Use append-only events as the audit source of truth

`session_events` stores atomic user, assistant, tool, file, shell, diff, plan, summary, checkpoint, error, and system note records. Payloads are JSON strings in SQLite and mapped to typed Go structs at the service boundary. UI defaults to `visibility = normal`.

Alternatives considered:

- Store each event type in a separate table only. That improves type constraints but makes timeline queries and future client support cumbersome.
- Store raw transcript blobs. That loses auditability and structured recovery.

### 3. Keep domain context separate from sessions

Coding state is stored in `coding_contexts` and linked by `session_id`. Future domains can add `email_contexts`, `calendar_contexts`, or `research_contexts` without bloating generic sessions.

Alternatives considered:

- Put project path, git branch, changed files, and command state on `sessions`. This is simple now but makes generic assistant sessions carry coding-only fields forever.

### 4. Context Builder is an app-layer use case

Context building assembles system prompt snapshot, user preference placeholder, goal, latest summary, latest checkpoint, coding context, recent normal events, tool/command summaries, basic search results when available, and current input. It returns structured sections plus budget/truncation metadata. Token counting starts as deterministic character or rough token estimation and can later be replaced behind the same interface.

Alternatives considered:

- Let frontend concatenate history. This duplicates policy and risks exposing hidden events.
- Always replay full history. This is expensive, unsafe, and degrades long-session behavior.

### 5. Deterministic summarizer/title fallback first

The first implementation should include `SummaryGenerator` and `TitleGenerator` interfaces but provide deterministic fallbacks based on first user message, latest events, changed files, tool summaries, and open todos. LLM-backed generation can be added after provider invocation is stable.

Alternatives considered:

- Require a model for summaries. This blocks local-first development and tests.
- Skip summaries until LLM execution exists. This weakens resume and compact behavior.

### 6. SQLite search starts basic and leaves room for FTS

Initial search filters sessions by title, summary, project path, status, source, type, and recent visible event content with indexed columns where practical. FTS/CJK-aware search can be added through a later migration or adapter without changing service method names.

Alternatives considered:

- Add vector search now. This adds dependencies and provider questions too early.
- No search. This conflicts with the session runtime requirements.

## Risks / Trade-offs

- [Risk] Migrating the existing `sessions` table may be fragile across developer databases. Mitigation: implement additive migrations with `hasColumn` checks and tests against empty and legacy schemas.
- [Risk] JSON payloads are flexible but weakly constrained. Mitigation: keep typed domain request/response structs, validate event type/visibility/status values in app services, and test serialization.
- [Risk] Session events can grow large. Mitigation: store large outputs in payload summaries by default, bound context builder output, and add future pruning/export policies separately from compaction.
- [Risk] Sensitive data may enter event payloads. Mitigation: centralize append-event redaction/visibility rules and default UI/context queries to normal visibility only.
- [Risk] Frontend session detail could become a chat-only view. Mitigation: include events, summary, checkpoints, and coding context as distinct review surfaces.

## Migration Plan

1. Add domain models for sessions, turns, events, tool calls, summaries, checkpoints, coding contexts, resume recap, and context builder output.
2. Extend app store interfaces and implement `SessionService` methods independent of Electron UI.
3. Add additive SQLite migration logic for new columns/tables and indexes.
4. Adapt existing assistant creation/message methods to dual-write or write through Session Core while preserving current response shapes.
5. Add Aivo bridge handlers and typed frontend services for session runtime operations.
6. Build frontend session list/detail/resume/checkpoint/context surfaces.
7. Add tests for migration, service lifecycle, visibility filtering, summaries, checkpoints, fork lineage, and project resume.

Rollback strategy: because migrations are additive, older code can ignore new tables and columns. If runtime writes fail, assistant submit flow should return contextual errors rather than silently falling back to non-durable state.

## Open Questions

- Needs Confirmation: Whether `deleteSession` should ever physically delete rows from SQLite in a future privacy/export feature, or remain soft delete only for V1.
- Needs Confirmation: The exact frontend route shape for session history once the broader workbench navigation stabilizes.
- Needs Confirmation: Whether the first CJK search implementation should use SQLite FTS5 tokenizer configuration or remain basic `LIKE` search until there is real usage data.
