## ADDED Requirements

### Requirement: Language server status is visible
Aivo SHALL expose per-project language server status for supported languages without blocking ordinary file tools when no language server is available.

#### Scenario: Supported language server starts
- **WHEN** a project contains supported Go or TypeScript/JavaScript source and the language server can be launched
- **THEN** Aivo reports language server status `ready` with language, root, and startup time metadata

#### Scenario: Language server unavailable
- **WHEN** no supported language server can be launched for a project
- **THEN** Aivo reports status `unavailable` with a safe reason and keeps read, search, and patch tools available

### Requirement: Diagnostics are available as a coding tool
Aivo SHALL provide model-facing diagnostics that return structured problems from language servers or declared fallback diagnostics.

#### Scenario: Diagnostics return problems
- **WHEN** the model calls `lsp_diagnostics` for a supported project with compile or type errors
- **THEN** Aivo returns diagnostics with path, range, severity, source, message, and bounded command or server output

#### Scenario: Diagnostics unavailable result
- **WHEN** diagnostics are requested for an unsupported project
- **THEN** Aivo returns a successful structured unavailable result instead of failing the session run

### Requirement: Definitions and references are available as coding tools
Aivo SHALL provide model-facing tools to locate definitions and references for symbols in supported source files.

#### Scenario: Definition lookup succeeds
- **WHEN** the model calls `lsp_definition` with a workspace-relative file path and position for a supported language
- **THEN** Aivo returns one or more workspace-relative definition locations with path, range, language, and preview text when available

#### Scenario: References lookup succeeds
- **WHEN** the model calls `lsp_references` with a workspace-relative file path and position for a supported language
- **THEN** Aivo returns bounded reference locations ordered by path and line

### Requirement: Symbol search uses LSP first and safe scan fallback second
Aivo SHALL keep `lsp_symbol_search` available and prefer language-server symbols when ready, falling back to bounded source scanning when needed.

#### Scenario: LSP symbol search available
- **WHEN** language server symbols are available for a project
- **THEN** `lsp_symbol_search` returns structured symbol locations from the language server

#### Scenario: Symbol search fallback
- **WHEN** language server symbols are unavailable but source scanning supports the file type
- **THEN** `lsp_symbol_search` returns bounded scan results and marks the result source as fallback
