# ADR-0020: Classify initial tool inspection separately from conversation use

- Status: Accepted
- Date: 2026-08-21
- Related Work: `CHG-2026-037-intent-scoped-tool-injection`
- Supersedes decisions in: `ADR-0016`, `ADR-0018`
- Closes OPEN: none

## Context

The existing source-group selector answers only which capability groups match. It cannot distinguish a user asking to inspect the available tool surface from a user asking the Agent to perform work with selected capabilities. Persisting inventory-driven selection wastes later prompt context, while selecting only a few groups prevents the primary model from answering a complete inventory question.

## Decision

- The initial auxiliary tool decision MUST classify the user intent as `inspect` or `use` in addition to returning only selected exact typed MCP/extension source IDs.
- `inspect` MUST expose the complete currently eligible concrete catalog to exactly one primary Provider request and its immutable Tool Snapshot. Host expansion MUST NOT impose a concrete-tool count cap. It MUST persist no automatic names and MUST mark initial automatic selection complete with an empty durable set.
- `use` MUST validate and completely expand selected source groups without a Host-owned concrete-tool count cap and atomically persist their concrete canonical names as the conversation automatic set.
- `tool_resolve` MUST remain use-only: a successful result atomically replaces the durable automatic set for subsequent model steps and turns.
- Manual tools, global candidate visibility, source readiness, mode eligibility, permissions, registration identity, and stale-call refusal MUST retain their existing authority.

## Rationale

Lifetime follows user intent: inspection needs broad temporary visibility for an accurate answer, while execution benefits from a stable capability surface. Keeping `tool_resolve` persistent preserves deliberate later refresh without making every user turn repeat auxiliary selection.

## Consequences

The auxiliary response is a strict classified object containing typed source IDs rather than concrete tools or generated namespaces. A first inventory request may be large because the Host includes every eligible schema for one Provider call; Provider-specific declaration limits or context limits may still reject the request, but the Host does not pre-emptively omit or reject tools by count. Session metadata needs no new field: an initialized empty automatic set represents completed non-persistent inspection.

## Rejected alternatives

- Persist every inventory match: keeps unnecessary schemas in all later requests.
- Re-run pre-call selection on every user turn: makes the conversation surface unstable and repeats auxiliary cost.
- Let the primary Agent call `tool_resolve` for inventory: violates the existing hidden-tool boundary and makes an explicitly persistent control serve conflicting lifetimes.
- Return a partial inventory when bounds are exceeded: presents an incomplete list as complete.
- Reject complete expansion at a fixed Host-owned tool count: makes valid larger catalogs appear empty and prevents an accurate inventory.

## Verification

`AT-TOOL-001` covers strict classification parsing and complete expansion beyond the former 64-tool cap. `AT-SESSION-001` covers one-request versus durable visibility and restart-stable empty initialization. `AT-EXTENSION-001` covers source-group membership and complete persistent replacement. `CT-SECURITY-001` and `CT-RELIABILITY-001` cover hidden-member exclusion, invalid output, cancellation, repetition, and cross-request isolation.
