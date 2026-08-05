# Aivo functional requirements

Each section owns one stable behavior. IDs are never reused; replaced or removed requirements remain with a status marker.

### REQ-PROVIDER-001 Provider configuration and health

Aivo must let the user configure and inspect supported provider accounts, resolve an available model, surface actionable authentication or dependency errors, and run provider diagnostics without placing credentials in renderer state or logs. Every local executable tool identity must already match the global `^[A-Za-z0-9_-]+$` Provider-safe contract with a 64-byte maximum before it enters a Provider request. Provider adapters must serialize and observe that exact canonical identity without encoding, decoding, escaping, or alias lookup.

Acceptance: `AT-PROVIDER-001` covers provider registry/configuration behavior, unchanged canonical tool names across declarations, history, final responses, and streams, plus the provider smoke path.

### REQ-PROJECT-001 Local project context

Aivo must let the user add or open a local project, maintain project-scoped runtime configuration, and isolate conversation, tool, worktree, and discovery context between projects.

Acceptance: `AT-PROJECT-001` covers project service and isolation behavior.

### REQ-PROJECT-002 Initial workspace for unscoped conversations

Aivo must present one local default directory during initialization, let the user accept it or select another directory, and persist the confirmed path before initialization completes. Aivo uses that same directory as the coding context for every temporary or otherwise projectless conversation. These conversations remain unscoped in project navigation. Aivo must not create per-conversation directories or silently switch a confirmed path. If the configured root is missing, Aivo recreates that exact root; an unconfigured path, a non-directory at that path, or a creation failure produces an actionable error without changing explicitly selected project behavior.

Acceptance: `AT-PROJECT-002` covers default-path presentation and confirmation, initialization validation, repeated unscoped conversation creation against one directory, exact-root recreation, explicit-project precedence, non-directory refusal, and absence of per-conversation directory creation.

### REQ-PROJECT-003 Agent project catalog and immutable session binding

Aivo must provide a trusted namespaced built-in extension that lets an eligible Agent query the bounded local project catalog, register an existing absolute local directory without creating or cloning content, and bind only its current unscoped coding conversation to an exact returned project. Project query is read-only; registration and association follow the active permission mode and identify the exact target. A conversation's first project association atomically updates its durable project ownership and coding context. The same association is idempotent, while switching, detaching, arbitrary-session mutation, association from a specialized workspace, and association with live interactive terminals are refused. After association, the next model request and desktop state use the bound project's workspace-dependent context without adding another unqualified core tool.

Acceptance: `AT-PROJECT-003` covers bounded query/pagination, existing-directory registration, hidden restoration, permission behavior, immutable and concurrent association, failure rollback, workspace refresh, and desktop project-state presentation.

### REQ-SESSION-001 Conversation and task execution

Aivo must let the user create or continue a conversation, submit a task, receive streaming progress, cancel execution, and recover or retry after an actionable failure without corrupting prior history. Manually activated tools belong only to the selected conversation; a selection made before a conversation exists is consumed by the next created conversation exactly once and is not a global default. Cancelling a turn keeps its user-visible history but terminalizes its owned execution and interactions; later model requests must not treat the cancelled turn's instructions or approvals as active work.

Acceptance: `AT-SESSION-001` covers session runtime, turn, streaming, cancellation, recovery, session-scoped and one-shot draft tool activation, pending-interaction cleanup, and isolation of a cancelled turn from the next model request.

### REQ-WORKSPACE-001 Sidebarless chat workspace

The desktop project chat workspace must prioritize the active conversation without reserving persistent left, right, or bottom panels. New conversation, history, project context, plugins, settings, model, permission, agent-mode, attachment, microphone, and submit entry points must remain reachable through compact top-level or composer controls, while conversation and permission/question states preserve responsive layout and visible keyboard focus. Persistent or top-bar-triggered standalone right activity panels and bottom terminal panels, their resize handles, and their top-bar triggers are not mounted in this workspace; a user-opened contextual inspector governed by `REQ-TOOL-002` may temporarily share the chat width.

