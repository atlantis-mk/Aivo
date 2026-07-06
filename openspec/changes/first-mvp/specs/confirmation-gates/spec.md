## ADDED Requirements

### Requirement: Identify sensitive actions
Aivo SHALL classify sensitive actions before execution, including file writes, deletion, overwrites, shell or process execution, credential use, external data transfer, uploads, publishing, and purchases.

#### Scenario: File overwrite is classified
- **WHEN** a task proposes overwriting an existing project file
- **THEN** Aivo classifies the operation as sensitive and requires confirmation before execution

#### Scenario: Shell command is classified
- **WHEN** a task proposes running a local shell or process command
- **THEN** Aivo classifies the operation as sensitive and requires confirmation before execution

### Requirement: Create pending confirmation record
Aivo SHALL create a pending confirmation record before executing a sensitive action.

#### Scenario: Sensitive action needs approval
- **WHEN** a sensitive action is requested by task orchestration
- **THEN** Aivo records a pending confirmation with action type, target summary, proposed effect, risk summary, task ID, and creation timestamp

#### Scenario: Sensitive action is visible to user
- **WHEN** a pending confirmation exists
- **THEN** the workbench shows the confirmation details and approve or deny actions to the user

### Requirement: Block execution until approval
Aivo SHALL block sensitive action execution until the corresponding confirmation is approved.

#### Scenario: Approval is missing
- **WHEN** a pending confirmation has not been approved
- **THEN** the associated sensitive action is not executed

#### Scenario: Confirmation is denied
- **WHEN** the user denies a pending confirmation
- **THEN** Aivo records the denial and prevents the associated sensitive action from executing

#### Scenario: Confirmation is approved
- **WHEN** the user approves a pending confirmation
- **THEN** Aivo records the approval and allows only the approved action scope to proceed

### Requirement: Invalidate stale or changed confirmations
Aivo SHALL prevent stale approvals from authorizing changed sensitive actions.

#### Scenario: Action details change
- **WHEN** the target, command, proposed effect, or risk summary changes after a confirmation is created
- **THEN** Aivo invalidates the prior confirmation and requires a new approval before execution

#### Scenario: Task is canceled
- **WHEN** a task is canceled while confirmations are pending
- **THEN** Aivo marks pending confirmations for that task as no longer actionable

### Requirement: Preserve confirmation audit trail locally
Aivo SHALL preserve local metadata for confirmation approvals, denials, and invalidations without storing secrets in confirmation records.

#### Scenario: Confirmation history is reviewed
- **WHEN** a user opens a task with sensitive actions
- **THEN** Aivo shows the confirmation decisions associated with the task

#### Scenario: Secret value would appear in confirmation
- **WHEN** a confirmation involves credential or secret use
- **THEN** Aivo records a redacted target or credential reference and does not store the secret value in the confirmation details
