# Aivo

Aivo is an AI agent desktop application. The desktop shell is Electron with a Vite React renderer, and the local agent runtime is written in Go.

## Layout

```text
apps/desktop   Electron main/preload plus Vite React renderer
core           Go agent runtime and local API
scripts        Build, development, release smoke, and packaging automation
build          Packaging assets placeholders
docs           Current product, architecture, data, security, test, and traceability specs
specs          Focused cross-module specifications
adr            Architecture decision records
changes        Routed Work Packages and immutable completion manifest
releases       Delivered-version records
```

Start documentation work at [docs/00-spec-index.md](docs/00-spec-index.md). Governance rules, Work thresholds, templates, and archive behavior are defined in [docs/09-document-governance.md](docs/09-document-governance.md).

## Development

```bash
cd /Users/atlan/Documents/Aivo
pnpm dev
```

This starts the Go core first, waits for `http://127.0.0.1:43117/health`, then starts the Electron/Vite desktop app.

```bash
cd core
go run ./cmd/aivo-core
```

Provider backend diagnostics:

```bash
cd core
go run ./cmd/aivo-core provider-smoke --provider <configured-provider>
```

See [docs/provider-backend.md](docs/provider-backend.md) for provider registry, auth, runtime policy, diagnostics, and smoke-check details.

Release packaging and signing checks are documented in [docs/release-quality.md](docs/release-quality.md).

Documentation and governance checks:

```bash
pnpm docs:check
pnpm scripts:test
```
