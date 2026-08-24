# ADR-0006: Use an Aivo-native Chrome-style extension runtime message broker

- Status: Accepted
- Date: 2026-08-06
- Related Work: `CHG-2026-019-chrome-style-extension-runtime`
- Closes OPEN: none

## Context

Chrome extensions keep extension pages and background execution decoupled through bounded one-shot messages and long-lived Ports. That lifecycle maps well to Aivo's supervised services and isolated Web Views. Electron, however, supports only a subset of Chrome extension APIs, loads only unpacked extensions into persistent sessions, and does not target full Chrome compatibility. Directly loading third-party Chrome extensions would also bypass Aivo's language-neutral manifest, Core trust lifecycle, Host-owned credentials, and tool authorization model.

## Decision

- Aivo MUST implement its own versioned runtime messaging contract and MUST NOT represent it as full Chrome Extension API compatibility.
- The Host MUST accept only the exact Manifest/API `2/2` pair and MUST reject v1 and mixed version pairs without an implicit compatibility adapter.
- Manifest/API v2 services MUST opt in with the exact `runtime.messaging` permission before a View can use the runtime bridge.
- Electron main MUST broker one-shot messages and long-lived Ports between one registered isolated View and only its Host-resolved current extension service generation.
- The extension page MUST receive a Chrome-shaped `aivoExtension.runtime` API, not `chrome.*`, so unsupported Chrome behavior cannot be inferred.
- The broker MUST use fixed well-known service paths, inject the Host-owned bearer itself, validate the sender/View/permission, bound JSON bodies and streamed events, cap Ports per View, and deterministically cancel every connection on teardown.
- Runtime messages MUST NOT invoke arbitrary Aivo RPC, change extension identity/origin/actions/surface, enumerate credentials, cross extension boundaries, or grant installation/trust/enablement/tool authority.
- The first Port transport MUST use authenticated loopback/HTTPS HTTP with a bounded NDJSON response stream plus bounded POST messages and DELETE close, preserving compatibility with fixed and dynamic service endpoints without a new production dependency.

## Rationale

- A Chrome-shaped lifecycle gives extension authors a familiar durable-page and event-channel model while Aivo retains its existing trust and language-neutral runtime boundary.
- Fixed Host-brokered endpoints work for JavaScript, Go, Python, and remote services and keep bearer credentials out of Web content.
- NDJSON streaming uses the existing HTTP service transport, supports cancellation/backpressure, and avoids exposing a raw backend WebSocket URL or adding a dependency.

## Consequences

- Manifest/API v2, the extension View descriptor, Electron bridge, and service protocol gain explicit contracts and focused version/security tests.
- Existing v1 packages must update their manifests before discovery; Aivo performs no automatic conversion.
- Extension services implementing runtime messaging must provide the well-known one-shot and Port endpoints.
- Ports live only as long as their owning View and current service generation; durable background state and storage require later Work.
- Actual Chrome extensions need a future explicit import/adapter design rather than direct compatibility claims.

## Rejected alternatives

- Electron `loadExtension`: incomplete Chrome compatibility, persistent-session coupling, boot-time reload requirements, and a privilege model that does not match Aivo tools.
- Expose the dynamic backend URL/token to the page: leaks credentials and weakens origin/generation ownership.
- Poll View state or navigate per Agent call: wastes resources, loses page state, and reproduces flashing.
- Add WebSocket dependencies immediately: unnecessary for the first bounded streaming slice and expands packaging/licensing surface.

## Verification

`AT-EXTENSION-003` covers strict Manifest/API v2 validation, v1 and mixed-version refusal, permission gates, one-shot messages, ordered Port events/messages, and no-navigation reuse. `CT-SECURITY-001` covers sender/identity/credential isolation and bounds. `CT-RELIABILITY-001` covers timeout, EOF, overflow, cancellation, repeated disconnect, and deterministic View/service teardown.
