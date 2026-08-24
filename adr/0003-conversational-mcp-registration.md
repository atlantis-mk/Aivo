# ADR-0003: Permit conversational MCP proposals with Host-owned confirmation

- Status: Accepted
- Date: 2026-08-06
- Related Work: `CHG-2026-017-conversational-mcp-registration`
- Closes OPEN: none

## Context

ADR-0002 correctly prevents models and auxiliary selection from installing or trusting executable extensions. Users nevertheless need a low-friction way to describe an MCP source in conversation and make it available later. Treating model output as trusted configuration would allow prompt injection or model error to persist commands, connect to remote origins, bind credentials, and start same-user processes. Treating installation as session activation would also violate conversation isolation and lose the desired cross-session availability.

## Decision

- Aivo MUST expose conversational MCP registration as a trusted Host-owned namespaced extension that creates non-mutating, exact, bounded proposals.
- The model and auxiliary resolver MUST NOT approve, trust, bind credentials, persist, enable, connect to, start, or execute a proposed source.
- The Host MUST validate proposals independently and MUST require an explicit native user confirmation bound to the exact immutable proposal before any durable mutation, external connection, or child process start.
- Proposal inputs and model-visible results MUST contain only configuration and credential references, never raw secret values.
- Confirmation MUST persist a source disabled, probe it with bounded cancellation, and make it ready and enabled only after successful capability discovery and collision validation. Failure MUST leave no eligible partial tool registration.
- Successful source enablement MUST be global durable installation state. It MUST NOT create or copy session manual-activation state. Later model exposure MUST remain request-scoped through the eligible catalog resolver and frozen Tool Snapshot.
- Initial scope MUST support MCP sources only. Arbitrary generated code, generic extension/plugin installation, package downloads, conversational removal, and always-injected global tools are excluded.

## Rationale

- A proposal preserves conversational convenience without assigning trust authority to probabilistic model output.
- Exact native confirmation makes the real command, origin, roots, and credential references visible at the privileged boundary.
- Reusing MCP discovery supplies authoritative tool schemas from the source instead of accepting model-authored executable schemas.
- Separating global installation from request/session activation preserves both cross-conversation availability and bounded prompt/tool exposure.
- Ephemeral proposals avoid a persistence migration and minimize stale authorization state.

## Consequences

- Core owns a bounded proposal lifecycle and a privileged confirmation contract in addition to existing MCP persistence and probing.
- Stdio commands and remote endpoints remain external dependencies with the user's process or remote authority; confirmation is not containment.
- Registration can fail after confirmation because discovery, collisions, or dependency health are validated before eligibility.
- A future generic extension marketplace, package installer, durable audit log, global always-active preference, or conversational removal requires separate approval.

## Rejected alternatives

- Let the model call existing save/enable RPC methods directly: this collapses proposal, authorization, persistence, and execution into an untrusted action.
- Save model-authored tool schemas directly: schemas and execution identity would not be grounded in a discovered implementation.
- Copy registered tools into every new session's active list: this violates session-scoped manual activation and grows model context without intent.
- Require settings-only registration: safe but does not deliver the requested conversational workflow.
- Persist pending proposals in SQLite: unnecessary for the first slice and introduces migration, replay, cleanup, and sensitive-data retention costs.

## Verification

`AT-EXTENSION-002` verifies proposal and confirmation contracts, exact presentation, validation, idempotency, restart availability, and later-conversation resolution. `AT-EXTENSION-001` verifies MCP discovery, canonical identity, collisions, catalogs, and snapshots. `AT-SESSION-001` verifies manual activation isolation. `CT-SECURITY-001` verifies authorization and secret/process/connection boundaries. `CT-RELIABILITY-001` verifies cancellation, timeout, rollback, retry, and cleanup.
