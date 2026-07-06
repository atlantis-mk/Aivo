## Context

Aivo currently has product, architecture, and engineering standards, but no initialized application source. The first MVP must turn the baseline into a runnable local-first desktop app that demonstrates the code-to-delivery workflow for a single power user.

The durable architecture already chooses a single-repo Electron desktop app, Go backend with `domain / app / infra` layering, React + TypeScript frontend, shadcn/ui as the UI baseline, SQLite for local metadata, filesystem storage for source and large artifacts, and app-layer confirmation gates for sensitive actions.

## Goals / Non-Goals

**Goals:**

- Create the first runnable Electron application with clear Go and React module boundaries.
- Add a first-run welcome and initialization flow that configures a provider before task execution.
- Support selecting a local project, persisting project metadata, and showing basic project context.
- Support a task workflow that can move from user intent to plan, approved execution, observable progress, artifacts, verification results, and delivery notes.
- Centralize sensitive-action policy in the Go app layer and expose pending confirmations to the UI.
- Persist task, tool-run, log, artifact, confirmation, and resumable state in SQLite with migrations.
- Provide behavior-first tests and quality gate commands once tooling is initialized.

**Non-Goals:**

- Full parity with opencode or hermes-agent.
- Multi-user collaboration, enterprise permissions, audit compliance, marketplace, or remote execution.
- Data analysis, document generation, video creation, or plugin ecosystems beyond the MVP architecture extension points.
- Building every advanced provider-specific feature; MVP provider setup covers selection, credential-reference capture, validation, and task-readiness.
- Autonomous destructive changes without explicit confirmation.
- A pure chat-only product surface.

## Decisions

### Decision 1: Initialize a single Electron app in the existing repository

Use Electron with Go and React/TypeScript in one repository, following the global architecture. The source layout should keep Go business logic in `domain` and `app`, concrete side effects in `infra`, Aivo bridge adapters thin, and frontend Electron access behind typed services.

**Rationale:** This proves the desktop workflow with local project access while keeping the product aligned with local-first control and reviewable autonomy.

**Alternatives considered:** A CLI-only MVP would move faster but would not validate the desktop workbench. A web-only MVP would conflict with local-first project access. Electron would duplicate the already selected Electron direction.

### Decision 2: Model tasks as resumable state machines owned by Go app use cases

Represent tasks, plans, tool runs, confirmations, artifacts, logs, verification results, and terminal states explicitly in Go domain/app models. React renders state and sends user intent, while Go decides transitions, permissions, retries, cancellation, and side-effect sequencing.

**Rationale:** The MVP needs long-running work, cancellation, review surfaces, and resumability. Keeping workflow ownership in Go avoids duplicated business rules in frontend components.

**Alternatives considered:** Frontend-owned workflow state would be easier for a demo but would make confirmation gates, cancellation, and persistence brittle. Direct Aivo bridge method logic would blur adapter and business boundaries.

### Decision 2A: Add provider initialization before task execution

First-run setup should show a welcome screen, then require the user to configure at least one provider before code-to-delivery tasks can run. The MVP provider setup exposes direct choices for OpenAI, Claude Code, Gemini, and Codex-compatible so official/common paths are convenient. It also exposes Custom API as an advanced option where the user selects a protocol and supplies a custom endpoint. Provider records store non-secret metadata and credential references only; raw API keys or tokens must not be stored in SQLite, logs, screenshots, or artifacts.

**Rationale:** Aivo cannot complete AI-assisted code-to-delivery work without a configured execution/model provider. Making setup explicit preserves local-first control and makes remote-provider data transfer visible before task execution.

**Alternatives considered:** Deferring provider setup until the first task would create unclear blocked states. Hardcoding one provider would conflict with user control. Hiding official choices inside Custom API would make common setup unnecessarily hard. Making every provider a completely separate implementation would scale poorly, so direct choices should share the provider configuration model where practical.

### Decision 3: Gate all sensitive side effects through app-layer confirmation records

Every sensitive action request produces a confirmation record with action type, target, risk summary, proposed effect, expiry or invalidation behavior, and approval/denial state. Infra adapters only run when the app layer supplies an approved operation.

