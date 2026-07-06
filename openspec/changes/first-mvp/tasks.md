## 1. Project Initialization

- [x] 1.1 Initialize the Electron application in the existing repository with Go module `aivo`, bundle ID `com.aivo.desktop`, pnpm, and React + TypeScript + Vite frontend manifests.
- [x] 1.2 Add repository setup documentation with install, local development, typecheck, test, and build commands that are available after initialization.
- [x] 1.3 Configure shadcn/ui `new-york` style, CSS variables, project aliases, semantic tokens, and `lucide-react` icons for the React frontend.
- [x] 1.4 Create the initial frontend shell with welcome, provider initialization, project selection, and task entry surfaces.

## 2. Backend Boundaries

- [x] 2.1 Create Go `domain`, `app`, and `infra` package boundaries for setup, providers, projects, tasks, confirmations, artifacts, logs, and persistence.
- [ ] 2.2 Define domain models and state transitions for setup state, provider configuration, projects, tasks, plans, task steps, tool runs, confirmations, artifacts, logs, and verification results.
- [ ] 2.3 Define app-layer use cases for setup flow, provider configuration and validation, project management, task lifecycle, confirmation decisions, artifact metadata, and log retrieval.
- [x] 2.4 Add thin Aivo bridge adapter methods with explicit request/response shapes that call app-layer use cases.
- [x] 2.5 Add frontend client/service modules that isolate generated Aivo bridge handler calls from React components.

## 3. Local Persistence

- [x] 3.1 Add GORM, `github.com/glebarez/sqlite`, and GORM-managed migrations with documented app-data-directory database and backup behavior.
- [ ] 3.2 Add initial versioned migrations for setup state, provider configurations, projects, tasks, task steps or tool runs, confirmations, artifacts, logs, verification results, and schema version tracking.
- [x] 3.3 Implement database initialization, ordered migration application, and failure handling on app startup.
- [ ] 3.4 Implement repositories for persisted MVP metadata entities with stable IDs and timestamps.
- [ ] 3.5 Implement filesystem artifact storage references while keeping large content and project source files outside SQLite.
- [ ] 3.6 Add redaction safeguards for credential references and sensitive log values before persistence or display.

## 3A. Provider Initialization

- [x] 3A.1 Implement first-run setup state that shows welcome and initialization before the workbench when setup is incomplete.
- [x] 3A.2 Implement provider option definitions for OpenAI, Claude Code, Gemini, Codex-compatible, Custom API, and compatible future adapters.
- [ ] 3A.3 Implement direct provider configuration forms plus Custom API forms with protocol, base URL, model/profile, and credential-reference handling.
- [ ] 3A.4 Implement provider validation for direct provider options and protocol-backed Custom API providers with safe error messages.
- [x] 3A.5 Persist non-secret provider metadata, readiness state, selected model or profile, and last validation timestamp.
- [x] 3A.6 Block task creation and execution until setup is complete and a provider is ready.
- [ ] 3A.7 Show remote-provider data transfer notice for direct remote providers and Custom API providers before task execution.
- [ ] 3A.8 Support Custom API validation with protocol selector, custom base URL, credential reference, selected model/profile, validation, and safe error states.

## 4. Local Project Context

- [ ] 4.1 Implement local project directory selection and backend path validation.
- [ ] 4.2 Persist project records with stable ID, display name, absolute path reference, created timestamp, and last-opened timestamp.
- [ ] 4.3 Implement recent project listing and reopen behavior.
- [ ] 4.4 Implement basic Git repository context inspection for branch and dirty working tree status.
- [ ] 4.5 Handle non-Git directories and inaccessible paths with explicit UI states and safe error messages.
- [ ] 4.6 Associate new code-to-delivery tasks with the selected project and block task creation when no project is selected.

## 5. Code Task Workflow

