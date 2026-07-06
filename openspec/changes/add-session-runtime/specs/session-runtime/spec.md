## ADDED Requirements

### Requirement: Sessions are durable first-class resources

The system SHALL persist sessions as generic long-running assistant work threads with stable identifiers, type, status, source, goal, summary, lineage, model snapshot, system prompt snapshot, token/cost metadata, timestamps, and extensible metadata.

#### Scenario: Create generic coding session

- **WHEN** a client creates a session with type `coding`, source `web`, title, goal, project path, and model reference
- **THEN** the system persists a session with status `active`, the supplied metadata, created and updated timestamps, and a stable id

#### Scenario: Archive and soft delete session

- **WHEN** a client archives a session and later soft deletes it
- **THEN** the system records status `archived` with `archivedAt` for archive and status `deleted` for delete without physically removing events, turns, summaries, checkpoints, or coding context

#### Scenario: List and filter sessions

- **WHEN** a client lists sessions with filters for type, status, project path, source, search text, and limit
- **THEN** the system returns matching non-deleted sessions ordered by most recent update unless the filter explicitly includes deleted sessions

### Requirement: Turns track each user-triggered run

The system SHALL create a turn for each user request and persist the turn status independently from events so partial event history remains available when a run fails or is cancelled.

#### Scenario: Complete turn

- **WHEN** a user request starts and completes successfully
- **THEN** the system records a turn with status transition from `running` to `completed`, links the user message event id when available, and stores completion time

#### Scenario: Failed turn retains events

- **WHEN** a turn records user message, tool call, and shell command events before failing
- **THEN** the system marks the turn `failed` with an error and the previously appended events remain queryable by session and turn

#### Scenario: Cancel turn

- **WHEN** a running turn is cancelled
- **THEN** the system marks it `cancelled`, records completion time, and appends an event that makes the cancellation visible in the session event stream

### Requirement: Event stream captures atomic assistant activity

The system SHALL persist session events for user messages, assistant messages, tool calls, tool results, file reads, file writes, file patches, shell commands, shell output, git diffs, plan updates, summaries, checkpoints, errors, and system notes.

#### Scenario: Append normal message events

- **WHEN** a user sends a message and the assistant replies
- **THEN** the system appends `user_message` and `assistant_message` events with role, content, visibility `normal`, timestamps, and optional token counts

#### Scenario: Append structured tool and command events

- **WHEN** a tool call runs a shell command and writes a patch
- **THEN** the system appends structured `tool_call`, `tool_result`, `shell_command`, `shell_output`, and `file_patch` events with payload fields preserving machine-readable details

#### Scenario: Hide sensitive or internal events by default

- **WHEN** events have visibility `hidden`, `internal`, or `redacted`
- **THEN** normal event listing, resume recap, and context building exclude those events unless the caller explicitly requests inclusion and has permission

### Requirement: Tool calls are persisted independently

The system SHALL persist tool call records linked to session, turn, and optional event with tool name, redacted arguments, status, result summary, error, and timestamps.

#### Scenario: Successful tool call

- **WHEN** a tool call is started and then succeeds with a large result
- **THEN** the system records status `success`, stores a structured result when safe, stores a bounded `resultSummary`, and links the relevant session event

#### Scenario: Sensitive tool result

- **WHEN** a tool call result contains secrets, credentials, cookies, or private payloads
- **THEN** the system redacts or omits sensitive fields and marks related event visibility as `redacted` or `internal`

### Requirement: Summaries preserve history while supporting compaction

The system SHALL support session summaries over event ranges without deleting or replacing original events.

#### Scenario: Create latest summary

- **WHEN** a client creates a summary with event range, facts, decisions, open tasks, changed files, and next suggested action
- **THEN** the system persists the summary, appends a summary event, updates the session summary, and returns it as the latest summary

#### Scenario: Compact session

- **WHEN** compaction is requested for a session with many events
- **THEN** the system creates or updates a summary and future context building prioritizes summary plus recent events while preserving full original event history

### Requirement: Checkpoints preserve resumable development state

The system SHALL support session checkpoints containing changed files, git metadata, diff summary, conversation summary, open todos, known issues, and next suggested action.

#### Scenario: Create checkpoint

- **WHEN** a client creates a checkpoint after a development milestone
- **THEN** the system persists the checkpoint, appends a checkpoint event, updates the session timestamp, and returns the checkpoint from latest-checkpoint queries

#### Scenario: List checkpoints

- **WHEN** a client lists checkpoints for a session
- **THEN** the system returns checkpoints ordered by newest first with changed files, open todos, known issues, and next suggested action

### Requirement: Forks preserve lineage with independent future history

The system SHALL allow a session to be forked into a new active session that records parent and fork lineage while owning independent future turns and events.

#### Scenario: Fork session

- **WHEN** a client forks an active session with a new title or goal
- **THEN** the system creates a new active session with `parentSessionId` and `forkedFromSessionId` referencing the source session, copies or references the latest summary, and does not require copying the complete source event stream

#### Scenario: Append event after fork

- **WHEN** the forked session receives a new user message
- **THEN** the event is appended only to the forked session and the source session event list remains unchanged

### Requirement: Context builder produces bounded safe resume context

The system SHALL build model-ready session context from snapshots, goal, latest summary, latest checkpoint, domain context, recent visible events, tool summaries, command summaries, relevant search results when available, and the current user input.

#### Scenario: Build default context

- **WHEN** a client builds context for a session with summary, coding context, checkpoint, visible events, hidden events, and redacted tool results
- **THEN** the returned context includes the system prompt snapshot, goal, latest summary, latest checkpoint, coding context, recent normal events, bounded tool summaries, and current user input while excluding hidden, internal, and redacted events

#### Scenario: Respect context budget

- **WHEN** a client supplies a maximum token budget or character budget
- **THEN** the system truncates lower-priority recent events or summaries according to deterministic priority rules and returns metadata describing omitted sections

### Requirement: Search and retrieval are available through service boundaries

The system SHALL expose session lookup by id, latest active session, latest session by project path, and basic search/filtering without coupling behavior to a specific UI component.

#### Scenario: Continue last session

- **WHEN** a client requests the last session
- **THEN** the service returns the most recently updated active non-deleted session and enough recap information to resume work

#### Scenario: Find latest session by project

- **WHEN** a client requests the latest session for a normalized project path
- **THEN** the service returns the most recently updated active non-deleted session associated with that path

#### Scenario: Basic text search

- **WHEN** a client searches sessions by text
- **THEN** the service searches title, summary, project path, and visible event content using the currently available SQLite-backed search strategy
