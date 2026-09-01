# ADR-0020: Resolve optional resources at runtime through resource_resolve

- Status: Accepted
- Date: 2026-08-21
- Related Work: `CHG-2026-037-intent-scoped-tool-injection`
- Supersedes decisions in: `ADR-0016`, `ADR-0018`
- Closes OPEN: none

## Context

The initial source-group selector runs before the primary Agent has inspected the task context. It can therefore mistake a resource-inventory question for a long-lived capability need, or select optional resources before the Agent knows whether its visible core surface is sufficient. Persisting inventory-driven selection wastes later prompt context, while automatically injecting every eligible schema into a first primary request makes inventory behavior expensive and surprising.

## Decision

- Core MUST NOT run automatic auxiliary resource selection before the first primary Provider request.
- The primary Agent MUST receive the required core tools and the Host-owned `resource_resolve` control by default, without the complete Skill, MCP, extension, or optional-tool catalog.
- `resource_resolve` MUST accept an explicit mode. `mode:"inspect"` returns bounded categorized summaries of matching currently eligible grouped-or-individual tool, Skill, and extension-context resources without exposing executable schemas, activating tools, or persisting automatic names. `mode:"use"` validates and completely expands selected resources without a Host-owned concrete-tool count cap and atomically persists selected concrete canonical tool names as the conversation automatic set for subsequent model steps and turns. Omitted or invalid modes MUST fail validation.
- Manual tools, explicit composer/tool-picker activation, global candidate visibility, source readiness, mode eligibility, permissions, registration identity, and stale-call refusal MUST retain their existing authority.

## Rationale

Timing follows model context: the primary Agent should ask for optional resources only after it knows it needs them. Separating `inspect` from `use` keeps inventory summary-only and non-persistent, while preserving deliberate durable replacement for actual execution capabilities.

## Consequences

The auxiliary response remains a strict object containing typed resource IDs rather than concrete grouped members or generated namespaces. First primary requests are smaller and more predictable because optional resources are absent until the Agent asks for them. `resource_resolve` requires a mode field, but session metadata needs no schema migration because only `use` updates the existing automatic set.

## Rejected alternatives

- Persist every inventory match: keeps unnecessary schemas in all later requests.
- Re-run automatic pre-request selection on every user turn: makes the conversation surface unstable and repeats auxiliary cost.
- Keep initial inspect injection for inventory prompts: spends an auxiliary call before the Agent has decided whether inventory is needed and may make the first request unexpectedly large.
- Let `inspect` expose full executable schemas: weakens the hidden-tool boundary when a bounded non-secret summary is enough for inventory.
- Return a partial use expansion when bounds are exceeded: presents an incomplete capability set as complete.
- Reject complete use expansion at a fixed Host-owned tool count: makes valid larger catalogs appear empty and prevents accurate activation.

## Verification

`AT-TOOL-001` covers strict `resource_resolve` mode validation and complete use expansion beyond the former 64-tool cap. `AT-SESSION-001` covers first-request omission of optional resources, runtime inspect summaries, durable use replacement, and restart-stable automatic selection. `AT-EXTENSION-001` covers source-group membership, Skill/context selection, and complete persistent replacement. `CT-SECURITY-001` and `CT-RELIABILITY-001` cover hidden-member exclusion, invalid output, cancellation, repetition, and cross-request isolation.
