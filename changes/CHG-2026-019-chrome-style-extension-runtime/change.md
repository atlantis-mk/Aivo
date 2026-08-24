# Add Chrome-style extension runtime messaging

## Problem or goal

Aivo extension Views can fetch same-origin resources and invoke individually declared UI actions, but they lack the Chrome-style event channel needed to keep one page mounted while a supervised extension service sends live application state. Reopening or navigating a View to represent each Agent call loses local state and causes visible flashing. The first Aivo-native runtime slice should provide bounded one-shot messages and a long-lived stream without loading arbitrary Chrome extensions into Electron.

## Expected behavior

`REQ-EXTENSION-003` makes Manifest/API v2 the only accepted extension contract. A trusted enabled service extension declaring `runtime.messaging` can use `aivoExtension.runtime.sendMessage(message)` for one bounded request/response and `aivoExtension.runtime.connect({name})` for one bounded long-lived Port. Electron main brokers both to fixed well-known endpoints on the current Host-resolved service generation, retains the backend bearer, and delivers Port events to only the owning isolated View. Agent calls and context revisions do not navigate or recreate a same-identity View.

## Non-goals

No Chrome Web Store or CRX installation, Electron `loadExtension`, arbitrary Chrome API compatibility, content scripts, browser-tab injection, remote executable code, cross-extension messaging, extension-to-Host privileged commands, durable extension storage, general background-worker scheduler, or automatic Manifest v1 conversion.

## Impact

Go domain/manifest/View descriptors use a Manifest v2 permission contract and reject all other manifest/API version pairs. Electron main and extension preload gain bounded message/Port brokering and deterministic per-View teardown. The UI test service implements the fixed protocol endpoints and streaming demonstration. Renderer application IPC, database schema, providers, model tool schemas, credentials, MCP, LSP, worktrees, platform scope, and production dependencies do not change.

## Implementation constraints

ADR-0002 and ADR-0006 own the privilege boundary. Only a registered isolated extension sender can use the bridge; Manifest/API version, permission, extension/View identity, current readiness, fixed endpoint paths, body/event sizes, connection count, connect/request timeouts, cancellation, and close are Host validated. The Host never exposes backend origins or bearer tokens to the page. Port messages are JSON values, ordered per connection, bounded before forwarding, and stopped on View close, renderer loss, service EOF/failure, extension stop, or explicit disconnect. The Host accepts only the exact Manifest/API `2/2` pair and performs no legacy fallback.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `EXT-RUNTIME-MANIFEST-001` | `REQ-EXTENSION-003`, `NFR-SECURITY-001` | Manifest v2 permission validation and Host descriptor propagation | `AT-EXTENSION-003`, `CT-SECURITY-001` | Completed |
| `EXT-RUNTIME-MESSAGE-001` | `REQ-EXTENSION-003`, `NFR-RELIABILITY-001` | Bounded one-shot View-to-service runtime messaging | `AT-EXTENSION-003`, `CT-RELIABILITY-001` | Completed |
| `EXT-RUNTIME-PORT-001` | `REQ-EXTENSION-003`, `NFR-SECURITY-001`, `NFR-RELIABILITY-001` | Bounded streamed Port with deterministic teardown | `AT-EXTENSION-003`, `CT-SECURITY-001`, `CT-RELIABILITY-001` | Completed |
| `EXT-RUNTIME-SAMPLE-001` | `REQ-EXTENSION-003` | UI test extension proves message/stream updates without navigation | `AT-EXTENSION-003` | Completed |
| `EXT-RUNTIME-QA-001` | `NFR-SECURITY-001`, `NFR-RELIABILITY-001` | Focused tests plus docs/core/lint/build and Electron syntax gates | `AT-EXTENSION-003`, `CT-SECURITY-001`, `CT-RELIABILITY-001` | Completed |

## Acceptance and evidence

- Manifest/API v1 is rejected with an actionable unsupported-version error; Manifest/API v2 rejects mixed version pairs and duplicate or unknown permissions, and Views without `runtime.messaging` cannot use the bridge.
- A permitted isolated View completes one JSON request/response without learning the service origin or bearer; oversized, malformed, timed-out, stopped, or stale-generation requests fail safely.
- A permitted View opens at most eight named Ports, receives ordered bounded NDJSON events, posts bounded messages, and observes one disconnect on explicit close, EOF, failure, or View teardown.
- Same-identity Agent calls retain the WebContents, runtime Port, and page-local state while context and streamed content update without navigation.
- Different View identity and explicit close retain the existing teardown/reopen behavior.
- Applicable gates are focused Go/Node/renderer tests, Electron syntax checks, `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, and `pnpm build`; manual Electron interaction remains required before `Verified`.

Implementation evidence recorded on 2026-08-06: the Host accepts only Manifest/API `2/2`, rejects v1 and mixed version pairs without fallback, validates only the exact `runtime.messaging` permission for service/external runtimes, propagates the permission through the Host-resolved View descriptor, and initializes v2 process/service clients with their declared API version. All current built-ins and test fixtures use v2. Electron validates the registered isolated sender and current re-resolved View generation before each new message/Port, retains the backend bearer, bounds one-shot responses and NDJSON events to 64 KiB, caps each View at eight Ports and 256 events/second, queues at most 32 pre-connect page messages, and aborts/DELETEs connections on every close path. The extension preload exposes Chrome-shaped `sendMessage` and `connect` behavior under `aivoExtension.runtime`; focused preload tests prove queued open ordering, message dispatch, and exactly-once local disconnect. The Manifest v2 UI test service proves authenticated one-shot, ordered streamed state, View-to-service Port messages, explicit close, and dynamic-port startup. After the v2-only change, `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, `pnpm build`, and `git diff --check` pass. A clean restarted Electron interaction with repeated Agent calls remains pending, so this Work stays `Implementing`.

## Security and data lifecycle

Messages may contain extension-owned application data but never Host credentials, arbitrary Aivo RPC access, raw prompts, or ambient renderer state. Electron main owns bearer injection, connection IDs, AbortControllers, size/rate limits, and cleanup. The service owns its transient message/Port state and releases it on disconnect or supervised shutdown. Payload contents are not logged or persisted by the Host.

## Compatibility and migration

No database migration or automatic compatibility adapter exists. Installed/source extension packages must declare Manifest/API `2/2`; v1 packages fail discovery until their manifest is updated. Existing v2 `getContext`, `onContextChanged`, `invokeAction`, View proxy, and tool execution contracts remain supported. Older Aivo versions may reject v2 packages, so rollback also requires a package version compatible with that older Host.

## Bug root cause (type=bug only)

N/A.
