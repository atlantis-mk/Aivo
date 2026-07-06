## ADDED Requirements

### Requirement: Tool catalog includes built-in, plugin, and MCP identities
Aivo SHALL expose all effective model-facing tools through one catalog with source, sourceID, registrationID, riskLevel, category, and toolsets.

#### Scenario: Plugin tool appears in catalog
- **WHEN** an enabled plugin registers a tool
- **THEN** the tool catalog includes that tool with source `plugin`, the plugin ID as sourceID, a registrationID, riskLevel, and toolsets

#### Scenario: MCP tool appears in catalog
- **WHEN** an enabled MCP server is ready and exposes tools
- **THEN** the tool catalog includes namespaced MCP tools with source `mcp`, the server ID as sourceID, a registrationID, riskLevel, and toolsets

### Requirement: Stale external tool calls are rejected
Aivo SHALL reject plugin or MCP tool calls whose advertised registration no longer matches the effective tool registry.

#### Scenario: Plugin registration changed
- **WHEN** the model calls a plugin tool using a registrationID from an earlier provider turn and the plugin registration has changed
- **THEN** Aivo rejects the call with a stale tool registration error and does not invoke the plugin process

#### Scenario: MCP tool removed
- **WHEN** the model calls an MCP tool that was advertised earlier but is no longer exposed by the server
- **THEN** Aivo rejects the call with a stale or unavailable tool error and records the failure in tool results

### Requirement: External tool execution is bounded and diagnosed
Aivo SHALL bound plugin and MCP tool execution and persist diagnostics for startup, probe, authentication, schema, timeout, and execution failures.

#### Scenario: MCP auth failure is diagnosed
- **WHEN** an MCP server requires authorization that is missing or expired
- **THEN** Aivo records a diagnostic with server ID, level, safe message, and time created

#### Scenario: Plugin tool timeout is diagnosed
- **WHEN** a plugin tool does not return before its timeout
- **THEN** Aivo records a diagnostic, returns a bounded tool failure, and keeps the session event stream queryable

### Requirement: MCP prompts and resources are inserted explicitly
Aivo SHALL let users insert MCP prompts and resources into a session only through explicit UI or service actions.

#### Scenario: User inserts MCP prompt
- **WHEN** the user selects an MCP prompt and provides required arguments
- **THEN** Aivo appends the normalized prompt content to the selected session context with server and prompt provenance

#### Scenario: MCP resource is not auto-injected
- **WHEN** an MCP server exposes resources and a coding session starts
- **THEN** Aivo does not include those resources in model context unless the user explicitly inserts or references them
