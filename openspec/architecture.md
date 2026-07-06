## Architecture Overview

Aivo is a local-first desktop AI productivity workbench built as a single modular Electron application. Go owns local capabilities, task orchestration, permission and confirmation gates, persistence, and desktop bridge adapters. React and TypeScript own the workbench UI, review surfaces, and user-facing interaction state. The V1 architecture prioritizes the code-to-delivery workflow while preserving room for later data, document, video, plugin, and optional remote-execution capabilities through explicit OpenSpec changes.

## Tech Stack

- Desktop runtime: Electron with Go.
- Desktop application identity: bundle ID `com.aivo.desktop`.
- Backend / local services: Go, organized around `domain / app / infra` boundaries.
- Go module path: `aivo` until a canonical remote repository path is established.
- Frontend: React + TypeScript + Vite template, pnpm, TanStack Router for client-side routing, Tailwind CSS v4 through `@tailwindcss/vite`, and shadcn/ui as the default UI component system.
- Persistence: SQLite for local application state and metadata, accessed through GORM with the pure-Go `github.com/glebarez/sqlite` dialector.
- Local artifacts: filesystem storage for project source, large files, generated deliverables, and exportable artifacts, with SQLite storing references and state.
- External capability providers: model APIs, remote services, publishing destinations, and future sync services are optional adapters, not core assumptions.
- Provider configuration: first-run setup records local provider preferences and non-secret metadata for direct provider choices such as OpenAI, Claude Code, Gemini, and Codex-compatible, plus a Custom API option for advanced endpoints. Custom API configurations select a protocol such as OpenAI, Claude Code, Gemini, Codex-compatible, or a future adapter protocol; credentials must stay in OS credential storage or explicit user-managed configuration.
- Session Runtime: sessions are durable first-class resources composed of turns and append-only events. Coding-specific state such as repository path, branch, changed files, and command context lives in coding context records rather than the generic sessions table.
- Code intelligence: language-server-backed diagnostics, symbols, definitions, and references are app-orchestrated local capabilities with bounded fallbacks, not frontend-side filesystem or process access.
- Tool extensibility: built-in, plugin, and MCP tools share the app-layer tool registry, permission engine, bounded output handling, and session event audit path before they are exposed to model providers.
- Testing: behavior-first automated tests, with type checks and relevant tests as required quality gates once commands are configured.

## System Boundaries

Aivo includes the desktop application, Go local services, React workbench UI, local SQLite database, local filesystem access mediated through Go services, task execution orchestration, reviewable logs and artifacts, and explicit confirmation flows for sensitive actions.

External systems include model providers, remote APIs, Git hosting, package registries, browser sessions, OS credential stores, collaboration or sync services, and any publish/upload destination. These must be accessed through `infra` adapters and governed by `app`-level policy.

V1 does not require enterprise permissions, centralized administration, audit compliance, a marketplace, always-on cloud execution, or multi-user collaboration. Those capabilities require explicit future architecture updates.

## Module Boundaries

