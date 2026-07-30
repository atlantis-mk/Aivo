# Aivo functional requirements

Each section owns one stable behavior. IDs are never reused; replaced or removed requirements remain with a status marker.

### REQ-PROVIDER-001 Provider configuration and health

Aivo must let the user configure and inspect supported provider accounts, resolve an available model, surface actionable authentication or dependency errors, and run provider diagnostics without placing credentials in renderer state or logs.

Acceptance: `AT-PROVIDER-001` covers provider registry/configuration behavior and the provider smoke path.

### REQ-PROJECT-001 Local project context

Aivo must let the user add or open a local project, maintain project-scoped runtime configuration, and isolate conversation, tool, worktree, and discovery context between projects.

Acceptance: `AT-PROJECT-001` covers project service and isolation behavior.

### REQ-SESSION-001 Conversation and task execution

Aivo must let the user create or continue a conversation, submit a task, receive streaming progress, cancel execution, and recover or retry after an actionable failure without corrupting prior history.

Acceptance: `AT-SESSION-001` covers session runtime, turn, streaming, cancellation, and recovery behavior.

### REQ-TOOL-001 Controlled local tool execution

Aivo must execute file, shell, terminal, code-intelligence, web, and agent tools through registered application services with validated inputs, explicit permission policy, bounded output, cancellation, and recorded safe results.

Acceptance: `AT-TOOL-001` covers tool registration, permissions, filesystem boundaries, shell/terminal lifecycle, and result recording.

### REQ-EXTENSION-001 Skills, plugins, and MCP

Aivo must discover and activate supported skills, plugins, and MCP servers through explicit manifests or configuration, isolate their lifecycle, validate exposed capabilities, and surface connection or execution failures without leaking credentials.

Acceptance: `AT-EXTENSION-001` covers discovery, manifest validation, MCP lifecycle, and plugin tool behavior.

### REQ-WORKTREE-001 Worktree and parallel work lifecycle

Aivo must create, associate, inspect, and clean up task worktrees and parallel agent activity with a visible owner and without deleting user work implicitly.

Acceptance: `AT-WORKTREE-001` covers worktree service ownership and parallel agent behavior.

### NFR-SECURITY-001 Secret and privilege isolation

Provider credentials, OAuth tokens, authorization headers, signing material, and other secrets must remain in privileged local services or approved secure storage. They must not enter renderer persistence, logs, crash output, analytics, committed fixtures, or user-visible diagnostics.

Acceptance: `CT-SECURITY-001` combines targeted credential/redaction tests with repository secret review.

### NFR-RELIABILITY-001 Lifecycle and data recovery

Long-running calls, goroutines, streams, terminals, subprocesses, MCP/LSP clients, and worktrees must have an owner, cancellation path, bounded output/backpressure, deterministic teardown, and actionable failure state. Persistence changes must be transactional and recoverable from a pre-migration backup.

Acceptance: `CT-RELIABILITY-001` covers runtime cancellation/cleanup tests and migration failure/rollback evidence when schema changes apply.

### NFR-UI-001 Responsive and complete states

Desktop layouts must support content-driven wide and narrow sizes, keyboard/focus behavior, scrolling and overflow, and loading, empty, error, long-content, cancellation, and permission states without modifying generated routes or shared primitives under `apps/desktop/src/components/ui`.

Acceptance: `AT-UI-001` combines lint/build with wide/narrow screenshots and focused manual acceptance for changed flows.

### NFR-OBSERVABILITY-001 Safe operation diagnostics

Important backend operations must emit structured start, completion, cancellation, and failure information with an operation ID, duration, and safe dependency/error classification. Logs and diagnostics must exclude secrets and raw sensitive prompt or tool content.

Acceptance: `CT-OBSERVABILITY-001` covers structured event fields and redaction behavior.