Acceptance: `AT-WORKSPACE-001` combines lint/build with wide and narrow screenshots plus focused acceptance for navigation reachability, composer controls, scrolling, overflow, absence of auxiliary panel chrome, and empty/conversation states.

### REQ-TOOL-001 Controlled local tool execution

Aivo must expose exactly the unqualified `read`, `bash`, `edit`, and `write` tools to a default coding Agent in stable order. The four primitives share one active Execution Environment and retain validated inputs, explicit permission policy, bounded model output, cancellation, deterministic cleanup, and safe recorded results. Workspace file paths supplied to `read`, `edit`, and `write` are relative to the active workspace root; only an exact retained-output reference returned by the Host may be absolute for `read`. `read` reads one bounded text or supported image file; `bash` runs one foreground non-interactive Bash command; `edit` atomically applies one batch of exact, unique, non-overlapping replacements against one original file snapshot; and `write` atomically creates or completely overwrites one text file. `apply_patch` and specialized file, Git, test, build, formatting, diagnostic, terminal, LSP, web, automation, and sub-Agent executors are not default tools and have no execution aliases. Workspace policy and confirmation are correctness controls, not process containment; the active local or external Execution Environment owns actual authority and isolation.

Acceptance: `AT-TOOL-001` covers the exact registry and schemas including relative-path guidance, text/image reads, Bash output/artifacts/timeouts, exact atomic edit/write behavior, permission and environment boundaries, cancellation/cleanup including invalidation of turn-owned pending approvals, direct `apply_patch` removal, historical fallback rendering, and safe result recording.

### REQ-TOOL-002 Contextual tool activity inspector

The desktop conversation must present every visible tool call as an accessible label containing the tool's name, arranged from left to right in a compact wrapping activity region without interleaved execution-description text. Activating any label or the surrounding activity region must open the same contextual right-side inspector for that complete activity, not a single-label view. The inspector must automatically open when a new tool call is received in an active conversation, without opening merely because historical calls were loaded. Manually closing it suppresses further automatic opening for that conversation while preserving manual label activation; a different or newly created conversation resets that suppression. The inspector must animate into the chat layout and present every tool call in the activity as one timeline ordered by call time, using an associated invocation description as the item title only when that description exists and showing the tool name as supporting text alongside execution status. Each timeline item must open and dismiss an overlapping in-panel detail view. The inspector must not reintroduce a persistent auxiliary workspace or terminal panel and must preserve responsive layout, keyboard focus, safe result rendering, and live status updates.

Acceptance: `AT-TOOL-002` combines lint/build with wide and narrow interaction screenshots covering per-tool naming, left-to-right wrapping, omission of interleaved execution descriptions, label and activity-region activation, complete activity coverage, call-time ordering and status, live-call automatic opening, historical-load non-opening, per-conversation manual-close suppression and reset, push-open/close animation, stacked detail navigation, long and missing content, repeated selection, and running, approval, success, and failure states.

### REQ-EXTENSION-001 Skills, plugins, and MCP

Aivo must discover supported built-in, process, local-service, external-service, and static extensions from a versioned language-neutral manifest without executing untrusted code. Trust, enablement, readiness, eligibility, activation, authorization, and execution are distinct states. Before each primary model request, the Host prepares a bounded eligible resource catalog across imported Skills, enabled plugins, Manifest v1 extensions, and MCP adapters; the auxiliary resolver selects exact eligible Skill, context, and tool IDs plus the exact selected Skill subset whose instructions are required. The Host—not the auxiliary model—injects canonical summaries for selected Skills, loads full Skill instructions only for that validated subset, injects selected extension context, and then freezes exact selected tool registration and schema identities in a Tool Snapshot assembled from core, mode-default, session-pinned, bounded same-session warm, and bounded auxiliary-selected tools. Catalog preparation may establish readiness or refresh an already enabled source, but selection cannot install, import, trust, enable, bind credentials, authorize, or execute a tool. Automatic Skill/context selection is request-scoped, and manual activation must not propagate between conversations. Every executable canonical tool name is an ASCII `_`/`-`-namespaced identifier matching the global Provider-safe contract before registration. Manifest v1 rejects invalid contributed names; generated MCP adapter names follow the same rule while retaining the upstream MCP name separately for execution. The same canonical name remains authoritative through catalogs, activation, Tool Snapshots, Provider declarations and history, returned calls, permission evaluation, execution, UI emission, and persistence; collisions are rejected and no Provider wire alias exists. Auxiliary selection receives only sanitized eligible catalog summaries and cannot author injected summary content. Extension processes, streams, calls, services, artifacts, and views have bounded output/backpressure, cancellation, draining, restart/update behavior, and deterministic teardown. Complex extension Web views use isolated Host surfaces without Node integration, privileged preload access, ambient credentials, or unrestricted navigation.

