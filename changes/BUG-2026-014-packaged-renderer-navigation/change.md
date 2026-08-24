# Restore packaged renderer assets and navigation

## Problem or goal

The packaged Electron renderer loads from `file://`, but Vite emitted root-relative asset URLs and TanStack Router used browser history. Reproduction: package v0.1.1, launch the installed application, then load or navigate to a route; the renderer resolves assets or history against the application bundle root instead of `dist/index.html`, producing an empty or non-routable window. Development over HTTP works because `/assets/...` and browser history resolve against the Vite server.

## Expected behavior

`NFR-RELIABILITY-001`: packaged desktop builds must load their renderer assets relative to `index.html`, use hash history under `file://`, retain browser history for HTTP development, and fail the build if the emitted entry point contains an absolute, missing, or escaping local asset reference.

## Non-goals

No route tree, user-facing navigation structure, Electron main/preload contract, updater contract, public HTTP API, persistence schema, or browser-hosted deployment behavior changes.

## Impact

The renderer Vite base, router history selection, packaging verification script, and focused tests change. Go domain/application/persistence/transport, Electron privileges, providers, extensions, MCP/LSP, terminals/processes, worktrees, dependencies, schema, credentials, and user data are unchanged. Native tag builds on macOS, Windows, and Linux will repeat packaging and smoke gates.

## Implementation constraints

The history owner remains the renderer router. Only the `file:` protocol selects hash history; HTTP and HTTPS keep existing browser history. Bundle verification is deterministic, performs no network access, rejects path traversal and missing files, and runs after every production renderer build. Repeated builds may regenerate identical icon outputs without changing runtime state.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `RENDERER-ASSET-001` | `NFR-RELIABILITY-001` | Vite emits relative packaged asset references and the build validates them | `CT-RELIABILITY-001` | Complete |
| `RENDERER-HISTORY-001` | `NFR-RELIABILITY-001` | `file:` uses hash history while HTTP(S) development retains browser history | `CT-RELIABILITY-001` | Complete |

## Acceptance and evidence

- Focused tests cover relative assets, absolute-asset refusal, missing-asset refusal, `file:` hash history, and HTTP(S) browser history.
- `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, `pnpm build`, and `git diff --check` must pass before verification.
- The tagged `v0.1.2` workflow must repeat native packaging and smoke checks on macOS Apple Silicon, macOS Intel, Windows x64, and Linux x64.

Evidence pending final local gates.

## Security and data lifecycle

No secret, private content, prompt/tool payload, authorization header, credential, persistence, logging, clipboard, backup, or network flow is added. The verifier reads only the generated renderer entry point and its referenced files, then exits without modifying user data.

## Compatibility and migration

No schema/data, API/RPC/IPC, settings, provider, credential, or irreversible migration. The v0.1.2 package remains compatible with v0.1.1 data. Rolling back restores the packaged renderer failure but does not transform data.

## Bug root cause

The renderer retained Vite's web-server default base and TanStack Router's browser-history default even though production Electron loads `dist/index.html` from `file://`. Existing lint/build checks proved compilation but did not inspect emitted HTML paths or select history by protocol, so HTTP development masked the defect. The new `CT-RELIABILITY-001` script and router-history tests fail against the previous defaults and pass for v0.1.2.
