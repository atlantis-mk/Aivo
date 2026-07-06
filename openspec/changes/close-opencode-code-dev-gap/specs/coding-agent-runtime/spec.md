## ADDED Requirements

### Requirement: Coding sessions support recoverable execution
Aivo SHALL track execution state for coding sessions independently from the session record so active, interrupted, failed, idle, and compacting states can be resumed or reviewed after restart.

#### Scenario: Execution state survives restart
- **WHEN** the app starts and finds a coding session with a previously running turn or tool call
- **THEN** Aivo marks the execution state as interrupted, preserves recorded events and tool calls, and shows a resume action without replaying side effects automatically

#### Scenario: Explicit interrupt stops owned work
- **WHEN** the user interrupts an active coding session
- **THEN** Aivo cancels owned provider and tool work where possible, records the interruption, and prevents new tool calls from starting until the session is resumed

### Requirement: Coding sessions support queued and steering input
Aivo SHALL distinguish input that steers the current run from input queued for the next idle boundary.

#### Scenario: Steering input is promoted at safe boundary
- **WHEN** a user sends input with delivery `steer` during an active provider/tool continuation
- **THEN** Aivo records the input and promotes it at the next safe provider-turn boundary before continuing the same run

#### Scenario: Queued input waits for idle boundary
- **WHEN** a user sends input with delivery `queue` while a coding session is still continuing
- **THEN** Aivo records the input and does not promote it until the current continuation reaches an idle boundary

### Requirement: Provider turns are durably settled before continuation
Aivo SHALL persist advertised tool identities, started tool calls, completed tool settlements, and assistant continuation state before a later provider request depends on those results.

#### Scenario: Tool result persisted before next provider turn
- **WHEN** the model requests a tool call and the tool succeeds
- **THEN** Aivo persists the tool result and related event before issuing the provider request that uses the tool result

#### Scenario: Stale running tool call is not replayed
- **WHEN** a provider turn is prepared after restart and a prior local tool call is still marked running
- **THEN** Aivo marks that tool call as interrupted and excludes automatic replay from the resumed request

### Requirement: Session compaction preserves full event history
Aivo SHALL create compact model context from summaries, checkpoints, and recent visible events without deleting original session events.

#### Scenario: Manual compaction creates summary
- **WHEN** the user or service requests compaction for a long coding session
- **THEN** Aivo creates or updates a session summary, records a compaction event, and keeps all original events queryable

#### Scenario: Context builder uses compacted context
- **WHEN** a compacted session builds model context
- **THEN** Aivo prioritizes the latest summary, latest checkpoint, coding context, and recent visible events within the requested budget
