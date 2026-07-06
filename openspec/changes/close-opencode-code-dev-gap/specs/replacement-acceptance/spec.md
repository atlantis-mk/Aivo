## ADDED Requirements

### Requirement: Replacement readiness is based on a coding task matrix
Aivo SHALL define and track a repeatable acceptance matrix before claiming it can replace opencode for desktop code development.

#### Scenario: Acceptance matrix exists
- **WHEN** replacement readiness is evaluated
- **THEN** Aivo has documented scenarios for small fix, multi-file feature, debug-to-build flow, interrupted resume, MCP/plugin participation, LSP lookup, and safety approval or denial

#### Scenario: Acceptance result is recorded
- **WHEN** a matrix scenario is run
- **THEN** Aivo records project, prompt, provider, agent mode, permission mode, commands run, files changed, verification outcome, and known limitations

### Requirement: All acceptance scenarios must pass before replacement claim
Aivo SHALL treat replacement readiness as false until every required acceptance scenario has a passing result or an explicitly accepted limitation.

#### Scenario: Incomplete matrix blocks readiness
- **WHEN** at least one required acceptance scenario has no passing result
- **THEN** Aivo reports replacement readiness as incomplete

#### Scenario: Matrix passes
- **WHEN** every required acceptance scenario has a passing result and no blocking limitation remains
- **THEN** Aivo reports replacement readiness as complete for desktop code development only

### Requirement: Acceptance excludes non-scope opencode surfaces
Aivo SHALL exclude CLI, TUI, SDK, GitHub Action, enterprise collaboration, and non-code workflows from replacement readiness scoring.

#### Scenario: CLI parity missing does not fail matrix
- **WHEN** CLI or TUI parity is absent but all desktop code-development scenarios pass
- **THEN** replacement readiness can still be complete for the defined desktop scope
