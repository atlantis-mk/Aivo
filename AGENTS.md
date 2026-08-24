# Aivo AI Coding Constitution

This file is the highest-priority repository rule for implementation, refactoring, testing, documentation, and release work.

## 1. Before starting work

Build the smallest sufficient context in this order:

1. When a Work ID is known, read `changes/<WORK-ID>/change.yaml` and `change.md` first. If it is listed in `changes/archive.json`, its directory is permanently read-only.
2. When no Work ID is known, read `docs/00-spec-index.md`, then search only filenames and `change.yaml` metadata to locate candidate Work. A Work Package is optional only for the direct-change cases in section 1.2.
3. Follow the current Work's `requirements`, `adrs`, `context_refs`, and `related_changes`; for combined documents, read only the matching stable-ID row or complete heading section.
4. Read `docs/09-document-governance.md` before creating or advancing Work, ADR, or Release records or changing governance. Read the applicable part of `docs/02-scope-matrix.md` for scope changes and `docs/07-test-release-plan.md` for test design, verification, or release.
5. Before product-code changes, read the complete applicable Requirement section. Before completion, check its Requirement/Test row in `docs/08-traceability.md`.

Except for section 1.2 direct changes, product-code work must identify a Work ID, Requirement ID, and Test ID. Governance changes use a Work with `type: governance` and may omit product Requirements.

### 1.1 Context routing and search boundaries

- A search hit is not a dependency. Shared IDs, module names, paths, or broad keywords do not establish a relationship between Work Packages.
- Read another Work body only when the current Work names it in `related_changes` or `supersedes`, a primary spec or ADR explicitly cites it, or a discovered conflict requires it.
- Archived, Verified, Rejected, Released, and historical Bug Work are evidence by default, not current task context.
- Search Work in two phases: exact Work/Requirement/Test IDs or paths against filenames and small `change.yaml` metadata matches, then open only candidates that answer a concrete unresolved question.
- Do not bulk-read Change bodies, recursively follow references more than one layer, or use broad full-text results as task context unless the user explicitly requests a repository-wide audit.
- Expand context one candidate at a time when ownership, security/trust, credentials, persistence/schema, public API/RPC/IPC, irreversible migration, platform boundaries, or conflicting specs are involved.

### 1.2 Work and documentation proportionality

Work is required when a product decision, contract, risk, migration, or cross-task coordination must be preserved—not because of the diff size or whether a change is UI, bug, or refactoring.

A direct change is allowed only when it stays within accepted behavior and scope, is local and easily reversible, introduces no product or architecture choice, crosses no security/trust, secret, data ownership, persistence/schema, public API/RPC/IPC, compatibility/migration, platform/scope, or release boundary, and can be fully implemented and verified in the current task. Direct changes may add or update focused regression tests, fixtures, and snapshots that prove existing expectations.

Typical direct changes include copy or visual polish, restoring existing responsive/accessibility behavior, ordinary bugs with a clear local root cause, internal refactoring, type/null fixes, test strengthening, non-release developer tooling, and local semantics-preserving performance improvements. These are examples, not a whitelist.

Create or reuse a Work when any of the following applies:

- A primary spec, Requirement, Scope, ADR, product decision, or security decision changes.
- Work crosses a security/trust, secret, data-ownership, persistence/schema, public API/RPC/IPC, compatibility/migration, platform/scope, or release/rollback boundary.
- A production dependency or license, irreversible operation, or cross-module/platform/version coordination is involved.
- A bug is severe, recurring, security/data-loss related, has an unclear root cause, or needs a durable remediation record.
- The task cannot be fully verified now or must preserve plans, tradeoffs, risk, or incomplete state for another agent.

Low-risk Work still uses the common YAML, routing, state machine, and gates, but each body section may be one short paragraph or explicit N/A. Never use a long-lived catch-all Work for unrelated changes.

## 2. Authority order

Resolve conflicts in this order:

1. `AGENTS.md`
2. accepted Scope Matrix
3. Functional Requirements
4. focused specs
5. accepted ADRs
6. Architecture, Data Model, and Security specs
7. accepted Change increments not yet merged into primary specs
8. current code and tests

