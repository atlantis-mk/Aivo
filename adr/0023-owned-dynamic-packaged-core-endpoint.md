# ADR-0023: Use an owned dynamic endpoint for packaged Core

- Status: Accepted
- Date: 2026-08-25
- Related Work: `CHG-2026-049-dynamic-packaged-core-port`, `BUG-2026-050-dynamic-core-csp`
- Closes OPEN: none

## Context

The packaged desktop currently probes the development port `127.0.0.1:43117` and reuses any process whose `/health` route responds successfully. That process may belong to a development session or another Aivo instance and may disappear without the packaged desktop owning or observing its lifecycle. Choosing a free port in Electron before spawning Core would retain a bind-time race.

## Decision

Packaged Electron MUST spawn and own its bundled Core with `AIVO_CORE_ADDR=127.0.0.1:0`. Core MUST let the operating system select the non-zero port and, only when explicitly requested by its parent, emit one versioned readiness record containing the actual root HTTP origin. Electron MUST accept only the supported record version and an exact loopback HTTP root origin, then health-check that endpoint before creating renderer consumers.

The main renderer preload MUST receive the accepted endpoint through Host-authored startup configuration. The main renderer Content Security Policy MUST permit HTTP and WebSocket connections to any port on the exact `127.0.0.1` host so the operating-system-assigned endpoint remains usable, while retaining the existing restrictions for every other host. Renderer input MUST NOT select or alter the endpoint. Packaged startup MUST NOT probe or reuse the fixed development port. Development mode and an explicit operator-owned `AIVO_CORE_URL` override may retain their existing endpoints.

## Rationale

- Binding port zero delegates collision-free allocation to the operating system and avoids a port-selection race.
- A versioned readiness record avoids parsing incidental log wording and keeps endpoint validation at the Electron privilege boundary.
- Owning the child process gives packaged shutdown and failure handling a deterministic lifecycle.

## Consequences

- Every packaged desktop process receives a distinct ephemeral loopback endpoint and no longer attaches to a development Core accidentally.
- Core gains a bounded opt-in stdout readiness contract; normal development output remains unchanged.
- Every Core URL consumer, including RPC, terminal, diagnostics, and extension views, must use the accepted runtime endpoint.
- The renderer CSP permits arbitrary ports only for the exact IPv4 loopback address; endpoint discovery and validation remain Host-owned rather than CSP-authorized.
- Target-platform package smoke remains required because child-process stdio and executable startup are platform behavior.

## Rejected alternatives

- Keep `43117` and retry after conflicts: this cannot distinguish ownership and still fails when another healthy process occupies the port.
- Ask Electron to find a free port and then pass it to Core: closing the probe listener before Core binds creates a time-of-check/time-of-use race.
- Reuse any healthy local Core: health alone proves neither compatible identity nor lifecycle ownership.

## Verification

`CT-RELIABILITY-001`, `CT-SECURITY-001`, desktop static/build checks, and target-platform package smoke validate the decision.
