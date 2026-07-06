## ADDED Requirements

### Requirement: Open local project
Aivo SHALL allow the user to select a local project directory and create or update a persisted project reference for it.

#### Scenario: User selects valid project directory
- **WHEN** the user selects an existing local directory as a project
- **THEN** Aivo persists a project record with a stable project ID, display name, absolute path reference, created timestamp, and last-opened timestamp

#### Scenario: User selects unavailable directory
- **WHEN** the selected directory does not exist or cannot be accessed
- **THEN** Aivo rejects the selection and shows a safe user-facing error without creating a project record

### Requirement: List recent projects
Aivo SHALL show persisted projects so the user can reopen prior work.

#### Scenario: Recent projects are displayed
- **WHEN** at least one project record exists
- **THEN** the workbench lists recent projects with display name, path reference, and last-opened time

#### Scenario: No projects exist
- **WHEN** no project records exist
- **THEN** the workbench shows an empty state with an action to open a local project

### Requirement: Inspect basic repository context
Aivo SHALL inspect selected project context through Go services and expose basic metadata useful for a code-to-delivery task.

#### Scenario: Git repository metadata is available
- **WHEN** the selected project is a Git repository
- **THEN** Aivo shows the current branch when available and whether the working tree has uncommitted changes

#### Scenario: Non-Git directory is selected
- **WHEN** the selected project is not a Git repository
- **THEN** Aivo still opens the project and marks Git metadata as unavailable instead of failing the project selection

### Requirement: Constrain project filesystem access
Aivo SHALL treat project paths as untrusted boundary input and route filesystem inspection through backend services.

#### Scenario: Path is validated before use
- **WHEN** the frontend submits a project path to the backend
- **THEN** the backend validates access to the path before persisting or inspecting it

#### Scenario: UI does not inspect filesystem directly
- **WHEN** frontend source is inspected
- **THEN** React components do not directly read project files or execute filesystem operations

### Requirement: Project context is available to tasks
Aivo SHALL associate tasks with a selected project so task execution can reference the intended local context.

#### Scenario: Task starts with selected project
- **WHEN** the user starts a task while a project is selected
- **THEN** the task record includes the selected project ID and uses that project as its default working context

#### Scenario: Task requires project context
- **WHEN** the user attempts to start a code-to-delivery task without a selected project
- **THEN** Aivo blocks task creation and prompts the user to select a local project first
