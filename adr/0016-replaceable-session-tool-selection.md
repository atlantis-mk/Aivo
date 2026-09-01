# ADR-0016: Use a replaceable session automatic tool set controlled by the primary Agent

- Status: Accepted
- Date: 2026-08-10
- Related Work: `CHG-2026-030-replaceable-session-tool-selection`
- Supersedes decisions in: `ADR-0002`, `ADR-0011`
- Closes OPEN: none

## Context

Request-scoped Host resource selection makes tool availability change without an explicit Agent decision, while warm leases accumulate recently used tools for an arbitrary number of turns. Hiding every discovery control also prevents the primary Agent from reporting a concrete capability gap and intentionally rebuilding its visible tool surface. Treating the global management switch as live revocation conflates future discovery visibility with conversation-owned activation.

## Decision

- Each conversation MUST own separate bounded manual and automatic canonical tool-name sets.
- The automatic set MUST be initialized from sanitized globally visible eligible candidates before the first primary request and MUST remain stable until successfully replaced.
- The primary Agent MUST always receive the Host-owned `resource_resolve` control in addition to the four execution primitives. This control is not an extension executor and cannot install, trust, enable, authorize, or execute another tool.
- Calling `resource_resolve` MUST use auxiliary selection over sanitized currently eligible candidates and MUST atomically replace the complete automatic set for the next model step. It MUST NOT union results with the prior automatic set or create a warm lease.
- Manual conversation activation MUST remain independent and MUST survive automatic replacement.
- Provider declarations and the immutable Tool Snapshot MUST contain only the four eligible primitives, `resource_resolve`, current manual tools, current automatic tools, and separately authorized mode controls such as narrowed delegation.
- Global tool preferences MUST filter future auxiliary candidates and new manual activation. They MUST NOT revoke an already selected conversation tool. Source disablement/readiness and exact current registration remain execution prerequisites.
- Resolver failure, cancellation, invalid selection, and required no-match MUST preserve the prior automatic set.

## Rationale

A stable visible surface lets the primary Agent reason about its actual capabilities. An explicit bounded refresh provides long-tail discovery without advertising hidden tool names or schemas. Replacing rather than accumulating keeps prompt size bounded and makes the selection state explainable. Separating global candidate visibility from conversation state matches the distinct ownership of global management and an active task.

## Consequences

The primary model receives one additional Host control schema. Initial selection adds one auxiliary call when an uninitialized conversation has eligible candidates. Later selection occurs only when the Agent calls the control. Existing warm metadata becomes inert. Global disablement is not an emergency kill switch for already active conversations; source disablement or conversation removal is required for revocation.

## Rejected alternatives

- Request-scoped selection before every primary request: availability changes implicitly and repeats auxiliary work.
- Add every new match to a cumulative session set: prompt cost grows without a replacement boundary.
- Three-turn warm leases: lease length is unrelated to task capability and creates hidden expiry.
- Keep discovery entirely invisible to the primary Agent: the Agent cannot deliberately refresh when its visible tools are insufficient.
- Make global preferences live revocation: it conflates candidate visibility with existing conversation ownership.

## Verification

`AT-SESSION-001`, `AT-TOOL-001`, and `AT-EXTENSION-001` cover initial persistence, replacement, manual independence, Provider visibility, and Tool Snapshot identity. `CT-SECURITY-001` covers candidate filtering, hidden-schema isolation, source disablement, and stale-call refusal. `CT-RELIABILITY-001` covers resolver failure/cancellation, idempotence, restart recovery, and inert warm metadata.
