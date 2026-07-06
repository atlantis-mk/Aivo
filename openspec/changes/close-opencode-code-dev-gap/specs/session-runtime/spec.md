## ADDED Requirements

### Requirement: Sessions expose execution control operations
Aivo SHALL expose app-layer operations to inspect, interrupt, resume, compact, and tail durable execution events for a session.

#### Scenario: Interrupt operation
- **WHEN** a client calls `InterruptSessionExecution` for an active session
- **THEN** Aivo requests cancellation of owned work and returns the updated execution state

#### Scenario: Resume operation
- **WHEN** a client calls `ResumeSessionExecution` for an interrupted or idle session with pending work
- **THEN** Aivo continues from durable session state and records the new execution state

#### Scenario: Event cursor operation
- **WHEN** a client calls `ListSessionEventsAfterCursor` with a prior cursor
- **THEN** Aivo returns durable events after that cursor and a next cursor suitable for later polling

### Requirement: Sessions support explicit compaction requests
Aivo SHALL compact model context through summaries while preserving original events.

#### Scenario: Compact operation
- **WHEN** a client calls `CompactSessionContext` for a session with enough history to summarize
- **THEN** Aivo creates or updates a summary, records compaction metadata, and returns the resulting compact context metadata
