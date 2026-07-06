## ADDED Requirements

### Requirement: Workbench shows project readiness for coding
Aivo SHALL show whether the selected project is ready for code development, including recent project metadata, Git status, non-Git state, and inaccessible-path errors.

#### Scenario: Git project readiness
- **WHEN** the user opens a Git-backed project
- **THEN** the workbench shows project name, path, branch when available, dirty state, and last opened time

#### Scenario: Non-Git project readiness
- **WHEN** the user opens a readable non-Git directory
- **THEN** the workbench allows coding sessions and marks Git metadata as unavailable without blocking task creation

#### Scenario: Inaccessible project
- **WHEN** a persisted recent project path is no longer readable
- **THEN** the workbench shows a safe error state and prevents new coding execution for that project until the path is fixed or removed

### Requirement: Workbench requires plan approval before side effects
Aivo SHALL show structured plans and require approval before file writes, shell commands, or other sensitive side effects run in a code-delivery task.

#### Scenario: Plan awaits approval
- **WHEN** the assistant produces a task plan with side-effecting steps
- **THEN** the workbench shows the ordered plan and blocks execution until the user approves it

#### Scenario: Plan declined
- **WHEN** the user declines the plan
- **THEN** Aivo records the decision and no planned side-effecting tool runs start

### Requirement: Workbench timeline is reviewable
Aivo SHALL render coding activity as an inspectable timeline covering tool calls, command output, permissions, file changes, retained output, diffs, diagnostics, and verification results.

#### Scenario: File change is inspectable
- **WHEN** a write, edit, or patch tool changes a file
- **THEN** the timeline shows path, additions, deletions, status, and expandable diff when available

#### Scenario: Command output is inspectable
- **WHEN** a shell, test, or diagnostics tool produces output that exceeds inline bounds
- **THEN** the timeline shows a bounded preview and an action to load retained output

#### Scenario: Permission request is visible
- **WHEN** a tool is waiting for approval
- **THEN** the timeline or permission surface shows target, proposed effect, risk summary, approve, deny, and remember controls when allowed

### Requirement: Workbench resume recap guides continuation
Aivo SHALL show a resume recap for prior coding sessions with enough state to continue work safely.

#### Scenario: Resume recap displayed
- **WHEN** the user reopens a project with a previous coding session
- **THEN** the workbench shows last command, changed files, latest checkpoint, open todos, known issues, and next suggested action when available
