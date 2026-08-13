# ADR-0015: Bind model delegation to Agent-mode subagent allowlists

- Status: Accepted
- Date: 2026-08-07
- Related Work: `CHG-2026-029-agent-mode-subagent-associations`
- Closes OPEN: none

## Context

Core already supports bounded child Agent sessions through `agent_delegate_task`, but any eligible parent toolset can name any subagent-capable mode. Dynamic mode management has no relationship field to guide the model or constrain that choice. Treating associations only as renderer hints would allow stale or forged model calls to bypass the user's configuration.

## Decision

- Managed and project Agent definitions MAY declare at most 16 normalized unique `subagents` IDs.
- Only primary-capable modes MUST be allowed to own associations. Every target MUST exist in the effective catalog, remain visible and subagent-capable, and MUST NOT be the owner itself or a protected worker.
- The effective association list MUST be an execution allowlist. Core MUST omit the delegation tool when the list is empty, narrow its Provider schema/context to the listed IDs when non-empty, and revalidate every call before child-session creation.
- The model MAY autonomously decide when a bounded associated delegation helps, but Core MUST NOT automatically fan out every request solely because associations exist.
- Save, role-change, and delete operations MUST prevent dangling global references. Project overlays MUST be validated before execution and MUST NOT mutate global definitions.
- Schema v8 MUST create or verify a v7 backup before recording support for persisted associations; existing rows MUST remain unchanged and begin with an empty allowlist.

## Rationale

- One Core-owned relation gives configuration, Provider exposure, and execution the same authority.
- Omitting the tool for an empty list avoids accidental legacy-wide delegation and makes user intent explicit.
- An enum-constrained schema plus execution-time validation improves model selection while retaining protection against forged/stale calls.
- Reusing the existing child-session executor preserves its established concurrency, cancellation, permission, and result lifecycle.

## Consequences

- Existing modes do not delegate until the user associates at least one child mode.
- Custom mode deletion and role changes acquire an additional reference check.
- Project configuration can tailor associations without changing global data, but an invalid effective graph blocks that mode's execution with an actionable error.
- Downgrade requires the v7 backup because older binaries refuse schema v8.

## Rejected alternatives

- Present associations only in the system prompt: model calls could still name unassociated modes and direct tool invocations would bypass the relationship.
- Keep unrestricted delegation when the list is empty: absence and intentional denial would be ambiguous, and existing parents would retain authority the user did not configure.
- Implement a new executor for each association: duplicate concurrency, cancellation, permission, and result semantics would increase failure paths.
- Automatically run all associated subagents on every prompt: it would waste provider capacity and ignore whether decomposition is useful for the current request.

## Verification

`AT-AGENT-001`, `AT-SESSION-001`, `CT-RELIABILITY-001`, and `AT-UI-001` cover validation, persistence/migration, Provider exposure, forged-call refusal, child lifecycle regression, and responsive configuration.
