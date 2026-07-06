## ADDED Requirements

### Requirement: Sensitive tool actions include complete permission metadata
Aivo SHALL classify sensitive tool actions before execution and include enough metadata for precise approval or denial.

#### Scenario: File write permission metadata
- **WHEN** a write, edit, or patch tool targets workspace files
- **THEN** Aivo creates a permission request containing action, toolName, paths, source, and proposed file change summary before mutation

#### Scenario: Shell permission metadata
- **WHEN** a shell, test, or diagnostics tool requests command execution
- **THEN** Aivo creates or evaluates permission metadata containing commandKey, cwd, network, stdin presence, env keys, timeout, and source

#### Scenario: External directory permission metadata
- **WHEN** a tool requests a path outside the selected workspace
- **THEN** Aivo either denies the action or creates an external-root permission request without exposing secret path details beyond what the user needs to decide

### Requirement: Saved approvals match concrete action scope
Aivo SHALL reuse saved approvals only when the current action matches the approved workspace, session or project scope, tool, action, path or command key, and relevant execution dimensions.

#### Scenario: Matching saved approval allows action
- **WHEN** a write tool repeats an approved edit for the same workspace, tool, action, and covered path
- **THEN** Aivo allows the tool without creating a new permission request

#### Scenario: Changed action requires new approval
- **WHEN** a command, path, network mode, stdin use, env key, or external root differs from the saved approval
- **THEN** Aivo requires a new permission decision before executing the action

### Requirement: Plugin and MCP tools are permission governed
Aivo SHALL route plugin and MCP tool execution through the same permission engine when the tool risk or capability requires approval.

#### Scenario: MCP tool requests network action
- **WHEN** an MCP tool is registered from an HTTP or SSE server and the model calls it
- **THEN** Aivo evaluates network/source permission metadata before executing the MCP call

#### Scenario: Plugin tool requests workspace mutation
- **WHEN** a plugin tool declares or attempts a workspace write capability
- **THEN** Aivo requires permission approval before the tool can mutate workspace files

### Requirement: Permission records avoid secret persistence
Aivo SHALL redact secret values from permission prompts, persisted permission requests, and saved approval rules.

#### Scenario: Secret env key requested
- **WHEN** a shell tool request includes an env key that appears secret-bearing
- **THEN** Aivo denies or redacts the env value and persists only the key name and safe reason
