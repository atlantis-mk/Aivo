# Repository Guidelines

## Project Structure & Module Organization

Aivo is a Electron desktop app with Go backend code at the repository root and a Vite/React frontend in `frontend/`. Core Go entry points live in `main.go`, `app.go`, and services such as `provider_catalog.go`; Go tests sit beside the code, for example `provider_catalog_test.go`. Frontend source is under `frontend/src`, with routes in `routes`, screens in `components`, shared utilities in `lib`, and generated Aivo bridge handlers in `apps/desktop/bridge`. Build metadata lives in `Electron metadata`, packaging files are under `build/`, and OpenSpec artifacts are in `openspec/changes`. The `opencode/` directory is the local "Open Code" reference project: use it to study existing implementations and interaction patterns when building similar features here.

## Build, Test, and Development Commands

- `npm run dev`: run the desktop app in live development mode with Vite hot reload.
- `npm run build`: create a production desktop build using the configured frontend build.
- `go test ./...`: run all Go tests.
- `cd frontend && npm run dev`: run only the Vite frontend for browser-focused UI work.
- `cd frontend && npm run build`: type-check and build the frontend with `tsc -b` and Vite.
- `cd frontend && npm run lint`: run ESLint over TypeScript and React files.

Run `npm install` in `frontend/` first if dependencies are missing.

## Coding Style & Naming Conventions

Format Go with `gofmt`; use formatter-produced tabs and tests named `Test...`. Keep backend JSON contracts explicit with struct tags matching frontend casing, such as `providerId`. Frontend files use TypeScript, React function components, double-quoted imports, and two-space indentation. Prefer shadcn-style primitives in `frontend/src/components/ui` and helpers in `frontend/src/lib`. Treat `opencode/` as read-only reference material unless a task explicitly asks for changes there. Do not hand-edit generated files such as `frontend/src/routeTree.gen.ts` or `apps/desktop/bridge`.

## Testing Guidelines

Add Go unit tests beside backend changes and run `go test ./...` before submitting. For frontend changes, run `npm run lint` and `npm run build`; add component or route tests only if a test framework is introduced. Cover provider catalog, auth, persistence, and Aivo bridge behavior with focused cases.

## Commit & Pull Request Guidelines

This repository currently has no commit history, so use clear imperative commits such as `Add provider auth validation` or conventional prefixes like `feat:` and `fix:`. Pull requests should include a summary, tests run, linked issues or OpenSpec change IDs, and screenshots for visible UI changes. Note configuration needed to verify auth flows.

## Security & Configuration Tips

Do not commit API keys, refresh tokens, or local auth store files. Prefer environment variables such as `OPENAI_API_KEY` for provider credentials. Document experimental browser auth settings in the relevant OpenSpec change or PR.
