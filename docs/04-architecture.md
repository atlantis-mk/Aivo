# Aivo architecture

## Boundary model

```text
React routes and feature modules
        |
Electron preload and typed desktop services
        |
Versioned local HTTP contracts and event streams
        |
Go application services
        |
Domain models and ports
        |
SQLite, providers, OS processes, MCP, LSP, and other adapters
```

## Ownership and dependency rules

### Renderer

- Routes compose feature modules; generated route files do not own feature behavior.
- Shared primitives under `apps/desktop/src/components/ui` are read-only.
- Filesystem, process, credential, persistence, and OS access goes through typed preload services.
- Layout uses document flow, flex, and grid with explicit overflow before fixed or absolute positioning.

### Electron boundary

- Main and preload APIs are minimal, typed, capability-oriented, and validate renderer-provided paths and arguments.
- Each process, stream, window, and subscription has explicit ownership, cancellation, and teardown.
- Generated bindings in `apps/desktop/bridge/go` are generated from their owner and never hand-edited.

### Go core

- `core/domain` does not depend on HTTP, GORM, Electron, or provider SDK shapes.
- `core/app` owns use-case orchestration, authorization decisions, and transaction boundaries.
- `core/infra/persistence`, provider clients, MCP/LSP, process integration, and HTTP transport are adapters behind focused interfaces.
- Goroutines inherit a parent context and have bounded output/backpressure and deterministic shutdown.

## Contracts and errors

- New public resources use versioned `/api/v2` contracts. Existing RPC methods may remain as tested compatibility adapters during migration.
- Stable errors distinguish validation, unauthorized/forbidden, not found, conflict, dependency unavailable, cancelled, and internal failure.
- Error responses include a machine code, safe user message, optional field details, and operation ID.
- Exhaustive request/response shapes are owned by code; specs own behavior and compatibility rules.

## Observability

Important operations record structured lifecycle events with operation ID, duration, cancellation reason, and safe dependency classification. API keys, refresh tokens, authorization headers, raw secrets, and sensitive raw prompt/tool payloads never enter logs or diagnostic exports.

## Compatibility

During a v2 slice, a compatibility adapter may connect new UI contracts to stable application services. Every adapter needs an owner, removal condition, and regression test. A v1 contract field or persistence object is not removed until migration and rollback evidence for its replacement passes.