Acceptance: `AT-EXTENSION-001` covers Manifest/Protocol v1 validation including global name characters and length, all runtime types, trust and lifecycle transitions, static/dynamic catalog preparation, request-scoped canonical Skill summary and validated instruction/context injection, session activation isolation, registration collision refusal and unchanged Provider identity, schema drift, bounded auxiliary activation, immutable Tool Snapshots, plugin and MCP adaptation with separate upstream names, partial catalog failure and recovery, process/service failure and recovery, isolated Web surfaces, credential binding, update/removal, and historical fallback behavior.

### REQ-WORKTREE-001 Worktree and parallel work lifecycle

Aivo must create, associate, inspect, and clean up task worktrees and parallel agent activity with a visible owner and without deleting user work implicitly.

Acceptance: `AT-WORKTREE-001` covers worktree service ownership and parallel agent behavior.

### NFR-SECURITY-001 Secret and privilege isolation

Provider credentials, OAuth tokens, authorization headers, signing material, and other secrets must remain in privileged local services or approved secure storage. They must not enter renderer persistence, logs, crash output, analytics, committed fixtures, or user-visible diagnostics.

Acceptance: `CT-SECURITY-001` combines targeted credential/redaction tests with repository secret review.

### NFR-RELIABILITY-001 Lifecycle and data recovery

Long-running calls, goroutines, streams, terminals, subprocesses, MCP/LSP clients, worktrees, and pending user interactions must have an owner, cancellation path, bounded output/backpressure, deterministic teardown, and actionable failure state. Cancellation must not leak executable intent, approvals, or questions into a later turn. Persistence changes must be transactional and recoverable from a pre-migration backup.

Acceptance: `CT-RELIABILITY-001` covers runtime cancellation/cleanup, cross-turn isolation, pending-interaction teardown, and migration failure/rollback evidence when schema changes apply.

### NFR-UI-001 Responsive and complete states

Desktop layouts must support content-driven wide and narrow sizes, keyboard/focus behavior, scrolling and overflow, and loading, empty, error, long-content, cancellation, and permission states without modifying generated routes or shared primitives under `apps/desktop/src/components/ui`.

Acceptance: `AT-UI-001` combines lint/build with wide/narrow screenshots and focused manual acceptance for changed flows.

### NFR-UI-002 Desktop visual scale and theme

The desktop renderer must use a shared, semantic typography and control scale grounded in Apple HIG logical reference dimensions while remaining usable across the supported macOS, Windows, and Linux Electron targets. The theme must preserve readable hierarchy, compact pointer-oriented controls, semantic light/dark colors, visible keyboard focus, and responsive reflow without modifying generated routes or shared primitives under `apps/desktop/src/components/ui`.

Acceptance: `AT-UI-002` combines theme-token inspection, lint/build, representative wide/narrow screenshots, overflow checks, and focused keyboard/primary-action acceptance.

### NFR-OBSERVABILITY-001 Safe operation diagnostics

Important backend operations must emit structured start, completion, cancellation, and failure information with an operation ID, duration, and safe dependency/error classification. Logs and diagnostics must exclude secrets and raw sensitive prompt or tool content.

Acceptance: `CT-OBSERVABILITY-001` covers structured event fields and redaction behavior.
