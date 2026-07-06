## ADDED Requirements

### Requirement: Confirmation gates cover extended coding tool risks
Aivo SHALL require confirmation for sensitive coding actions including file writes, overwrites, deletion, shell/process execution, network transfer, stdin/env use, external directories, plugin tools, and MCP tools when policy requires approval.

#### Scenario: Network shell command requires approval
- **WHEN** a shell tool requests inherited network access
- **THEN** Aivo creates a confirmation or permission request that clearly states network access is part of the approved scope

#### Scenario: MCP tool requires approval
- **WHEN** an MCP tool has medium or higher risk and no saved approval covers its source and action
- **THEN** Aivo blocks execution until the user approves or denies the request

### Requirement: Confirmations are invalidated when action scope changes
Aivo SHALL invalidate a pending or remembered confirmation when the command, paths, network mode, env keys, stdin use, external roots, or tool source changes.

#### Scenario: Changed command invalidates approval
- **WHEN** a shell command differs from the command that was approved
- **THEN** Aivo requires a new approval before running the changed command
