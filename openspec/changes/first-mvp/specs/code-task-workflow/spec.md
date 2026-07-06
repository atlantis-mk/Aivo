## ADDED Requirements

### Requirement: Create code-to-delivery task
Aivo SHALL allow the user to create a task for a selected project from a natural-language task description.

#### Scenario: Valid task is created
- **WHEN** the user submits a non-empty task description with a selected project
- **THEN** Aivo creates a task record with a stable task ID, project ID, description, initial status, and creation timestamp

#### Scenario: Empty task is rejected
- **WHEN** the user submits an empty or whitespace-only task description
- **THEN** Aivo rejects the request and shows a validation error without creating a task record

### Requirement: Review task plan before execution
Aivo SHALL produce or accept a structured task plan and require the user to approve execution before running local side effects.

#### Scenario: Plan awaits approval
- **WHEN** a task plan is generated or attached to a task
- **THEN** Aivo shows the ordered plan steps and marks the task as waiting for plan approval

#### Scenario: User declines plan
- **WHEN** the user declines the task plan
- **THEN** Aivo does not execute planned side effects and marks the task as stopped or awaiting revision

### Requirement: Execute observable task steps
Aivo SHALL execute approved task steps through app-layer orchestration and record observable progress.

#### Scenario: Step starts running
- **WHEN** an approved task step starts
- **THEN** Aivo records a tool-run or step event with status, timestamp, and the associated task ID

#### Scenario: Step completes successfully
- **WHEN** a task step completes successfully
- **THEN** Aivo records the completed status and any produced artifact, log, or verification result references

#### Scenario: Step fails
- **WHEN** a task step fails
- **THEN** Aivo records the failed status, preserves contextual error information, and exposes a safe error summary to the UI

### Requirement: Show logs, artifacts, diffs, and verification results
Aivo SHALL make task outputs inspectable through the workbench.

#### Scenario: Task log is recorded
- **WHEN** a task emits progress or execution output
- **THEN** Aivo stores log entries with ordering information and renders them in the task timeline or log view

#### Scenario: Artifact is produced
- **WHEN** a task produces a generated file, delivery note, diff, or verification result
- **THEN** Aivo persists artifact metadata and shows the artifact in the task review surface

### Requirement: Cancel running task
Aivo SHALL allow the user to cancel a running task and propagate cancellation to owned work where possible.

#### Scenario: User cancels task
- **WHEN** the user cancels a running task
- **THEN** Aivo requests cancellation through the app layer, records the cancellation, and prevents new task steps from starting

#### Scenario: Cancellation completes
- **WHEN** owned running operations acknowledge cancellation
- **THEN** Aivo marks the task as canceled and records final task state

### Requirement: Resume interrupted task state
Aivo SHALL preserve enough task state to show interrupted tasks after the app restarts.

#### Scenario: App restarts with incomplete task
- **WHEN** the app restarts after a task was waiting, running, failed, or canceled
- **THEN** Aivo lists the task with its last persisted status, logs, artifacts, and available next actions

#### Scenario: Completed task remains reviewable
- **WHEN** a task has completed
- **THEN** the user can reopen the task and inspect its plan, logs, artifacts, verification results, and delivery notes