Do not assume code is correct when it conflicts with a spec. Determine whether the implementation or specification is wrong. Changes to accepted behavior require a Work and synchronized scope, Requirement, test, and Traceability updates. Code owns exhaustive public enum, schema, and signature listings; documents point to their implementation locations instead of becoming a second owner.

## 3. Product and architecture boundaries

- Aivo is a local-first AI agent desktop application. The current product shell is Electron with a Vite/React renderer and a local Go runtime.
- The renderer reaches privileged filesystem, process, credential, persistence, and OS capabilities only through typed preload/desktop services and versioned local core contracts.
- Go domain and application layers own business behavior; HTTP, SQLite/GORM, model providers, MCP, LSP, processes, and OS integrations remain adapters.
- New v2 HTTP contracts use resource-oriented, versionable shapes and a unified actionable error convention.
- Cloud sync, collaboration, accounts, telemetry, mobile clients, and a restored built-in browser are not current Required scope unless an accepted Work changes `docs/02-scope-matrix.md`.
- Open product decisions use stable `OPEN-*` IDs. Agents must not close decisions that change product, data, security, or platform boundaries without explicit approval.

## 4. Security, data, and implementation rules

- Never commit API keys, refresh tokens, credential stores, local databases, user prompts, private repositories, auth sessions, signing material, or unsanitized migration fixtures.
- Do not log secrets, authorization headers, raw prompt/tool payloads, provider responses containing user data, or sensitive filesystem contents. Structured events must use safe summaries and operation IDs.
- Persistence migrations require an explicit version transition, pre-migration backup, transaction/rollback behavior, previous-version sanitized fixtures, and failure recovery tests.
- Goroutines, streams, terminals, provider calls, MCP/LSP clients, and child processes require clear ownership, cancellation, bounded output/backpressure, and deterministic teardown.
- Implement the smallest verifiable vertical slice across domain, persistence, transport, preload, renderer, failure states, tests, and migration impact. Do not claim completion from mocks, TODOs, static demo data, or compilation alone.
- Platform signing and installer verification must run on the target OS; cross-compilation does not replace release acceptance.

## 5. Work, ADR, verification, and release

- Work states are `Draft -> Accepted -> Implementing -> Verified`; rejected work ends at `Rejected`. `Released` is supported only for compatible historical records.
- New behavior may change product code only after `Accepted`. Merge accepted behavior into primary specs and Traceability before moving to `Implementing`.
- `Verified`, `Rejected`, and historical `Released` Work must be sealed with `pnpm work:archive -- <WORK-ID>` in the same task. Sealed directories and existing archive entries are permanently read-only; corrections require a new Work.
- Significant decisions about persistence ownership/schema migration, privilege boundaries, public transport contracts, provider credential ownership, plugin/MCP trust, sandbox/command authorization, platform scope, or irreversible migration require a new or revised ADR.
- Release records are created from `releases/_template.md`, reference only sealed Work, and pair with a same-name Git tag. Releases do not rewrite Requirements or sealed Work.
- `pnpm docs:check` validates documentation, routing, traceability, and archive integrity. The applicable code gates remain `pnpm test:core`, `pnpm lint`, and `pnpm build`.

# Repository Guidelines

## Project Structure & Module Organization

Aivo is an Electron desktop app with a Vite/React renderer and a local Go agent runtime. Desktop code lives in `apps/desktop`: `electron/` contains the main and preload processes, `src/routes` defines TanStack Router screens, `src/features` holds feature UI, `src/components/ui` contains shared shadcn-style primitives, and `bridge/` contains generated desktop-to-core bindings. Go runtime code lives in `core`, with domain types in `core/domain`, application services in `core/app`, persistence in `core/infra/persistence`, HTTP transport in `core/internal/transport/http`, and the CLI entry point in `core/cmd/aivo-core`. Current primary specifications live in `docs`; focused behavior belongs in `specs`; decisions in `adr`; incremental Work in `changes`; release facts in `releases`. `openspec/changes/aivo-v2` remains preparation and migration evidence.

