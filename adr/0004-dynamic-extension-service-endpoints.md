# ADR-0004: Let supervised local services announce an owned dynamic loopback endpoint

- Status: Accepted
- Date: 2026-08-06
- Related Work: `CHG-2026-018-dynamic-extension-service-endpoints`
- Closes OPEN: none

## Context

Manifest v2 local services may declare a fixed loopback URL. Fixed ports can collide with unrelated applications or other extensions. Having Core probe an unused port and then asking a child to bind it creates a check-then-bind race, while allowing arbitrary announced URLs would let executable extensions redirect the privileged Host proxy outside the intended loopback boundary. Complex Web views still need HTTP and streamed responses even when packaged static views are added separately.

## Decision

- Manifest v2 local services MAY use `runtime.transport: dynamic-http` and MUST omit `runtime.url` in that mode; fixed HTTP services remain supported.
- A dynamic child MUST bind an operating-system-assigned loopback port before emitting exactly one newline-terminated readiness record with protocol `aivo-extension-service/1` and its URL.
- Core MUST bound the readiness record to 16 KiB and ten seconds, inherit caller cancellation, and terminate the child on timeout, EOF, overflow, malformed data, or invalid endpoint.
- Core MUST accept only root `http://` origins on `127.0.0.1`, `::1`, or `localhost` with an explicit non-zero port and no userinfo, query, or fragment.
- Core MUST retain the validated endpoint only in the current supervised runtime generation and MUST use that endpoint, rather than renderer or tool-result data, for JSON-RPC and View proxy descriptors.
- The Host MUST generate and inject the bearer before process start and MUST preserve existing header injection, redaction, View isolation, idle restart, draining, and teardown behavior.
- The endpoint and bearer MUST NOT enter persistence, model context, renderer state, extension page state, or diagnostic logs.

## Rationale

- Binding port zero before reporting gives the child exclusive ownership without a port-selection race.
- A single versioned bounded startup record is simpler and more portable than descriptor inheritance while leaving the steady-state service protocol unchanged.
- Strict loopback-origin validation preserves the existing Host proxy trust boundary.
- Keeping fixed URLs compatible avoids breaking existing development extensions.

## Consequences

- Supervised service stdout is reserved for the readiness record in dynamic mode; diagnostics must use safe stderr handling or another bounded channel.
- Every dynamic restart can receive a different endpoint, so View resolution must consult the live client generation.
- Extension authors must implement the readiness record before initialization requests can arrive.
- Packaged static Views and socket/pipe transports remain separate future decisions.

## Rejected alternatives

- Keep fixed ports: collisions remain and multiple instances require manual coordination.
- Let Core probe and pass an apparently free port: another process can bind it before the child.
- Accept any child-reported URL: this would turn the Host proxy into an arbitrary network bridge.
- Require Unix-domain sockets or inherited descriptors now: cross-platform Electron/Windows support and packaging are substantially more complex.
- Replace all services with packaged static Views: static assets do not cover real-time HTTP and streaming interfaces.

## Verification

`AT-EXTENSION-001` covers Manifest v2 validation, successful dynamic startup, actual-endpoint tool/View routing, and fixed-service behavior. `CT-SECURITY-001` covers bounded handshake and origin refusal. `CT-RELIABILITY-001` covers cancellation, timeout, restart, stale-generation exclusion, and deterministic process cleanup.