| Module | Responsibility | Depends On | Must Not Depend On |
| --- | --- | --- | --- |
| `domain` | Core models, invariants, business rules, pure decision logic, and state transitions. | Go standard library and domain-owned value types. | Electron, React, filesystem, process execution, SQLite drivers, network clients, model providers, OS APIs, UI state. |
| `app` | Use-case orchestration, task lifecycle, permission checks, confirmation gates, transactions, cancellation, retries, and coordination across domain rules and infra ports. | `domain` and consumer-owned interfaces for required side effects. | Aivo bridge handlers as business logic, React components, concrete infra implementations when an interface is warranted. |
| `infra` | Filesystem, process execution, SQLite repositories, OS integration, credential access, model/API clients, Aivo bridge adapters, logging sinks, and artifact storage. | `app` interfaces, `domain` types where needed, concrete platform libraries. | React UI logic, product workflow decisions, duplicated domain rules. |
| Frontend routing | TanStack Router route tree, layout routes, URL parameters/search state, route-level loading/error boundaries, and navigation between setup, workbench, task, artifact, and settings surfaces. | React, frontend client/services, typed view models, route modules. | Go business rules, raw Aivo bridge handlers outside service boundaries, direct filesystem/process/model access, side-effect policy. |
| Frontend client/services | Typed access layer for Aivo bridge calls, frontend request/response mapping, data loading hooks, and workflow service wrappers. | Generated Aivo bridge handlers and frontend types. | Go domain rules, direct filesystem/process/model access, scattered raw Aivo bridge calls from arbitrary components. |
| React UI/features | Workbench screens, task review, plans, diffs, logs, artifact inspection, controls, loading/empty/error/success states. | Frontend routing, frontend client/services, feature-local UI state, typed view models. | Raw Aivo bridge handlers outside the client layer, backend-only business rules, direct side effects. |
| SQLite persistence | Durable local application state, task/project/tool-run/artifact metadata, resumable workflow state, and indexes. | `infra` repository implementations and migration tooling. | Large artifact payload ownership, source-of-truth project files, secrets in plaintext. |
| Provider adapters | Model and agent-provider configuration, health checks, capability discovery, and provider invocation adapters. | `app` provider interfaces, `domain` provider types where needed, concrete SDK/CLI/process clients. | React UI logic, hardcoded credentials, bypassing confirmation or external-transfer policy. |
| Session Runtime | Session, turn, event, summary, checkpoint, tool-call, and context-building use cases for resumable assistant work. | `domain` session models, `app` service interfaces, SQLite repositories, provider-independent summarizer interfaces. | UI-specific state, coding-only fields in generic session records, raw secrets in visible event content. |
| Tool Runtime and Code Intelligence | Built-in, plugin, MCP, shell, file, and language-server capabilities for coding sessions, including tool catalog identity, permission evaluation, output bounding, and audit records. | `domain` tool/code-intelligence types, `app` permission/session services, infra process/filesystem/LSP/MCP/plugin adapters. | React UI logic, provider-specific payloads as execution rules, unbounded filesystem/process/network access. |

## Data Model Principles

- Use stable identifiers for persisted and long-running entities such as projects, tasks, tool runs, artifacts, confirmations, and local context records.
- Use stable identifiers for sessions, turns, session events, tool calls, summaries, checkpoints, and domain context records.
- SQLite stores local application state and metadata; project source files, large generated files, media, and exportable deliverables remain on the filesystem with persisted references.
- The database file lives under the Electron application data directory, not inside the user-selected project repository.
- Provider settings stored in SQLite must be limited to non-secret metadata such as provider type, optional protocol, display name, base URL when applicable, selected model or command profile, status, last validation time, and credential reference labels.
- Database schemas must be migrated through GORM models, `AutoMigrate`, and targeted `Migrator` compatibility steps before persistence-dependent features ship.
- Migration policy must apply forward migrations in order, create a pre-migration database backup once user data exists, stop persistence-dependent startup on migration failure, and avoid destructive schema changes without an explicit recovery path.
- Keep data ownership clear: `domain` defines core entity meaning, `app` owns workflow state transitions, and `infra` owns storage representation.
- Keep Session Core generic: coding, email, calendar, research, and task-specific context belongs in separate domain context tables or records linked to a session.
- Do not store secrets, credentials, personal data, raw model transcripts, or sensitive file contents in logs, artifacts, snapshots, SQLite, or config unless explicitly required and protected.
- Treat user-provided paths, imported files, external API responses, model output, and command arguments as untrusted boundary input.

## API / Interface Principles

