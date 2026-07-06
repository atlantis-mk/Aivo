# Repository Guidelines

## Project Structure & Module Organization

Aivo is an Electron desktop app with a Vite/React renderer and a local Go agent runtime. Desktop code lives in `apps/desktop`: `electron/` contains the main and preload processes, `src/routes` defines TanStack Router screens, `src/features` holds feature UI, `src/components/ui` contains shared shadcn-style primitives, and `bridge/` contains generated desktop-to-core bindings. Go runtime code lives in `core`, with domain types in `core/domain`, application services in `core/app`, persistence in `core/infra/persistence`, HTTP transport in `core/internal/transport/http`, and the CLI entry point in `core/cmd/aivo-core`. Product, architecture, and change specs live under `openspec`; supporting docs are in `docs`; automation scripts are in `scripts`.

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

This checkout has no Git history available, so use clear imperative commits such as `Add session persistence tests`; conventional prefixes like `feat:` or `fix:` are acceptable when useful. Pull requests should include a concise summary, tests run, linked OpenSpec change or issue IDs, screenshots for UI changes, and notes about provider configuration or auth flows needed for verification.

## Security & Configuration Tips

Do not commit API keys, refresh tokens, local auth stores, or generated secrets. Prefer environment variables such as `OPENAI_API_KEY` for provider credentials, and document any required local configuration in the relevant OpenSpec change or PR.