**Rationale:** Confirmation gates are a core product principle and must be consistent across file writes, command execution, deletion, overwrites, credential use, and external transfer.

**Alternatives considered:** UI-only confirmations would be easy to bypass. Per-adapter confirmation prompts would duplicate policy and make behavior inconsistent.

### Decision 4: Use SQLite for metadata and filesystem references for large content

Add versioned migrations for local metadata tables. Store project source and large artifacts on disk; store stable IDs, paths, hashes or sizes where useful, task relationships, status, and timestamps in SQLite.

**Rationale:** SQLite supports local durability, transactions, resumability, and queryable task history without moving ownership of user source files into the app database.

**Alternatives considered:** JSON files would be simpler but weaker for task history and migrations. Embedding large artifacts in SQLite would complicate backup and performance.

### Decision 5: Build a workbench UI instead of a landing page

The first screen should be the usable product: project selector, task composer, task timeline, confirmations, logs, diffs, artifacts, and verification results. Use shadcn/ui components and project tokens once initialized.

**Rationale:** The product direction requires serious project execution, reviewability, and reduced context switching, not a marketing entry screen.

**Alternatives considered:** A chat-first screen would underrepresent artifacts and review surfaces. A decorative landing page would not validate the MVP workflow.

## Risks / Trade-offs

- [Risk] MVP scope grows into a full autonomous coding agent before foundations are stable. → Mitigation: implement a thin but complete vertical slice first, with explicit extension points and incomplete advanced capabilities marked out of scope.
- [Risk] Process execution and file writes can damage local work. → Mitigation: enforce app-layer confirmations, show targets and risk summaries, and keep logs/artifacts reviewable.
- [Risk] SQLite schema decisions harden too early. → Mitigation: start with stable core entities and versioned migrations; keep large content on the filesystem.
- [Risk] Aivo bridge handlers become a dumping ground for business logic. → Mitigation: keep bindings as adapters and test app/domain use cases directly.
- [Risk] Frontend becomes tightly coupled to generated bindings. → Mitigation: use typed frontend client/services and feature hooks as the only UI access path.
- [Risk] Model/provider integration may require credentials or network transfer. → Mitigation: treat provider calls as optional adapters governed by settings and confirmations, and allow deterministic/manual fallback paths where possible.

## Migration Plan

1. Initialize Electron, Go module `aivo`, the React/TypeScript/Vite frontend, pnpm, TanStack Router, shadcn/ui, and test tooling in the repository.
2. Create Go module boundaries for `domain`, `app`, and `infra`, plus thin Aivo bridge adapters.
3. Add GORM, `github.com/glebarez/sqlite`, GORM model migration tooling, app-data-directory database initialization, and initial metadata schema.
4. Implement project selection and metadata persistence.
5. Implement provider setup, provider validation, and app-ready state.
6. Implement the task lifecycle vertical slice with plan review, confirmations, logs, artifacts, verification results, cancellation, and resume.
7. Build welcome, initialization, and workbench UI around the typed frontend service boundary.
8. Add behavior-first tests for OpenSpec scenarios and run configured type/test/build gates.

Rollback is not user-data-critical before first release. After persistence ships, rollback must preserve the database file and avoid destructive schema changes without a recovery path.

## Resolved Implementation Choices

- Go module path is `aivo` until a canonical remote repository path is established.
- Application bundle ID is `com.aivo.desktop`.
- Frontend package manager is pnpm.
- Electron frontend template is React + TypeScript + Vite.
- SQLite access uses GORM with the pure-Go `github.com/glebarez/sqlite` dialector.
- Migrations are GORM model migrations plus targeted `Migrator` compatibility steps with a pre-migration database backup once user data exists.
- Direct provider adapters validate provider-specific readiness using sensible defaults and the configured credential reference/model profile. Custom API adapters validate the user-provided protocol, base URL, credential reference, and model/profile against a small non-secret readiness request or provider metadata endpoint when available.
- Local command-backed profiles are not part of the default Custom API form; if a protocol later needs command execution, it should use a separate adapter path and confirmation behavior.
- A deterministic manual/minimal planner fallback may exist for development and demo flows, but real AI task execution requires an explicitly configured provider.
