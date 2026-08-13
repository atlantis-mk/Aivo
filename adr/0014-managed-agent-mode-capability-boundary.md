# ADR-0014: Keep Agent-mode capabilities out of managed definition data

- Status: Accepted
- Date: 2026-08-07
- Related Work: `CHG-2026-028-remove-agent-mode-toolsets`
- Closes OPEN: none

## Context

Schema-v6 Agent-mode definitions included `toolsets` in the same editable and persisted payload as display metadata, prompts, model sampling, and role. That made global mode CRUD a capability-policy editor and duplicated capability ownership across code defaults, stored overrides, and project configuration. The user requested removal from both the UI and backend data.

## Decision

- Global managed Agent-mode DTOs and stored definition JSON MUST NOT expose, accept as authority, or persist `toolsets`.
- Code MUST remain the capability owner for built-in modes, including when a built-in has a durable metadata/prompt override.
- Core MUST attach only the `safe` default to custom global modes at runtime.
- Existing project runtime configuration MAY continue to apply toolsets as the final project-scoped compatibility and policy overlay.
- Schema v7 MUST create or verify a v6 backup, then transactionally remove `toolsets` members from all stored Agent-mode definitions.
- Unknown legacy client input for the removed member MUST be ignored and MUST NOT reappear in reads or storage.

## Rationale

- Capability policy is safer and easier to reason about when it is not mixed with user-editable global mode metadata.
- Reattaching code defaults preserves the intended behavior of built-ins after edits without storing a second capability owner.
- A safe-only custom default avoids silently granting coding, shell, network, or extension capabilities.
- Retaining project overlays avoids breaking explicit repository-local runtime policy in this migration.

## Consequences

- Custom global modes can no longer opt into broader toolsets through management CRUD.
- Existing stored custom toolset values are removed from current data; the verified v6 backup is the recovery source.
- Runtime and transport serialization must be separated so internal toolsets remain available to Core but absent from management data.
- Downgrading requires restoration of the pre-migration backup rather than opening the v7 database in a v6 binary.

## Rejected alternatives

- Hide only the renderer field: backend clients and persisted rows would retain the unwanted second policy owner.
- Preserve stored toolsets but stop returning them: stale capability grants would continue to affect runtime invisibly.
- Remove toolsets from project configuration too: that would break an established project-scoped compatibility contract beyond the requested global-data change.

## Verification

`AT-AGENT-001`, `CT-RELIABILITY-001`, and `AT-UI-001` cover serialization omission, runtime capability restoration, migration backup/cleanup, project precedence, and the management surface.
