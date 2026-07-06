## ADDED Requirements

### Requirement: Coding resume recap includes recovery fields
Aivo SHALL include last command, changed files, latest checkpoint, open todos, known issues, and next suggested action in coding resume recap when available.

#### Scenario: Recap includes command and checkpoint
- **WHEN** a coding session has a recorded last command and latest checkpoint
- **THEN** the resume recap includes both fields with safe user-facing values

#### Scenario: Recap handles missing data
- **WHEN** a coding session has no checkpoint or command history
- **THEN** the resume recap still returns session, project, and recent visible events without failing

### Requirement: Coding continuity reports interrupted work
Aivo SHALL show interrupted turns and tool calls as resumable state instead of hiding them.

#### Scenario: Interrupted tool call appears in recap
- **WHEN** a coding session contains a tool call marked interrupted during recovery
- **THEN** the resume recap or timeline exposes that interrupted tool call with a safe status and next action
