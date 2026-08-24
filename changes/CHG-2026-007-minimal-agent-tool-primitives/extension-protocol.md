# Language-neutral extension and Web UI protocol

## Model

Aivo extensions are protocol implementations, not TypeScript-only plugins. Supported runtime forms are:

| Runtime | Description |
| --- | --- |
| `builtin` | Go implementation compiled into Aivo Core |
| `process` | A supervised executable written in Python, JavaScript, Go, Rust, or any language |
| `service` | A supervised local HTTP/event service |
| `external` | A configured remote service |
| `static` | Skills, prompts, schemas, and other non-executable resources |

An extension may contribute model-visible tools, invisible policy interceptors, an Execution Environment, context resources, or Web views. Every contribution is declared before executable code is trusted or loaded.

## Manifest v1

```json
{
  "schemaVersion": 1,
  "id": "com.example.github",
  "name": "GitHub Extension",
  "version": "1.0.0",
  "description": "GitHub tools and project views",
  "apiVersion": "1",
  "runtime": {
    "type": "process",
    "command": "./bin/github-extension",
    "args": ["--stdio"],
    "transport": "stdio"
  },
  "contributes": {
    "tools": [
      {
        "name": "github.search_issues",
        "description": "Search GitHub issues",
        "schema": "schemas/search-issues.json",
        "activation": "auto",
        "capability": "github.issues.read"
      }
    ],
    "views": [
      {
        "id": "github-tool-detail",
        "title": "GitHub Result",
        "type": "web",
        "route": "/ui/tool-result",
        "surfaces": ["tool-detail"],
        "tools": ["github.search_issues"]
      }
    ]
  },
  "requirements": {
    "network": true,
    "credentials": ["github"],
    "platforms": ["darwin", "linux", "windows"]
  }
}
```

Manifest discovery is declarative. Paths resolve inside the extension package and cannot escape it. Requirements disclose runtime needs; they do not grant containment or authority.

Go built-ins embed the same manifest shape and use `runtime.type=builtin`. A process package owns its executable runtime or declares an external executable requirement; Aivo does not infer language or automatically install untrusted dependencies.

## Contributions

- Tool Extension: registers model-visible namespaced tools.
- Policy Extension: registers pre/post-execution interceptors without adding model schemas.
- Environment Extension: provides all four primitive operations and their coherent filesystem/process/artifact boundary.
- Context Extension: supplies skills, prompts, resources, or dynamic context without pretending to be an execution tool.
- View Extension: supplies isolated Web views for settings, pages, dialogs, tool details, or notifications.

Ordinary extension tools use stable namespaces such as `web.search`, `agent.delegate`, `automation.create`, `mcp.github.search_issues`, or `plugin.notion.create_page`. Load order never resolves collisions. Duplicate names or attempts to register a reserved core name fail atomically.

## Protocol v1

One JSON-RPC 2.0 method model spans stdio, local service, and external service transports. Streaming events use framed JSON/NDJSON for processes and SSE or WebSocket for services. Images and large payloads use artifact references rather than unbounded base64 in JSON.

Core methods:

```text
extension/initialize
extension/activate
extension/deactivate
extension/shutdown
catalog/list
catalog/changed
tool/execute
tool/cancel
ui/list
ui/event
credential/request
health/check
```

Protocol initialization negotiates version and capabilities before any contribution becomes Ready. Tool execution carries cancellation, operation, session, turn, Tool Snapshot, and registration identity. Progress events are bounded and ordered. A result returns bounded model content plus structured details/artifact/view references.

## Registration identity

```ts
type ToolRegistration = {
  name: string
  extensionId: string
  extensionVersion: string
  schemaHash: string
  implementationHash: string
  registrationId: string
}
```

Actual runtime registration must match declared names and schemas. Registration is atomic per extension version. Changed version, schema, implementation, or dynamic catalog creates a new registration identity; existing request snapshots keep their prior identity until calls drain.

## Lifecycle

```text
Discovered -> Validated -> Untrusted -> Enabled -> Starting -> Ready -> Active -> Draining -> Stopped
```

`Error` is a first-class state reachable from validation, startup, runtime, health, or shutdown failures.

- Discovered/Validated reads only declarative metadata and schemas.
- Untrusted executable code cannot load, connect, obtain credentials, or activate.
- Enabled permits startup but does not expose every declared tool.
- Starting owns process/service launch, protocol negotiation, health, and cancellation.
- Ready contributes an eligible catalog.
- Active means at least one tool, policy, environment, context, or view is in use.
- Draining accepts no new work, waits for bounded completion, then cancels and tears down remaining work.
- Stopped releases processes, sockets, streams, views, artifacts, and credential leases.

Project configuration cannot silently trust a new extension, enable a globally disabled extension, bind credentials, override core tools, or weaken user restrictions. Model calls cannot install or trust extensions.

## Static and dynamic catalogs

