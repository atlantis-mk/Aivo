## Standards Scope

These standards apply to all Aivo implementation work unless a later OpenSpec architecture decision explicitly supersedes them.

Aivo is a local-first desktop AI productivity assistant. The first implementation phase focuses on the code-to-delivery workflow, reviewable autonomy, local project access, and explicit confirmation for sensitive actions.

## Stack Summary

The application code has not been initialized yet. These standards record the chosen default stack and must be reconciled with generated Electron manifests when source is initialized.

| Area | Stack / Tooling | Notes |
| --- | --- | --- |
| Desktop | Electron + Go | Use the React/TypeScript/Vite template and bundle ID `com.aivo.desktop`. |
| Package management | pnpm | Use pnpm for frontend dependency installation, scripts, and shadcn CLI execution. |
| Frontend | React + TypeScript + Vite + TanStack Router + Tailwind CSS v4 + shadcn/ui | React owns the workbench UI and user-facing state; TanStack Router owns client-side navigation; Tailwind v4 powers shadcn tokens and utilities; shadcn/ui is the default component system. |
| Backend / Local Services | Go module `aivo` | Use `domain / app / infra` boundaries by default. |
| Database / Persistence | SQLite + GORM (`github.com/glebarez/sqlite`) | Use GORM models, `AutoMigrate`, and `Migrator` helpers before persistence-dependent features ship. |
| Testing | Go test + Vitest + Testing Library | Required gates are type checks and relevant behavior-first tests. |

## Universal Coding Standards

- Keep product behavior aligned with `openspec/product.md`.
- Keep implementation aligned with `openspec/architecture.md` once that file exists.
- Prefer small, cohesive modules over large mixed-responsibility files.
- Keep business rules in one place; do not duplicate logic across frontend, app, and infrastructure layers.
- Make sensitive actions explicit and user-confirmed, including credential use, publishing, deletion, overwrites, purchases, and external data transfer.
- Handle errors explicitly with useful context; do not silently swallow failures.
- Do not introduce new dependencies without a documented reason; prefer the standard library and existing project dependencies.
- Keep configuration out of hardcoded implementation paths when it affects environments, providers, secrets, or user-controlled behavior.

## Go Backend and Local Service Standards

- Use `domain / app / infra` as the default layering model.
- Put core models, business rules, invariants, and pure decision logic in `domain`.
- Keep `domain` independent from Electron, React, filesystems, processes, databases, network APIs, model providers, and UI concerns.
- Put use-case orchestration, task flows, transaction boundaries, permission checks, and sensitive-action confirmation flows in `app`.
- Put filesystems, process execution, model/API clients, persistence, OS integration, and Aivo bridge adapters in `infra`.
- Define interfaces at the consumer side when they clarify testability or boundary ownership; avoid premature interfaces for single concrete implementations.
- Pass `context.Context` through operations that perform I/O, call external services, execute processes, or may be cancelled.
- Wrap cross-boundary errors with enough context for diagnosis while preserving the original error for `errors.Is` / `errors.As` where relevant.
- Do not use panics for ordinary errors. Reserve panics for impossible programmer errors or process startup failures where recovery is not meaningful.
- Keep goroutine lifetimes owned and cancellable. Do not start background work without a clear shutdown path.
- Keep package APIs narrow. Export only symbols intended for cross-package use.

## Electron Desktop Boundary Standards

- Treat Aivo bridge handlers as an adapter boundary, not as the place for business rules.
- Aivo bridge methods should validate boundary inputs, call `app` use cases, and convert results or errors into UI-safe response shapes.
- Keep local filesystem, shell/process, credential, and external network access behind explicit Go services with confirmation gates where product principles require them.
- Do not let React components call many unrelated Aivo bridge handlers directly. Provide a typed frontend client/service layer for Aivo bridge calls.
- Keep long-running work observable by the UI through task state, logs, progress events, or result artifacts as appropriate.
- Preserve local-first behavior by default. Any remote model, API, upload, publish, or sync path must be visible to the user and controlled by explicit settings or confirmations.

