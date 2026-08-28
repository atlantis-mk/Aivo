# ADR-0029: Make tool selection grouping explicit registration metadata

- Status: Accepted
- Date: 2026-08-26
- Related Work: `CHG-2026-034-source-group-tool-selection`
- Supersedes decisions in: `ADR-0018`, `ADR-0020`
- Closes OPEN: none

## Context

Source- and namespace-derived grouping makes every tool from an MCP or extension behave as one selection unit even when the registrant intended independent tools. It also makes built-in Aivo tools appear under generated internal namespace labels and causes global management, conversation activation, composer selection, and auxiliary selection to use different item counts and switch granularity. Registration is the only point that can author stable grouping intent without asking the renderer or auxiliary model to infer membership.

## Decision

- Every executable tool MUST remain an independently named, registered, permission-checked, and snapshot-bound execution identity.
- A registration MAY additionally declare one bounded Host-owned selection group with a stable ID, display name, description, and complete member set. A tool MUST belong to at most one selection group. Absence of group metadata MUST mean individual selection.
- Registry and Manifest validation MUST reject malformed, duplicate, inconsistent, empty, or unknown group membership before any member is exposed. Manifest v2 MAY declare optional tool groups without changing its supported 2/2 version pair.
- MCP adapters MUST explicitly register each enabled server's current tools and Host resource utilities as one server selection group. Extensions and built-ins MUST be individual unless their registration or Manifest explicitly declares a group.
- Management, composer, and conversation activation surfaces MUST render one item per declared group or individual tool. A group item MUST disclose its concrete members but MUST expose only one group switch or selection action. Group activation changes the complete current eligible member set; individual activation changes only that tool.
- The auxiliary selector MUST receive exactly one sanitized candidate line containing the group display name for each group and one line for each individual tool. It MUST return only unique exact typed resource identities. It MUST NOT return or alter concrete membership for a group.
- The Host MUST expand a selected group to every currently eligible globally visible member and expand an individual candidate to exactly itself. Persisted conversation automatic state and immutable Tool Snapshots MUST continue to contain concrete canonical tool identities.
- Inspection MAY expose the complete eligible concrete catalog for one request as defined by `ADR-0020`; use and `tool_resolve` MUST preserve grouped-or-individual selection granularity before Host expansion.

## Rationale

Explicit metadata lets registrants choose the correct user and model selection unit while preserving existing per-tool execution security. It removes UI inference from generated namespaces, makes counts and switches consistent across surfaces, and lets the auxiliary model choose capabilities without becoming the owner of group membership.

## Consequences

- Catalog DTOs and Manifest validation gain optional group metadata, but no persistence schema changes.
- Existing MCP behavior remains grouped because the adapter declares a group explicitly. Existing extensions without a group declaration become individually selectable.
- Group visibility changes are applied to every concrete member; existing per-tool preferences may initially produce a partial group and are reconciled by the next group switch.
- Auxiliary prompt and strict response shapes change during development from source-only selection to typed group-or-tool resource selection.

## Rejected alternatives

- Infer groups from `namespace` or `sourceId`: generated transport metadata does not express registration intent and caused the current inconsistent UI.
- Store only group IDs in conversations or snapshots: later membership changes would silently change an existing capability surface.
- Let the auxiliary model return group members: duplicates Registry authority and permits fabricated or stale membership.
- Hide group members in management UI: prevents users from understanding what the single group switch controls.

## Verification

`AT-TOOL-001` verifies registration validation, one candidate per group or individual, strict selection, complete expansion, and snapshot identity. `AT-EXTENSION-001` verifies Manifest groups, MCP server groups, individual extension tools, and hidden-member exclusion. `AT-SESSION-001` verifies group and individual manual/automatic activation. `CT-SECURITY-001` verifies that renderer and model input cannot author membership. Desktop focused tests verify shared grouping, member disclosure, one group switch, counts, and composer references.
