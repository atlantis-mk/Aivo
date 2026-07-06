## ADDED Requirements

### Requirement: Show first-run welcome
Aivo SHALL show a welcome screen before initialization when the app has no completed setup state.

#### Scenario: App starts without setup
- **WHEN** Aivo starts and no completed setup state exists
- **THEN** the first visible screen introduces Aivo as a local-first code-to-delivery workbench and provides a primary action to start initialization

#### Scenario: Setup is already complete
- **WHEN** Aivo starts and setup has already been completed
- **THEN** Aivo skips the welcome screen and opens the workbench or last active project state

### Requirement: Configure provider during initialization
Aivo SHALL require the user to configure or select at least one provider before code-to-delivery task execution is available.

#### Scenario: No provider is configured
- **WHEN** the user reaches initialization and no provider is configured
- **THEN** Aivo shows direct provider choices for OpenAI, Claude Code, Gemini, and Codex-compatible, plus a Custom API option for advanced endpoints

#### Scenario: User selects provider
- **WHEN** the user selects OpenAI, Claude Code, Gemini, Codex-compatible, or Custom API
- **THEN** Aivo shows the fields and validation method required for that provider option without exposing unrelated provider fields

### Requirement: Persist non-secret provider metadata
Aivo SHALL persist provider configuration metadata without storing raw API keys, tokens, or secrets in SQLite, logs, screenshots, or generated artifacts.

#### Scenario: Direct provider is saved
- **WHEN** the user saves an OpenAI, Claude Code, Gemini, or Codex-compatible provider configuration
- **THEN** Aivo persists provider type, display name, selected model or profile, status, last validation time, and a credential reference label without persisting the raw secret value

#### Scenario: Custom API provider is saved
- **WHEN** the user saves a Custom API provider configuration
- **THEN** Aivo persists provider type, selected protocol, display name, base URL, selected model or profile, status, last validation time, and a credential reference label without persisting the raw secret value

### Requirement: Validate provider readiness
Aivo SHALL validate provider readiness before marking initialization complete.

#### Scenario: Provider validation succeeds
- **WHEN** the selected provider validates successfully
- **THEN** Aivo marks the provider as ready and enables the action to continue to project selection

#### Scenario: Provider validation fails
- **WHEN** the selected provider validation fails
- **THEN** Aivo shows a safe error message, keeps initialization incomplete, and allows the user to edit the configuration

### Requirement: Make remote provider use visible
Aivo SHALL distinguish local providers from remote API providers and make external data transfer visible before use.

#### Scenario: Remote provider selected
- **WHEN** the user selects OpenAI, Claude Code, Gemini, Codex-compatible, Custom API, or another remote provider option
- **THEN** Aivo displays that task prompts and relevant project context may be sent to the configured provider endpoint when the user starts an approved task

#### Scenario: Custom API selected
- **WHEN** the user selects Custom API
- **THEN** Aivo shows fields for display name, protocol, base URL, credential reference, and default model or profile

### Requirement: Block tasks until setup is ready
Aivo SHALL block code-to-delivery task execution until setup is complete and a provider is ready.

#### Scenario: User tries to create task before provider ready
- **WHEN** the user attempts to start a code-to-delivery task before provider setup is complete
- **THEN** Aivo blocks task creation and prompts the user to complete provider initialization

#### Scenario: Provider becomes unavailable
- **WHEN** the configured provider becomes unavailable after setup
- **THEN** Aivo shows a provider issue state and prevents new task execution until the provider is fixed or another provider is selected
