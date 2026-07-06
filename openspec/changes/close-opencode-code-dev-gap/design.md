## Context

Aivo already includes a durable session runtime, model-provider abstraction, coding tool registry, permission engine, plugin manager, MCP manager, terminal service, and desktop workbench timeline. The remaining replacement gap is not a greenfield agent rewrite; it is the reliability and completeness needed for desktop code development parity with the parts of opencode that matter to Aivo's scope.

The implementation must stay within the current Electron + Go + React architecture. Go remains the owner of local side effects, tool execution, persistence, permission decisions, and bridge adapters. React remains the owner of review and interaction surfaces through typed service clients.

## Goals / Non-Goals

Goals:

- Make one desktop coding session recoverable across interruption, provider/tool failure, and app restart.
- Let the model use robust code intelligence, file tools, shell/test tools, plugin tools, and MCP tools under one permission and catalog contract.
- Make all local side effects observable and reviewable through the workbench.
- Define replacement readiness through repeatable real coding tasks rather than blanket opencode feature parity.

Non-goals:

- Do not implement CLI, TUI, SDK, GitHub Action, server API parity, enterprise collaboration, remote execution, or plugin marketplace behavior.
- Do not migrate opencode code or replace Aivo's Go app-layer orchestration with opencode's Effect runtime.
- Do not auto-inject MCP prompts/resources into model context without explicit user action.

## Decisions

### 1. Incremental hardening over runtime rewrite

The current `domain / app / infra` boundaries remain intact. New behavior is added as app-layer use cases and additive persistence records. Existing session, tool, permission, plugin, MCP, and workbench APIs should be extended rather than replaced unless a compatibility shim is explicitly retained.

### 2. Execution state is separate from generic session identity

Session records remain durable work threads. Execution-specific state such as active run status, pending inputs, interrupt status, compaction status, and recovery markers belongs in execution state records linked to the session. This prevents generic sessions from becoming tied to one coding runner implementation.

### 3. Provider-turn safety is a persistence rule

Each provider turn must durably record advertised tool identities, started tool calls, completed settlements, and assistant continuation before the next provider request depends on them. On startup, any tool call still marked running is surfaced as interrupted and is not replayed automatically.

### 4. Code intelligence is best-effort but structured

Aivo should prefer language-server-backed diagnostics, symbols, definitions, and references when the workspace supports them. If no language server is available, the existing bounded scan fallback remains available for symbol search and other requests return a structured unavailable result instead of failing the whole task.

### 5. Permissions are action metadata, not prompt guidance

All sensitive tools must formulate permission metadata before execution. Saved approvals match concrete workspace, session, tool, action, path or command key, and relevant dimensions such as network, stdin, env keys, external roots, and source. Permission prompts and stored rules must not include secret values.

### 6. Plugin and MCP tools share the same runtime contract

Built-in, plugin, and MCP tools appear in one catalog with stable source, sourceID, registrationID, risk level, and toolsets. A tool call whose advertised registration no longer matches the active registry is rejected as stale. Plugin and MCP diagnostics remain visible in settings and tool detail surfaces.

### 7. Acceptance is task-matrix based

Replacement readiness is measured by a repeatable set of representative coding tasks. Passing the matrix requires successful implementation, reviewable outputs, correct permission behavior, and documented verification results.

## Data and API Shape

- Extend `SubmitSessionMessageRequest` to consistently support `delivery`, `resume`, `agentMode`, `modelRef`, `toolsets`, and `permissionScope`.
- Add session execution request/response types for interrupt, resume, compaction, execution state, and event cursor listing.
- Add code intelligence result types for language server status, diagnostics, symbols, definitions, and references.
- Extend permission request metadata with action, tool name, paths, command key, network, stdin, env keys, external roots, and source.
- Keep `ToolCatalogEntry` as the catalog shape and require complete identity and risk metadata for every effective tool.

## Risks / Trade-offs

- Real LSP integration can be unreliable across languages. Mitigation: start with Go and TypeScript/JavaScript plus explicit unavailable states and fallback scans.
- Queued input and resume can create surprising continuation behavior. Mitigation: implement deterministic delivery modes, visible execution state, and tests for steer versus queue.
- Saved approvals can become too broad. Mitigation: match on concrete metadata and require fresh approval when action details change.
- MCP and plugin tools may be flaky or hostile. Mitigation: timeout, diagnostics, source identity, permission checks, bounded outputs, and no automatic prompt/resource injection.

## Rollout

Implement in five phases: runtime recovery, code intelligence, tool/permission hardening, plugin/MCP productionization, and workbench plus acceptance matrix. Each phase must keep existing coding flows working and add tests before expanding UI exposure.
