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

### Agent tools and extensions

- The default coding Agent contract consists only of `read`, `bash`, `edit`, and `write`, bound to one coherent Execution Environment.
- Every executable tool has one canonical ASCII name matching `^[A-Za-z0-9_-]+$` with a 64-byte maximum. Registry and Manifest validation reject invalid names before catalog or snapshot assembly; Provider adapters use the canonical name unchanged and own no wire alias codec.
- Before a provider request, the Host prepares eligible Skill, plugin, Manifest v1 extension, and MCP catalogs, injects canonical summaries for selected Skills, materializes full instructions only for an exact validated selected Skill subset, injects selected bounded context resources, and freezes the exact registration and schema identity of every activated tool; execution rejects calls that do not match that Tool Snapshot.
- Long-tail tools, policies, environments, context resources, and Web views enter through versioned language-neutral extension manifests and protocols. Discovery and trust do not execute extension code.
- Required Host-owned Agent capabilities may ship as trusted built-in Manifest v1 extensions. The `aivo.projects` built-in contributes only `_`-namespaced, mode-default project query, registration, and immutable current-session association tools; it does not expand the four unqualified primitive registry.
- The Host auxiliary resolver may select only bounded eligible tool, Skill, and context registrations plus the selected Skill subset requiring instructions before the primary model call. It cannot author injected summaries, install/import, grant trust, enable a source, bind credentials, grant authority, or execute. Automatic summary/instruction/context selections are request-scoped.
- Extension processes, services, streams, calls, views, and artifacts have explicit owners, bounded queues, cancellation, draining, and deterministic teardown.
- Isolated extension Web content reaches privileged behavior only through a versioned Host bridge and validated manifest-declared actions.

## Contracts and errors

- New public resources use versioned `/api/v2` contracts. Existing RPC methods may remain as tested compatibility adapters during migration.
- Stable errors distinguish validation, unauthorized/forbidden, not found, conflict, dependency unavailable, cancelled, and internal failure.
- Error responses include a machine code, safe user message, optional field details, and operation ID.
- Exhaustive request/response shapes are owned by code; specs own behavior and compatibility rules.

## Observability

Important operations record structured lifecycle events with operation ID, duration, cancellation reason, and safe dependency classification. API keys, refresh tokens, authorization headers, raw secrets, and sensitive raw prompt/tool payloads never enter logs or diagnostic exports.

## Compatibility

During a v2 slice, a compatibility adapter may connect new UI contracts to stable application services. Every adapter needs an owner, removal condition, and regression test. A v1 contract field or persistence object is not removed until migration and rollback evidence for its replacement passes.