## Build, Test, and Development Commands

- `pnpm dev`: starts the Go core, waits for `http://127.0.0.1:43117/health`, then launches the Electron/Vite desktop app.
- `pnpm build`: type-checks and builds the desktop workspace.
- `pnpm lint`: runs `oxlint` for the desktop TypeScript/React code.
- `pnpm dev:core`: runs only the Go core via `go run ./cmd/aivo-core`.
- `pnpm test:core`: runs all Go tests with `go test ./...`.
- `cd core && go run ./cmd/aivo-core provider-smoke --provider <name>`: runs provider backend diagnostics.

## Coding Style & Naming Conventions

Format Go with `gofmt`; keep tests beside implementation files as `*_test.go` with `Test...` functions. In TypeScript, follow the existing React function component style, two-space indentation, and kebab-case filenames such as `provider-connection-dialogs.tsx`. Use shared UI primitives from `apps/desktop/src/components/ui`, but do not modify anything inside that directory. Put app-specific wrappers or composition code outside `components/ui`, and use utilities from `apps/desktop/src/lib`. Do not hand-edit generated files such as `apps/desktop/src/routeTree.gen.ts` or `apps/desktop/bridge/go/*`.

## Backend Development Standards

Design backend APIs with clear RESTful resource boundaries, stable request/response contracts, explicit validation rules, and versionable shapes that can evolve without breaking callers. Keep the Go project directory structure clear and extensible, preserving domain, application, infrastructure, transport, and command boundaries instead of mixing business logic with adapters. Use a unified error handling and response convention so validation failures, permission errors, missing resources, conflicts, internal failures, and external dependency errors are distinguishable and actionable.

Use goroutines and channels only with clear ownership, cancellation, lifecycle, and backpressure behavior; protect shared state with appropriate synchronization and avoid data races. Keep database access explicit and reviewable: use transactions for multi-step consistency, handle rollbacks correctly, avoid N+1 query patterns, add indexes where query paths require them, and keep query behavior observable enough to optimize. Add structured logging, traceable operation context, and service monitoring hooks around important backend flows without logging secrets or sensitive user data.

Manage configuration through typed config loading, environment-specific defaults, validation at startup, and secure handling of secrets or safety-critical parameters. Treat performance, code reuse, and long-term maintainability as backend design constraints: prefer simple composable services, bounded interfaces, measurable optimizations, focused packages, and tests around concurrency, persistence, error paths, and API contracts.

## Frontend Layout Standards

Design layouts responsively from the start, with content-driven breakpoints and adaptations rather than fixed desktop assumptions. Use the project spacing system consistently, keep visual alignment relationships clear, and choose positioning methods deliberately: prefer normal document flow, flex, and grid before absolute positioning, reserving fixed or absolute placement for cases that require it. Control heights flexibly with min/max constraints, intrinsic sizing, and overflow behavior so content can expand, shrink, wrap, or scroll without breaking the interface. Handle empty, loading, error, long-content, narrow-viewport, and other boundary states as part of the layout work. Keep layout code maintainable by using shared primitives, semantic tokens, small composition components, and readable class organization instead of one-off spacing, sizing, or positioning hacks.

## Testing Guidelines

Backend coverage is Go-first: add focused unit tests near changed packages and run `pnpm test:core` before submitting. Frontend currently has lint/build verification but no dedicated test runner; for UI changes, run `pnpm lint` and `pnpm build`, and include screenshots when behavior or layout changes.

## Commit & Pull Request Guidelines

Use clear imperative commits such as `Add session persistence tests`; conventional prefixes like `feat:` or `fix:` are acceptable when useful. Pull requests should include a concise summary, tests run, linked Work or issue IDs, screenshots for UI changes, and notes about provider configuration or auth flows needed for verification.

## Security & Configuration Tips

Do not commit API keys, refresh tokens, local auth stores, or generated secrets. Prefer environment variables such as `OPENAI_API_KEY` for provider credentials, and document any required local configuration in the relevant Work or PR.