- React initiates user intent; Go `app` use cases decide whether and how work proceeds.
- Aivo bridge handlers are adapter endpoints. They validate boundary input, call `app` use cases, and return UI-safe response shapes.
- Important boundary request and response shapes must be explicit and typed in Go and TypeScript.
- Keep Aivo API access behind a typed frontend client or service layer. Do not scatter generated binding calls through arbitrary React components.
- Keep route modules thin: routes may compose layouts, parse typed path/search state, coordinate loading, and render route-level pending/error states, but workflow decisions and Aivo bridge calls must stay in frontend services/hooks and Go app use cases.
- Define Go interfaces at the consumer side when they clarify testability or ownership across `app` and `infra`; avoid premature interfaces around one-off concrete code.
- Use `context.Context` for I/O, process execution, external calls, persistence operations, and cancellable long-running work.
- External providers, shell execution, file operations, and publishing destinations must be accessed through infra adapters selected and governed by app-layer policy.
- Plugin tools, MCP tools, code-intelligence tools, and shell/test tools must be registered through app-layer tool catalog and permission policy before model exposure.
- Provider setup and invocation must use typed app-layer interfaces so task orchestration can select a configured provider without coupling workflow rules to a specific model API, CLI, or SDK.

## State, Side Effects, and Error Handling

- Go `app` is the authoritative owner of task lifecycle, permission checks, confirmation requirements, cancellation, retries, idempotency policy, and side-effect sequencing.
- React owns transient interaction state and rendering state, but not authoritative business workflow state.
- Long-running work must be observable through task state, logs, progress events, result artifacts, or verification results as appropriate.
- File writes, command execution, credential use, model/API calls, uploads, publishing, deletion, overwrites, purchases, and external data transfer must pass through app-layer policy and confirmation gates.
- Side effects expected to retry must be idempotent or guarded by stable operation identifiers.
- Errors must be explicit, contextual, and preserved for diagnosis. Do not silently swallow Go errors or Promise rejections.
- User-facing errors should be safe to display and must not leak secrets, credentials, or sensitive local paths beyond what is necessary for the user to act.
- Goroutines and frontend async work must have clear ownership, cancellation, and shutdown behavior.

## Security and Permissions

- Aivo is local-first by default. Remote model/API usage, uploads, publishing, sync, or external transfer must be visible to the user and controlled by settings or explicit confirmations.
- Sensitive actions require user confirmation, including credential use, publishing externally, deleting or overwriting important files, making purchases, executing risky commands, or sending data outside the local environment.
- Credentials and tokens must not be hardcoded or stored in repo files. Prefer OS credential storage or explicit user-managed configuration when credential persistence is needed.
- First-run provider setup must distinguish local/CLI providers from remote API providers and make any external data transfer visible before the provider is used for task execution.
- Direct provider choices validate with provider-specific defaults and fewer required fields. Custom API providers validate protocol-specific readiness using the configured base URL, credential reference, and selected model/profile. Local command profiles, if later added for a protocol, must use a separate command-backed adapter path and remain governed by the same confirmation and non-secret persistence rules.
- Aivo bridge methods must validate boundary inputs and must not expose broad filesystem, process, or network primitives without app-layer authorization.
- Logs, generated artifacts, snapshots, and config files must avoid secrets and unnecessary personal data.
- Session events default to user-visible `normal` content only when safe. Internal reasoning, hidden system data, sensitive tool results, and redacted payloads must not be shown in normal UI or included in model resume context by default.
- V1 optimizes for a single power user and does not implement enterprise authorization, org administration, or audit compliance.

## Testing Strategy

- Use behavior-first tests that verify user-visible behavior, OpenSpec scenarios, workflow rules, and failure paths.
- Keep domain tests focused on core rules, invariants, state transitions, and pure decision logic.
- Keep app tests focused on orchestration, confirmation gates, cancellation, retries, idempotency, permission denial, and error mapping.
- Use integration tests for behavior that depends on real infra adapters such as SQLite migrations, filesystem operations, process execution, Aivo bridge boundaries, OS integration, or provider clients.
- Frontend tests should cover important workflow states: loading, empty, error, success, progress, reviewable outputs, and disabled or pending confirmation states.
- Required quality gates are relevant automated tests and type checks once project commands are initialized. Lint and build commands should run when configured or when the change risk justifies them.
- Report any unrun checks with the reason.

## Code Organization Rules

