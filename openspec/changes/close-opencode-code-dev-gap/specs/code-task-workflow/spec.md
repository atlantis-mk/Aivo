## ADDED Requirements

### Requirement: Code tasks are observable from plan through verification
Aivo SHALL make every code-delivery task inspectable from natural-language request through plan, approved execution, tool activity, verification, and final review.

#### Scenario: Approved task runs observable steps
- **WHEN** the user approves a generated plan
- **THEN** Aivo records each started and completed tool or task step with status, timestamps, safe summary, and result references

#### Scenario: Verification result appears in review
- **WHEN** a task runs lint, build, diagnostics, or tests
- **THEN** Aivo records the verification command, status, bounded output, retained output reference when needed, and final result in the task review surface

### Requirement: Code tasks recover after interruption
Aivo SHALL show waiting, running, failed, interrupted, canceled, and completed tasks after app restart with available next actions.

#### Scenario: Interrupted task after restart
- **WHEN** the app restarts after a task was running
- **THEN** Aivo shows the task as interrupted or recoverable with recorded plan, events, changed files, and resume or review actions
