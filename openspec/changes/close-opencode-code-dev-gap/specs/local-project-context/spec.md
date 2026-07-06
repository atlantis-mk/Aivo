## ADDED Requirements

### Requirement: Recent projects support replacement-level coding readiness
Aivo SHALL persist and display recent project records with enough metadata to decide whether a coding session can start.

#### Scenario: Recent project is ready
- **WHEN** a recent project path is readable
- **THEN** Aivo shows display name, absolute path, last opened time, Git availability, branch when available, and dirty state

#### Scenario: Recent project is unavailable
- **WHEN** a recent project path is missing or inaccessible
- **THEN** Aivo preserves the record, marks it unavailable, and blocks new coding execution for that project until resolved

### Requirement: Project Git metadata is best-effort
Aivo SHALL allow non-Git projects while making Git metadata availability explicit.

#### Scenario: Non-Git project opens
- **WHEN** a user opens a readable directory that is not a Git repository
- **THEN** Aivo marks Git metadata unavailable and allows file-backed coding sessions