## React and TypeScript Frontend Standards

- Use TypeScript for frontend code and keep important boundary data typed.
- Use TanStack Router for client-side navigation. Route modules should define typed path/search params, layout composition, and route-level pending/error states.
- Use shadcn/ui as the default UI component system. Prefer installed shadcn components or registry components before creating custom primitives.
- Use Tailwind CSS v4 through `@tailwindcss/vite`; keep semantic CSS variables in `frontend/src/style.css`.
- Configure shadcn/ui with the `new-york` style, CSS variables, `lucide-react` icons, and aliases for `@/components`, `@/features`, `@/lib`, `@/routes`, and `@/services` unless generated Electron paths require a documented adjustment.
- Compose shadcn components using their documented structure and variants. Do not bypass required subcomponents for dialogs, sheets, drawers, tabs, selects, cards, forms, menus, or other composed controls.
- Use project shadcn aliases, design tokens, semantic colors, configured icon library, and component variants. Avoid raw color utilities, parallel styling systems, or hardcoded import aliases when a shadcn/project token or alias exists.
- Keep presentational components focused on rendering and interaction. Move workflow orchestration, data loading, and Aivo bridge calls into hooks, clients, or feature services.
- Keep route modules thin. Do not put task lifecycle rules, confirmation policy, provider invocation, filesystem/process behavior, or raw Aivo bridge calls inside routes.
- Keep route modules under frontend `src/routes`; keep feature UI, hooks, and view-model mapping under `src/features/*`; keep typed Electron access under `src/services`.
- Do not introduce TanStack Query or another server-state cache in the MVP unless repeated caching, invalidation, or background refresh requirements are documented.
- Keep component state local unless the workflow requires sharing across routes, task surfaces, or artifact review views.
- Keep Aivo API access behind a clear client/service boundary; do not scatter generated binding calls throughout arbitrary components.
- User-facing flows must handle loading, empty, error, and success states.
- Build interfaces for reviewable work: plans, diffs, logs, generated artifacts, and verification results should be inspectable when relevant.
- Preserve accessibility basics: semantic controls, labels, keyboard reachability, focus handling, and meaningful error text.
- When adding or updating shadcn components, use the shadcn CLI/docs for the current project configuration and review generated source before relying on it.
- Avoid UI-only duplication of backend business rules. Frontend validation may improve user experience, but Go boundary validation remains authoritative.

## Data and API Standards

- Keep request and response shapes explicit at process, Electron, model-provider, persistence, and filesystem boundaries.
- Validate untrusted input at system boundaries, including user-provided paths, command arguments, imported files, external API responses, and model output.
- Use stable identifiers for persisted or long-running entities such as tasks, artifacts, projects, and tool runs.
- Make side effects idempotent where retries are expected, especially for file writes, job execution, sync, and external calls.
- When persistence is introduced, use GORM-managed SQLite connections with model-based migrations and targeted `Migrator` compatibility steps. Store the SQLite database in the Electron app data directory, create a pre-migration backup once user data exists, and stop persistence-dependent startup on migration failure.
- Validate CLI-backed providers with command availability, version output where supported, and non-secret readiness checks where available. Do not scrape or log secrets while checking provider readiness.
- Do not store secrets, credentials, or personal data in logs, generated artifacts, snapshots, or config files unless explicitly required and protected.

## Testing Standards

