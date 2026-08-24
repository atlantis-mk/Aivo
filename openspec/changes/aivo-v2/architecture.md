# Aivo v2 architecture guardrails

## Boundary model

```text
React routes/features
        |
Electron preload and typed desktop services
        |
Versioned local HTTP contracts and event streams
        |
Go application services
        |
Domain models and ports
        |
SQLite / providers / OS processes / MCP / LSP adapters
```

## Rules

### Renderer

- Routes compose feature modules; feature state must not be embedded into the
  generated route tree.
- Shared UI primitives under `apps/desktop/src/components/ui` are not modified.
- Privileged filesystem, process, credential, and persistence operations go
  through typed services exposed by preload.
- Layouts use normal flow, flex, and grid with explicit overflow behavior before
  fixed or absolute positioning.

### Electron boundary

- Keep preload APIs minimal, typed, and capability-oriented.
- Validate all renderer-provided paths and arguments in the privileged layer.
- Every process or stream has explicit ownership, cancellation, and teardown.

### Go core

- Domain types do not depend on HTTP, GORM, Electron, or provider SDK shapes.
- Application services own use-case orchestration and transaction boundaries.
- Persistence and external runtimes remain adapters behind focused interfaces.
- Goroutines have a parent context, a clear owner, bounded output/backpressure,
  and a deterministic shutdown path.

### Contracts and errors

- New endpoints use a versioned `/api/v2` namespace and resource-oriented
  request/response shapes. Existing RPC methods may remain as a compatibility
  adapter during migration.
- Contracts distinguish validation, unauthorized/forbidden, not found,
  conflict, dependency unavailable, cancelled, and internal failures.
- Error responses include a stable machine code, safe user-facing message,
  optional field details, and a traceable operation identifier.
- Secrets and sensitive prompt/tool content are excluded from logs by default.

### Persistence

- Every schema transition has a numeric version, forward migration, backup
  requirement, fixtures from the previous version, and failure tests.
- Multi-step writes use explicit transactions and rollback checks.
- Query paths used by session timelines, project lists, and tool activity are
  reviewed for pagination and indexes.

## Compatibility policy

During a slice migration, the new UI may use a v2 contract while an adapter
maps it to stable application services. Compatibility code must have an owner,
removal condition, and test. A v1 field or table is not removed until the data
migration and rollback checks for its replacement pass.

## Observability minimum

Important operations record structured start/completion/failure events with an
operation ID, duration, cancellation reason, and safe dependency classification.
No API key, refresh token, authorization header, or raw secret value may appear
in logs or diagnostics exports.