Static tools are indexed from validated manifest schemas without starting executable code. Dynamic providers such as MCP connect only after explicit enablement, retrieve and validate their catalog, and cache a catalog hash. Auxiliary selection considers only Ready, eligible, `auto` entries. `manual` entries require user activation; Agent Mode `default` entries activate only under the declared mode.

Catalog refresh, service reconnect, or MCP schema change produces new registrations. Unavailable or stale entries are excluded before auxiliary selection. Full schemas are loaded only for the selected/pinned/default tools needed in the next model request.

## Auxiliary activation

The Host first applies deterministic filters for trust, enablement, mode, project, platform, configuration, activation policy, availability, and catalog revision. If multiple relevant long-tail candidates remain, the configured auxiliary model receives only the current intent and sanitized compact entries and returns exact names. Host validation is authoritative.

Auxiliary selection cannot install code, establish trust, bind credentials, change policy, choose an Execution Environment, or execute a tool. Failure or uncertainty selects nothing and does not block core execution. Selection state is separated into pinned, bounded warm, and current-turn tools.

## Policy and environment

Activation is prompt exposure, not authorization. A call passes through schema validation, ordered Policy interceptors, the selected Execution Environment, and result interceptors. Modified arguments are revalidated. Policy denial is an explicit result. Policy extensions cannot grant authority outside the actual environment.

An Environment Extension replaces the operations behind all four core primitives as one atomic selection. Switching environment creates a new Tool Snapshot. Failure never silently falls back from a contained or remote environment to local execution.

## Web Service UI

Web Service views are the recommended complex extension UI. Supported Host surfaces are `page`, `dialog`, `tool-detail`, `settings`, and `notification`. Persistent workspace chrome is not granted by default.

The Host exposes a logical URL such as `aivo-extension://<extension-id>/<route>` and privately proxies it to a supervised local or configured external service. Real ports, service authentication, and credential tokens are not exposed to the page. Local services bind only to an approved loopback/socket boundary.

Every extension view runs without Node integration or privileged preload access, with context isolation, Electron sandboxing, restrictive CSP, blocked arbitrary navigation/window creation by default, origin separation, and a versioned message bridge. The bridge provides theme, locale, view context, bounded data references, resize/close, notification, and declared action invocation. Actions return through Host validation and policy; a Web page cannot directly invoke arbitrary privileged Aivo services.

Tool results keep `modelContent` separate from UI `details`. A Web view receives a bounded state or private data reference. If an extension is stopped, removed, or unreachable, stored safe summary/details provide generic historical rendering.

## Credentials and private data

Manifests declare credential slot names, never values. User/project configuration binds slots to Host-owned secure-store entries. Runtime code requests only bound values for the operation that needs them and cannot enumerate the store. Values are excluded from UI state, logs, errors, manifests, session metadata, and auxiliary resolver input.

Executable extension code runs with its process or remote identity and is trusted software, not a sandboxed script. A separate process improves lifecycle and crash isolation but is not containment. External process isolation and least-mounted files, network, and credentials remain the real security boundary.

## Update and removal

An update is validated as a new immutable registration set. The old version drains while new turns use the new version only after Ready. Failed updates leave the prior version available. Uninstall removes eligibility, drains calls and views, releases resources, and deletes extension-owned non-user data while preserving bounded historical tool records.

## Frozen protocol and lifecycle policy

- Protocol v1 uses JSON-RPC 2.0 over newline-delimited UTF-8 JSON for stdio. Frames are capped at 1 MiB, each direction has a 64-frame bounded queue, and progress is coalesced to at most 16 pending events per operation before the producer is backpressured.
- Supervised local HTTP uses a random 256-bit Host bearer secret over loopback; Unix-domain sockets are preferred on macOS/Linux and named pipes on Windows when the runtime supports them. External services require HTTPS and an explicitly bound Host credential. SSE is the initial service stream; WebSocket capability must be negotiated.
- A package records a SHA-256 integrity digest over its normalized manifest, schemas, declared Web assets, and executable entry points. Any executable digest or version change returns the package to `Untrusted`; no signing authority is introduced in this Work.
- A crashed process may restart three times with one-, two-, and four-second delays. An unused Ready process or service stops after five minutes; Active work and views prevent idle shutdown.
- Web content uses `aivo-extension://` through a Host proxy with `default-src 'none'`; scripts, styles, images, fonts, and connections are allowed only from the extension's isolated logical origin and explicitly proxied resources. Inline scripts and arbitrary navigation are denied.
- The v1 bridge permits theme/locale/view state, bounded resize, close, notification, and manifest-declared action invocation. Artifact and data references are session-bound, operation-scoped where possible, expire after ten minutes unless a retained historical summary owns them, and are revoked on extension stop.
- Credential leases are operation-scoped, non-enumerable, and revoked on completion or cancellation. Each third-party executable is supervised as its own child process by Core; a separate privileged supervisor process is not introduced.
- Manifest and Protocol major version `1` require exact compatibility. Unknown optional fields are ignored, unknown required capabilities fail validation, and removal or incompatible evolution requires a future accepted Work with a documented deprecation window.