- [ ] 5.1 Implement task creation validation and persistence for non-empty task descriptions.
- [ ] 5.2 Implement structured task plan creation or attachment and waiting-for-approval task state.
- [ ] 5.3 Implement plan approval and decline transitions before any local side effects run.
- [ ] 5.4 Implement app-layer task step execution with observable status, timestamps, contextual errors, and artifact/log/result references.
- [ ] 5.5 Implement ordered task log recording and retrieval for the workbench timeline.
- [ ] 5.6 Implement artifact metadata creation for generated files, delivery notes, diffs, and verification results.
- [ ] 5.7 Implement task cancellation with context propagation and prevention of new step starts.
- [ ] 5.8 Implement task recovery views for waiting, running, failed, canceled, and completed tasks after app restart.

## 6. Confirmation Gates

- [ ] 6.1 Implement sensitive action classification for file writes, overwrites, deletion, process execution, credential use, external transfer, uploads, publishing, and purchases.
- [ ] 6.2 Implement pending confirmation record creation with action type, target summary, proposed effect, risk summary, task ID, and timestamp.
- [ ] 6.3 Block sensitive action execution until a matching confirmation is approved by the user.
- [ ] 6.4 Implement confirmation approval, denial, and non-actionable states in the app layer and persistence.
- [ ] 6.5 Invalidate confirmations when action details change or the task is canceled.
- [ ] 6.6 Render pending and historical confirmation decisions in the task review UI without storing secret values.

## 7. Workbench UI

- [x] 7.0 Review `ui/design.pen`, `ui/interaction-spec.md`, and exported screenshots before implementing frontend surfaces.
- [x] 7.1 Build welcome screen and setup entry state using shadcn/ui components.
- [ ] 7.2 Build provider initialization screen with direct choices for OpenAI, Claude Code, Gemini, Codex-compatible, and Custom API, including validation, success, error, and blocked states.
- [ ] 7.3 Build project selection, recent projects, project metadata, and project error states using shadcn/ui components.
- [ ] 7.4 Build task composer, plan review, approval, decline, cancel, resume, and completed-task review states.
- [ ] 7.5 Build task timeline and log surfaces with loading, empty, error, progress, and success states.
- [ ] 7.6 Build artifact, diff, verification result, and delivery note review surfaces.
- [x] 7.7 Build pending confirmation UI with clear target, proposed effect, risk summary, approve, and deny actions.
- [x] 7.8 Ensure common controls are keyboard reachable and provide meaningful labels and error text.

## 8. Tests and Verification

- [ ] 8.1 Add domain tests for setup readiness, provider state transitions, task state transitions, confirmation invalidation, and sensitive action classification.
- [ ] 8.2 Add app-layer tests for provider validation, setup gating, project validation, task creation, plan approval, execution blocking, cancellation, resume, and error mapping.
- [ ] 8.3 Add persistence tests for migrations, provider metadata, repository reads/writes, artifact references, and failed migration handling.
- [ ] 8.4 Add frontend tests for welcome/setup states, provider validation states, project empty/error/success states, task plan review, confirmation prompts, logs, artifacts, and disabled pending states.
- [x] 8.5 Run configured Go checks, frontend type checks, automated tests, and build or development verification.
- [ ] 8.6 Document any unrun checks with the exact command and reason.

## 9. MVP Handoff

- [ ] 9.0 Keep UI handoff artifacts current if implementation materially changes screen structure, state coverage, or interaction rules.
- [ ] 9.1 Update `openspec/architecture.md` if implementation introduces architecture-level decisions not already captured.
- [ ] 9.2 Update `openspec/standards.md` if concrete tool commands, package manager, lint/test conventions, or shadcn configuration become durable standards.
- [ ] 9.3 Review all first-mvp spec scenarios against implemented behavior and mark any deferred scenario explicitly before completion.
- [ ] 9.4 Prepare delivery notes summarizing the MVP workflow, verification results, known limitations, and next recommended OpenSpec changes.
