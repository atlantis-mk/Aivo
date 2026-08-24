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

- Core owns a validated prompt registry composed from embedded Markdown defaults and private global managed overrides. Each consumer resolves a stable prompt ID through one immutable execution snapshot. Managed files carry content only; code owns category contracts, variables, required/disable/delete policy, structured wrappers, parsers, permissions, and capability enforcement. The associated-subagent explanation is code-generated from the effective validated allowlist rather than resolved from a managed prompt ID. Invalid working files remain inactive and expose only safe diagnostics. Agent-mode persistence references prompt IDs after schema v9, while existing project runtime files remain the final project-scoped overlay.
- Core composes the effective Agent-mode catalog from code-owned built-in defaults, schema-v8 global definitions, and project runtime configuration in that precedence order. Management RPCs may create user modes or replace/reset visible built-ins, but their DTOs and stored payloads exclude toolsets; built-ins reattach code-owned capabilities and custom global modes receive only the safe default before an optional project policy overlay. Primary-capable definitions may own a bounded subagent-ID allowlist. Core validates its effective references, omits delegation when empty, generates narrowed Provider prompt context and tool schema when non-empty, and revalidates each target before forking through the existing bounded child-session executor. Hidden internal workers remain protected and session/association references are checked before custom deletion or role changes. The renderer presents eligible associations through a multi-select dropdown and submits typed definitions, but never owns prompt, capability, permission, or delegation truth.
- The default coding Agent contract consists only of `read`, `bash`, `edit`, and `write`, bound to one coherent Execution Environment.
- Every executable tool has one canonical ASCII name matching `^[A-Za-z0-9_-]+$` with a 64-byte maximum. Registry and Manifest validation reject invalid names before catalog or snapshot assembly; Provider adapters use the canonical name unchanged and own no wire alias codec.
- Core's global tool-preference store owns a bounded future-selection visibility set of canonical tool names. Management catalogs annotate hidden registrations, while conversation choosers and auxiliary candidates exclude them. Existing conversation selections remain stable; source lifecycle and the current immutable Tool Snapshot, rather than the global visibility switch, own execution eligibility. A preference-read failure blocks new automatic/manual selection instead of silently exposing candidates.
- Each conversation stores independent manual and concrete automatic tool-name sets. For the first primary request, the auxiliary model classifies capability inspection or use and may return only exact typed MCP/non-built-in-extension source IDs. Eligible standalone tools bypass source selection and remain injected as individual tool resources; MCP Registry identities use the exact server ID, while generated namespaces remain only concrete naming/serialization details. Inspection exposes the complete eligible concrete catalog for one Provider request and snapshot, then leaves an initialized empty durable automatic set; use persists every member Host expands from selected sources plus eligible standalone tools. The always-visible `tool_resolve` control is use-only. Core validates and expands sources before atomically persisting use selection. The visible pre-call event contains only running/failure state or typed extension/MCP/tool/skill resource summaries and lifetime; each MCP/extension appears once with a member count, while concrete members stay internal to session state and Tool Snapshots. Replacement is non-cumulative and does not alter manual state. Before every Provider request, the Host prepares current Skill, Manifest v2 extension, and MCP catalogs, injects selected Skill/context resources, and freezes only eligible core, control, manual, request-only inspection, automatic, and separately authorized mode registrations in the Tool Snapshot. Later catalog changes do not mutate an existing automatic set.
- Long-tail tools, policies, environments, context resources, and Web views enter through versioned language-neutral extension manifests and protocols. Discovery and trust do not execute extension code.
- Required Host-owned Agent capabilities may ship as trusted built-in Manifest v2 extensions. Their contributed executors, including `aivo.projects`, are classified as standalone `tool` resources rather than user-installed extension sources; they do not enter auxiliary source-ID selection or expand the four unqualified primitive registry.
- The trusted `aivo.tools` built-in contributes only a non-mutating, turn-owned MCP registration proposal executor. MCP source inventory and concrete-tool metadata remain Host management and auxiliary-selection inputs rather than a primary-Agent listing executor. Only the privileged Host can confirm an exact proposal, persist a source disabled, probe it, and make it globally enabled and ready; the built-in cannot grant its own authorization, handle raw secrets, or populate cross-session manual activation.
- The Host auxiliary tool resolver receives only user intent and bounded single-line `kind:id：description` entries for MCP and non-built-in extension sources. It accepts only a strict classified JSON object whose `sources` contain unique exact typed IDs; concrete tool names, Skills, generated namespaces, reasons, and extra fields are invalid. Inspection expands the complete eligible catalog once and requires no source IDs; use validates and fully expands only returned sources. A blank MCP functional description uses the bounded deterministic assembly of every current eligible tool name and description; a blank extension description remains blank. Source identity, membership, eligibility, lifetime, and expansion are Host-owned, and descriptions are untrusted data. Skill/context selection remains a separate request-scoped resolver concern.
- MCP description generation is a separate explicit Host management operation. Core reads the complete current stored discovered-tool catalog, projects only bounded tool names and descriptions as untrusted data to the configured auxiliary model without executable tools, and returns a normalized unsaved draft. Connection configuration and credentials never cross this model boundary; empty or over-bound catalogs fail instead of producing a partial or inferred description.
- Composer resource mentions cross the desktop submit contract only as bounded typed kind/ID references. Core validates the complete set before user-event creation, applies project references through immutable session association, merges Skill and tool references into that session, and expands extension/MCP references only to the source's currently registered globally eligible tools. Visible prompt text and renderer display metadata are never authorization sources.
- The composer's combined local file-or-directory action is separate from resource-reference submission. Electron main owns the native chooser and reads file bytes only from the exact path returned by that chooser; preload exposes only a bounded file payload or selected directory path. The renderer routes files through existing attachment capability validation and directories through existing project registration/context selection.
- Extension processes, services, streams, calls, views, and artifacts have explicit owners, bounded queues, cancellation, draining, and deterministic teardown.
- A supervised local service may use a fixed validated loopback URL or bind port zero and announce its owned endpoint through one bounded versioned readiness record. Core owns the current-generation endpoint and bearer, rejects non-loopback or malformed announcements, and View resolution never trusts renderer or tool-result endpoint data.
- A Manifest v2 `views[].tools` association is resolved by the Host into optional bounded tool-detail metadata; renderer or extension result payloads cannot author backend origins, credentials, routes, or privileged surfaces. Electron may attach an isolated `WebContentsView` over a renderer-owned inspector content rectangle, but the Host-owned header, bounds, lifecycle, and fallback remain native.
- An attached embedded View is reused only while its owner plus Host-resolved extension/View/surface identity is stable. Electron main owns a monotonic mount revision and bounded selected-call context; context-only changes notify that one isolated WebContents without navigation, while identity changes and close paths retain full teardown.
- The extension loader accepts only the exact Manifest/API `2/2` pair and rejects v1 or mixed versions without a compatibility adapter. A v2 service may declare `runtime.messaging`; Electron main then brokers only that extension View's bounded `aivoExtension.runtime` one-shot messages and named long-lived Ports to fixed well-known endpoints on the current Host-resolved service generation. The Host owns bearer injection, connection IDs, limits, cancellation, and teardown; the page never receives the backend endpoint or a generic Aivo RPC surface.
- Local extension installation is a Host-owned preview/copy/confirm operation. Electron selects a source directory; Core validates and hashes without execution, revalidates the exact confirmation, copies into a private integrity-addressed staging directory beneath the platform application-data `Aivo/Default/Extensions` root, reloads the copy, and atomically publishes only the matching generation. Explicit isolated database stores may use a database-sibling root. Core alone grants trust, persists and restores the managed installation, migrates exact former-root packages through the same verified copy path, retains old generations until safe cleanup, and scopes uninstall or migration deletion to an exact Host-owned extension directory. Renderer, model, and extension content cannot choose destinations, write managed packages, delete files, or mint trust.
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