- Use behavior-first tests: verify externally meaningful behavior, spec scenarios, and failure paths instead of implementation details.
- Add or update tests for every changed OpenSpec scenario that can be automated.
- Cover relevant non-happy paths: validation failures, empty states, permission denial, cancellation, retries, unavailable tools, and external service errors.
- Keep unit tests close to domain and app behavior. Use integration tests for Electron, filesystem, process, persistence, or provider boundaries when behavior depends on real adapters.
- Use Go's standard `go test ./...` for backend tests. Use Vitest and Testing Library for frontend component, hook, route-state, and service-boundary tests.
- Keep fixtures readable, deterministic, and minimal.
- Required quality gates are type checks and relevant tests. Lint and build commands should be run when configured or when the change risk justifies them.
- Do not mark implementation complete until relevant tests pass, or unrun tests are explicitly reported with the reason.

## Definition of Done

A task is done only when:

- Implementation satisfies the relevant OpenSpec scenarios and product principles.
- Sensitive actions and destructive operations require explicit user confirmation.
- Relevant type checks and tests pass, or unrun commands are explicitly reported with reasons.
- Configured lint/build checks are run when available and relevant to the change.
- Code follows `openspec/architecture.md` once available and this standards document.
- Errors, empty states, cancellation, and edge cases relevant to the scenario are handled.
- No obvious duplicate code, dead code, or temporary workaround remains unmarked.
- Documentation, tasks, or OpenSpec artifacts are updated when the change alters conventions, public behavior, or engineering policy.

## Review Checklist

- Does this remain aligned with `openspec/product.md` and any existing `openspec/architecture.md`?
- Are Go `domain / app / infra` boundaries preserved?
- Are Aivo bridge handlers thin adapters rather than business-logic containers?
- Are React components separated from workflow orchestration and Electron client calls?
- Does frontend UI use shadcn/ui components, variants, semantic tokens, and project aliases before custom markup or styling?
- Are route modules thin, typed, and free of workflow business rules or raw Aivo bridge calls?
- Are frontend files placed in the expected route, feature, service, component, or lib boundary?
- Are sensitive local or external actions explicitly confirmed?
- Are important inputs validated at the right boundary?
- Are errors explicit, contextual, and not swallowed?
- Does the change introduce a dependency, global state, or cross-layer call without rationale?
- Are tests behavior-focused and tied to user-visible or spec-defined scenarios?
- Are unverified scenarios and unrun checks reported clearly?

## Banned Anti-Patterns

- Business logic inside Aivo bridge handler methods or React presentational components.
- Business logic, raw Aivo bridge calls, or sensitive side-effect policy inside frontend route modules.
- Cross-layer calls that bypass `domain / app / infra` boundaries.
- Silent `catch` blocks, ignored Go errors, or swallowed promise rejections.
- Broad `any`, untyped JSON, or map-shaped data at important TypeScript or Go boundaries when a concrete type is practical.
- Hardcoded credentials, model providers, absolute paths, or environment-specific configuration.
- Shell/process execution from UI paths without a Go service boundary and confirmation policy.
- Long-running goroutines or frontend async tasks without cancellation or ownership.
- Duplicate implementations of the same business rule in frontend and backend.
- New dependencies without documented rationale.
- Custom UI primitives, raw color styling, or one-off component systems where a shadcn/ui component or token should be used.
- Large mixed-purpose components, services, packages, or utility modules.

## Tooling Commands

Initialize scripts and documentation around these command names unless generated Electron tooling requires a documented adjustment.

- Install: `pnpm install` in the frontend workspace after Electron initialization.
- Local desktop development: `pnpm dev`.
- Go type/build check: `go test ./...` is the required backend correctness gate; add `go test` package filters only when justified by scope.
- Frontend typecheck: `pnpm typecheck`.
- Frontend test: `pnpm test`.
- Full test: `go test ./...` plus `pnpm test`.
- Desktop build verification: `pnpm build` when packaging or desktop integration changes are in scope.
- Lint: optional until configured; once configured, expose it as `pnpm lint` and document when it is required.

## Needs Confirmation

- Application source has not been initialized yet; generated Electron directory names and binding output paths must be reconciled after initialization.
- Final app icon remains pending.
- Lint is not a required default completion gate until configured, but should be run when available and relevant.
