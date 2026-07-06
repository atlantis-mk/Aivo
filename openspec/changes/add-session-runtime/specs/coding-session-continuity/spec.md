## ADDED Requirements

### Requirement: Coding sessions create project context

The system SHALL create or update coding context for coding sessions without placing coding-only fields in the generic session record.

#### Scenario: Create coding context from project path

- **WHEN** a coding session is created with a project path
- **THEN** the system normalizes and stores the project path, attempts to record git branch, commit sha, repo URL, changed files, terminal cwd, language stack, package manager, and permissions in a coding context linked to the session

#### Scenario: Missing git metadata

- **WHEN** the project path is not a git repository or git metadata cannot be read
- **THEN** the system still creates the coding context and leaves git-specific fields empty without failing session creation

### Requirement: Coding session submit flow records turn and events

The system SHALL make the coding assistant submit flow append a user message event, create a turn, record assistant output, and update turn status.

#### Scenario: Submit coding message with fallback response

- **WHEN** a user submits a message to a coding session and no model execution is available
- **THEN** the system appends a `user_message` event, creates a running turn, appends a deterministic assistant message event, marks the turn completed, and updates the session title from the first message when no title exists

#### Scenario: Submit coding message failure

- **WHEN** message submission fails after the user message event is persisted
- **THEN** the system marks the turn failed, appends an error event when possible, and preserves the user message event for audit and resume

### Requirement: Coding resume recap is user-safe

The system SHALL return a coding resume recap that is understandable to users and omits system messages, internal reasoning, and sensitive tool results.

#### Scenario: Resume session by id

- **WHEN** a client resumes a coding session by id
- **THEN** the recap includes title, goal, latest summary, project path, branch, changed files, open todos, last command, next suggested action, updated time, latest checkpoint, and recent normal events

#### Scenario: Continue latest project session

- **WHEN** a client requests continuation for a project path
- **THEN** the service returns the latest active coding session for that path with the same safe recap fields

#### Scenario: Hidden data is excluded from recap

- **WHEN** a session contains hidden, internal, or redacted events
- **THEN** the recap excludes those event contents and does not reveal system prompt snapshots or sensitive tool payloads

### Requirement: Coding checkpoints capture repository progress

The system SHALL create coding checkpoints that capture repository and conversation state for later resume, fork, or rollback preparation.

#### Scenario: Manual checkpoint

- **WHEN** a user requests a checkpoint for a coding session
- **THEN** the system records changed files, branch, commit sha, diff summary, conversation summary, open todos, known issues, next suggested action, and appends a checkpoint event

#### Scenario: Checkpoint without diff

- **WHEN** git diff cannot be read or the project is not a git repository
- **THEN** the system still creates the checkpoint with available context and records the missing diff as a known issue or metadata note

### Requirement: Coding fork copies continuation context

The system SHALL allow coding sessions to fork while preserving enough coding context to continue independently.

#### Scenario: Fork coding session

- **WHEN** a coding session is forked
- **THEN** the new session receives copied coding context, latest summary reference or content, lineage identifiers, and an active status, while future events and checkpoints belong only to the fork

### Requirement: Frontend exposes session continuity surfaces

The system SHALL provide frontend service and UI surfaces for session list, detail, resume, checkpoint list, coding context, recent events, and summary display through typed Electron client boundaries.

#### Scenario: Show session list

- **WHEN** the user opens the code assistant workbench or session history
- **THEN** the UI shows recent non-deleted sessions with title, type, project path, status, source, updated time, and resume action

#### Scenario: Show session detail

- **WHEN** the user opens a session detail view
- **THEN** the UI shows safe normal events, latest summary, checkpoints, coding context, and a resume action without displaying hidden, internal, or redacted events by default

#### Scenario: Resume from UI

- **WHEN** the user clicks resume for a session
- **THEN** the frontend calls the typed session service, receives the safe recap and context metadata, and opens the workbench on that session