- Keep the repository as a single modular desktop application unless a future architecture decision explicitly changes this.
- Keep Go business logic in `domain` and `app`; keep Aivo bridge handlers and concrete side-effect implementations in `infra`.
- Keep React presentational components focused on rendering and interaction; move workflow orchestration, data loading, and Aivo bridge calls into hooks, clients, or feature services.
- Build frontend UI with shadcn/ui components by default. Use the project shadcn registry/configuration for component source, design tokens, icons, and aliases instead of inventing parallel component primitives.
- Use TanStack Router for frontend navigation. Define durable routes for first-run setup, provider setup, project workbench, task review, artifact review, and settings once those surfaces exist.
- Keep route modules under frontend `src/routes` and feature code under `src/features/*`. Routes compose layouts and call feature hooks/services; feature modules own task, project, provider, artifact, and confirmation UI.
- Do not add TanStack Query by default for the MVP. Route loaders and feature hooks may call typed frontend services directly; introduce a cache/query library only when repeated server-state caching, invalidation, or background refresh becomes necessary.
- Do not duplicate business rules between frontend and backend. Frontend validation may improve UX, but Go boundary validation remains authoritative.
- Keep packages and components cohesive. Avoid large mixed-purpose utility modules, service objects, or UI components.
- New dependencies require a documented reason and should be added only when the standard library or existing dependencies are insufficient.
- Future changes should update this file only when they alter architecture-level direction, module boundaries, major dependencies, persistence strategy, security model, or testing policy.

## Architecture Decisions

### ADR-001: Local-first Electron desktop architecture

**Decision:** Use Electron + Go + the React/TypeScript/Vite frontend template as the default application architecture. Use pnpm for frontend package management, Go module path `aivo` until a canonical remote repository exists, and bundle ID `com.aivo.desktop`.

**Rationale:** Aivo needs local project access, desktop workflow integration, reviewable task execution, and a rich workbench UI. Electron lets Go own local capabilities and bridge them to a React/TypeScript frontend without making the product a pure web SaaS or CLI-only tool.

**Alternatives Considered:** Pure web SaaS, CLI-only agent, Electron with Node-first backend, and IDE-only extension.

**Consequences:** Desktop packaging and Aivo bridge design are architectural concerns. Go remains the authoritative boundary for local side effects, while React focuses on interaction and review surfaces. Future repository publishing may update the Go module path through an explicit architecture update.

### ADR-002: Single-repo modular application with `domain / app / infra` boundaries

**Decision:** Keep V1 as a single repository and a single modular desktop application. Use Go `domain / app / infra` boundaries for backend code.

**Rationale:** The first product phase needs fast iteration and coherent local execution without the overhead of distributed services. The layering keeps business rules testable and prevents Electron, filesystem, process, persistence, and provider details from leaking into core behavior.

**Alternatives Considered:** Separate frontend/backend packages with stronger process boundaries, multiple local services, and early remote agent architecture.

**Consequences:** Module boundaries must be enforced by code organization and review. Future multi-process or remote execution support requires an explicit architecture change.

### ADR-003: SQLite as the default local persistence store

**Decision:** Use SQLite for local application state and metadata, accessed through GORM with the pure-Go `github.com/glebarez/sqlite` dialector and GORM-managed migrations.

**Rationale:** Aivo is local-first and needs durable task, project, tool-run, artifact, and resumable workflow state. SQLite provides reliable desktop-local persistence, queryability, transactions, and migration support without requiring an external database service.

**Alternatives Considered:** JSON/YAML/Markdown files as the primary store, no default persistence choice until first data feature, and an external database.

**Consequences:** Persistence-dependent features must define migrations, recovery expectations, and stable IDs. Large artifacts and project files remain in the filesystem; SQLite stores references, indexes, and state. Database files live in the Electron app data directory and migration failures block persistence-dependent startup rather than silently continuing.

### ADR-004: Go `app` layer owns execution and sensitive side-effect orchestration

**Decision:** React sends user intent to Go. The Go `app` layer coordinates task execution, permission checks, confirmations, logs, lifecycle state, retries, cancellation, and infra adapter calls.

**Rationale:** Code execution, file writes, external model calls, publishing, deletion, overwrites, and credential usage are high-impact actions. Centralizing policy in `app` preserves local-first control, reviewable autonomy, and consistent safety behavior.

