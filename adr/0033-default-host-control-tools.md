# ADR-0033: Make plan and question operations default Host controls

- Status: Accepted
- Date: 2026-08-28
- Owners: Aivo maintainers
- Work: CHG-2026-062-default-plan-question-tools

## Context

`update_plan` and `ask_user` mutate Host-owned conversation state: one publishes a structured execution plan and the other creates a turn-owned blocking interaction. Encoding either operation as prompt keywords would make side effects ambiguous, difficult to validate, and vulnerable to accidental triggering by user or tool text. Treating them as ordinary dynamically selected resources can also remove them precisely when an Agent needs to communicate progress or request a material decision.

## Decision

1. `update_plan` and `ask_user` are reserved always-on Host control tools alongside the four default execution primitives.
2. They are registered by workspace construction, included in every eligible primary Agent tool snapshot, omitted from global management and conversation selection, and cannot be disabled or replaced by an extension or Provider.
3. The sole trigger is a schema-valid structured tool call. Prompts and tool descriptions explain appropriate use but never create side effects through keyword parsing.
4. `update_plan` remains permission-exempt because it updates only bounded Host presentation state. `ask_user` retains turn ownership, cancellation, waiting, reply validation, and deterministic cleanup.
5. Other Agent, automation, goal, and provider operations do not inherit this status. Provider-declaration and Provider-account tools remain dormant and route-activated.

## Consequences

- Every primary Agent can reliably report a plan and request a user decision without capability discovery.
- The default tool contract contains six always-on identities: `read`, `bash`, `edit`, `write`, `update_plan`, and `ask_user`, plus the separate `tool_resolve` Host selection control.
- Tests and documentation must distinguish the four execution primitives from the two Host controls while applying the same reservation, visibility, and snapshot guarantees.

## Alternatives considered

- Parse keywords or fenced text from assistant output. Rejected because ordinary content could trigger Host side effects and streaming/schema behavior would be ambiguous.
- Keep both as dynamically selected resources. Rejected because progress reporting and material clarification are control-plane capabilities required independently of task-specific resource discovery.
