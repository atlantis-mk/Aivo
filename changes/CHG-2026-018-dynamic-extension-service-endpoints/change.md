# Allocate dynamic loopback endpoints for local service extensions

## Problem or goal

Manifest v1 local-service extensions currently publish one fixed `runtime.url`. A fixed loopback port can already belong to another application and prevents independent extension instances from selecting collision-free endpoints. Local services that provide real-time HTTP or streamed Web-view data need an endpoint chosen while the supervised child owns the listener.

## Expected behavior

`REQ-EXTENSION-001` permits a local service to declare `runtime.transport: dynamic-http` without `runtime.url`. After trust and enablement, the child binds an operating-system-assigned loopback port and emits one bounded `aivo-extension-service/1` readiness record. Core validates and retains the actual endpoint only for that process generation, uses it for JSON-RPC and Web-view proxying, and preserves the existing Host bearer boundary. Fixed loopback `runtime.url` services remain compatible.

## Non-goals

No packaged static Web views, Unix-domain socket or Windows named-pipe transport, remote endpoint discovery, public listener, persistence, service multiplexing, arbitrary stdout protocol, port reservation by Core, or new renderer authority.

## Impact

Core Manifest validation accepts the new local-service transport, the Extension Supervisor owns a bounded startup handshake and actual runtime endpoint, and View resolution uses the validated client endpoint rather than the declarative Manifest URL. Electron IPC, renderer DTOs, database schema, providers, MCP/LSP, credentials, dependencies, and platform scope do not change. The development UI extension migrates from fixed port `47831` to an operating-system-assigned port.

## Implementation constraints

The child must bind before announcing the endpoint so another process cannot win a check-then-bind race. The Host accepts exactly the versioned readiness shape within ten seconds and 16 KiB, requires root `http://` on `127.0.0.1`, `::1`, or `localhost` with an explicit non-zero port and no userinfo/query/fragment, and closes/kills the child on timeout, EOF, overflow, malformed JSON, or invalid origin. The endpoint is never model-visible or persisted. The existing random bearer remains operation-private and all lifecycle, restart, idle, view-count, cancellation, and teardown ownership remains in Core.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `DYN-ENDPOINT-DOC-001` | `REQ-EXTENSION-001`, `NFR-SECURITY-001` | Accepted Manifest/handshake contract and ADR | `AT-EXTENSION-001`, `CT-SECURITY-001` | Completed |
| `DYN-ENDPOINT-CORE-001` | `REQ-EXTENSION-001`, `NFR-RELIABILITY-001` | Bounded dynamic endpoint startup and cleanup | `AT-EXTENSION-001`, `CT-RELIABILITY-001` | Completed |
| `DYN-ENDPOINT-VIEW-001` | `REQ-EXTENSION-001`, `NFR-SECURITY-001` | View descriptors proxy only the validated generation endpoint | `AT-EXTENSION-001`, `CT-SECURITY-001` | Completed |
| `DYN-ENDPOINT-FIXTURE-001` | `REQ-EXTENSION-001` | UI test extension binds port zero and reports readiness | `AT-EXTENSION-001` | Completed |
| `DYN-ENDPOINT-QA-001` | `NFR-SECURITY-001`, `NFR-RELIABILITY-001` | Focused failures plus full repository gates and real registration | `CT-SECURITY-001`, `CT-RELIABILITY-001` | Completed |

## Acceptance and evidence

- Two dynamic local services can receive independent non-zero loopback ports without Manifest edits or preselected ports.
- Enablement, tool execution, View resolution, actions, idle restart, stop, update, and removal use only the endpoint owned by the current generation.
- Timeout, premature EOF, oversized records, invalid JSON/protocol, missing/zero ports, credentials, non-root paths, query/fragment, and non-loopback origins fail before readiness and deterministically terminate the child.
- Existing fixed loopback service Manifests retain their behavior.
- Cancellation during startup tears down the child; repeated discovery/enablement does not retain stale endpoints.
- Verification includes focused Go and Node tests, `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, and `pnpm build`, plus a real Core registration and View resolution on macOS.

Evidence recorded on 2026-08-06 (macOS): focused Go tests passed for Manifest compatibility, two simultaneous dynamic services, tool/View/action routing, malformed and oversized frames, cancellation, EOF, timeout, and origin/port/path/query refusal. The Node extension tests passed for service behavior and the executable port-zero readiness handshake. `pnpm test:core`, `pnpm docs:check`, `pnpm scripts:test`, `pnpm lint`, and `pnpm build` passed; lint and build emitted only the repository's existing Fast Refresh, large-barrel, and chunk-size warnings. A separate current Core on `127.0.0.1:43118` registered `dev.aivo.ui-test` as Ready, resolved its logical View through dynamically assigned port `64487`, then stopped/re-enabled the extension and resolved the new generation through port `64504`. No UI layout changed, so wide/narrow screenshots are N/A.

## Security and data lifecycle

The readiness record contains only a protocol identifier and ephemeral loopback origin. Core stores it only in the live runtime client; it is not persisted, sent to models, exposed to the extension Web page, or included in diagnostics. The random Host bearer remains in child environment and Core/Electron main memory, and the Web page sees only its logical `aivo-extension://` origin. Failed or stopped generations lose both endpoint and bearer.

## Compatibility and migration

No schema or stored-data migration. `service` Manifests with omitted/`http` transport and a valid fixed `runtime.url` remain supported. `dynamic-http` requires no URL and requires a conforming readiness handshake. Rollback removes dynamic transport acceptance; fixed services continue to work, while dynamic-only extensions fail validation until upgraded or reverted.

## Bug root cause (type=bug only)

N/A.