**Alternatives Considered:** Direct Aivo bridge method implementations with embedded business logic and a separate agent service.

**Consequences:** Aivo bridge handlers remain thin adapters. Infra services expose capabilities, but app use cases decide when they can run. A separate agent process or remote executor requires a future architecture update.

### ADR-005: shadcn/ui as the frontend component baseline

**Decision:** Use shadcn/ui as the default React UI component system for Aivo frontend development.

**Rationale:** Aivo needs a consistent desktop workbench UI with reusable, accessible components while still allowing source-level control over component code. shadcn/ui fits the React + TypeScript stack and supports component composition without introducing a separate opaque design-system runtime.

**Alternatives Considered:** Fully custom component primitives, a closed component library, and adopting no default UI system until later.

**Consequences:** Frontend work should prefer installed or registry-provided shadcn/ui components, semantic design tokens, and project aliases. New component dependencies, custom primitives, or substantial deviations from shadcn patterns need documented rationale.

### ADR-006: TanStack Router for frontend navigation

**Decision:** Use TanStack Router as the default client-side router for the React frontend.

**Rationale:** Aivo needs durable desktop navigation across first-run setup, provider configuration, project workbench, task review, artifact review, and settings. TanStack Router fits the TypeScript-first frontend direction, supports typed path and search parameters, and gives route-level pending and error boundaries without moving workflow ownership out of the Go app layer.

**Alternatives Considered:** No router with app-state-only view switching, React Router, and file-system routing tied to a full-stack web framework.

**Consequences:** Frontend initialization should add `@tanstack/react-router` and organize route modules separately from feature components and Electron client/services. Routes remain UI composition and navigation boundaries; task lifecycle, confirmations, and sensitive side-effect decisions remain owned by Go `app` use cases.

### ADR-007: Minimal MVP frontend and data tooling

**Decision:** Use pnpm, Tailwind CSS v4 with the Vite plugin, shadcn/ui `new-york` style with CSS variables, `lucide-react` icons, and project aliases such as `@/components`, `@/features`, `@/lib`, `@/routes`, and `@/services`. Do not add TanStack Query or another frontend server-state cache by default.

**Rationale:** The MVP needs a focused desktop workbench and typed Aivo bridge service boundary more than a broad client-side data stack. Direct route loaders and feature hooks keep the first implementation understandable while preserving a clean place to add caching later.

**Alternatives Considered:** npm or yarn, shadcn default styling without fixed aliases, adding TanStack Query immediately, and colocating all route and feature code.

**Consequences:** Frontend source should keep `src/routes` for route modules, `src/features/*` for user-facing workflow surfaces, `src/services` for typed Electron clients, `src/components` for shared shadcn/ui components, and `src/lib` for small shared utilities.

### ADR-008: Provider validation starts with safe readiness checks

**Decision:** MVP provider setup validates remote API providers with non-secret health/readiness checks and validates local CLI providers by command availability, version output where supported, and non-interactive readiness checks where available. A deterministic manual/minimal planner fallback may exist for development and demo flows, but AI task execution should require an explicitly configured provider.

**Rationale:** CLI providers do not all expose the same auth or health APIs. A safe readiness model avoids scraping secrets or forcing execution during setup while still making blocked provider states visible before task execution when possible.

**Alternatives Considered:** Require full real provider execution during setup, skip validation entirely, or make the manual fallback the primary MVP provider.

**Consequences:** Provider records need statuses such as configured, verified, unverified, unavailable, and failed. First execution must surface safe actionable errors if a configured-but-unverified CLI provider cannot run.

## Constraints / Needs Confirmation

- Application source has not been initialized yet; generated binding locations and exact Electron-created directory names must be confirmed during initialization.
- The final app icon remains pending.
- Concrete command names should be implemented through pnpm workspace scripts, Electron commands, and Go commands during project initialization.
- V1 does not adopt multi-process local services, remote execution, multi-user collaboration, enterprise permissions, or plugin marketplace architecture. Those require future OpenSpec changes.
