## Why

Aivo needs a first runnable MVP that turns the product baseline into a usable local desktop workbench. The MVP should prove the code-to-delivery workflow with reviewable task execution, local project context, and explicit user control before expanding into data, document, video, or collaboration capabilities.

## What Changes

- Initialize the Electron + Go + React/TypeScript desktop application foundation in the existing single repository.
- Add a first-run welcome and initialization experience that explains the local-first workflow, configures an AI/provider adapter, then moves the user into project selection.
- Add provider configuration with direct choices for OpenAI, Claude Code, Gemini, and Codex-compatible, plus a Custom API option for advanced endpoints and future compatible adapters, without storing raw secrets in SQLite or logs.
- Add a workbench experience for selecting or opening a local project and viewing its basic repository context.
- Add a task execution workflow that lets a user describe a code-to-delivery task, review a plan, run approved local steps, observe logs/progress, and inspect generated artifacts.
- Add confirmation gates for sensitive or high-impact actions in the MVP workflow, including file writes, command execution, deletion, overwrites, credential use, and external data transfer.
- Add local SQLite-backed metadata for projects, tasks, tool runs, confirmations, artifacts, logs, and resumable task state, while keeping source files and large artifacts on the filesystem.
- Add a typed frontend service boundary for Aivo bridge calls and shadcn/ui-based workbench surfaces for projects, task timeline, logs, diffs, artifacts, confirmations, and verification results.
- Add behavior-first tests and configured quality gates for the MVP workflow where project tooling supports them.

## Capabilities

### New Capabilities

- `desktop-app-foundation`: Electron desktop shell, Go/React project structure, typed bridge boundaries, shadcn/ui baseline, and local configuration needed to run the app.
- `provider-configuration`: First-run provider setup, direct provider selection, Custom API endpoint setup, protocol selection for custom endpoints, connection validation, credential-reference handling, and app-ready state.
- `local-project-context`: Open and persist local project references, inspect basic repository metadata, and make selected project context available to task workflows.
- `code-task-workflow`: Create, plan, run, cancel, resume, and review code-to-delivery tasks with observable steps, logs, tool runs, artifacts, verification results, and delivery notes.
- `confirmation-gates`: Require explicit user confirmation before sensitive local or external actions proceed.
- `local-persistence`: SQLite-backed metadata and migration foundation for projects, tasks, tool runs, confirmations, artifacts, logs, and resumable state.

### Modified Capabilities

- None.

## Impact

- Affected code: new Electron app source, Go `domain / app / infra` packages, provider adapter interfaces, React + TypeScript onboarding/workbench UI, typed frontend client/services, shadcn/ui components, database migrations, and test scaffolding.
- APIs and boundaries: Aivo bridge methods expose provider setup, project, task, confirmation, artifact, and log use cases through typed request/response shapes; React components use frontend services instead of scattered raw bindings.
- Dependencies: Electron, Go module `aivo`, pnpm, React, TypeScript, Vite, TanStack Router, shadcn/ui, GORM, `github.com/glebarez/sqlite`, GORM-managed migrations, Vitest, and Testing Library.
- Systems: local filesystem, local process execution, local SQLite database, optional model/provider adapters behind app-layer policy, and generated artifacts stored on disk with metadata references.
