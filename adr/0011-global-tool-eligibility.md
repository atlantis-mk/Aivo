# ADR-0011: Persist global tool eligibility as Host-owned preferences

- Status: Superseded in part by `ADR-0016`
- Date: 2026-08-07
- Related Work: `CHG-2026-025-global-tool-eligibility`
- Closes OPEN: none

## Context

Tool source enablement, per-conversation activation, and automatic Host selection are separate states. The global tool UI currently reuses the session activation RPC, so no durable per-tool global state exists and automatic selection can contradict the displayed switch.

## Decision

- Core MUST own a bounded set of globally disabled canonical tool names in a dedicated preference table; absence MUST mean enabled.
- Global tool configuration MUST use a dedicated typed RPC and MUST NOT write session activation.
- Superseded by `ADR-0016`: a disabled tool remains listable but cannot enter future conversation choices, auxiliary candidates, or new manual activation; an existing conversation selection is not retroactively revoked.
- Source trust, enablement, readiness, and eligibility MUST remain prerequisites; per-tool enablement MUST NOT start or enable a source.
- Configuration-read failure MUST fail the Provider preparation path rather than silently bypass known global policy.
- Runtime execution MUST continue to require the immutable snapshot so a stale or forged tool call cannot bypass the global filter.

## Rationale

- A dedicated table gives the Host one reviewable global policy owner without overloading source lifecycle or session metadata.
- A canonical-name deny set preserves compatibility because missing state enables all existing tools and newly registered names start enabled.
- Filtering before resolution and again through snapshot assembly keeps UI, automatic selection, Provider exposure, and execution consistent.

## Consequences

- A new local RPC and schema-v5 preference table require synchronized Core, persistence, migration, and renderer contract tests.
- Reusing a previously disabled canonical name keeps it disabled, which is safer than silently re-enabling a replacement registration.
- Older application versions ignore the setting and may expose tools after downgrade.

## Rejected alternatives

- Store the switch in renderer local storage: automatic Host selection cannot trust or read renderer-owned state.
- Reuse an unrelated application-config JSON column: it would hide tool policy inside another subsystem's format and weaken migration ownership.
- Reuse session activation: absence from one session is not global disablement and automatic selection is intentionally independent.
- Mutate live registry entries only: state would not survive restart and independently rebuilt registries could bypass it.
- Remove disabled tools from the management catalog: users could not discover or re-enable them.

## Verification

`AT-TOOL-001`, `AT-EXTENSION-001`, `AT-SESSION-001`, and `CT-SECURITY-001` cover persistence, catalog annotation, automatic-selection exclusion, snapshot exclusion, RPC separation, and restart behavior.
