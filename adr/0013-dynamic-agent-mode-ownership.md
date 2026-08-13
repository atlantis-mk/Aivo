# ADR-0013: Persist dynamic Agent modes as Core-owned definitions

- Status: Accepted
- Date: 2026-08-07
- Related Work: `CHG-2026-027-dynamic-agent-modes`
- Closes OPEN: none

## Context

Selectable Agent modes are code defaults plus optional file-based overlays. That model has no safe desktop CRUD owner, cannot distinguish a built-in default from a user edit, and cannot make the management UI and runtime resolver share one durable catalog. Definitions influence prompts, model selection, tool eligibility, and permission scope, so renderer-only state is insufficient.

## Decision

- Core MUST own durable global Agent-mode definitions in a dedicated schema-v6 table and expose typed CRUD application methods.
- Code MUST remain the owner of built-in defaults. A saved row with a built-in ID MUST act as a full validated override, and deleting that row MUST restore the code default.
- Hidden internal worker IDs MUST NOT be created, edited, or deleted through management RPCs.
- User-created modes MUST be deletable only when no durable session references the ID; deletion MUST be transactional and leave no partial catalog change.
- Global persisted definitions MUST overlay code defaults, and project runtime configuration MUST remain the final project-only compatibility overlay.
- Core MUST validate identity, bounds, mode role, permission scope, toolsets, model reference, and protected identities before persistence and again before runtime use.
- Renderer state MUST NOT be an authority for prompts, capabilities, permissions, or deletion safety.

## Rationale

- A dedicated table gives UI and runtime one restart-safe owner without teaching the renderer to edit arbitrary files.
- Code defaults plus removable overrides preserve upgrades and provide an exact reset path for built-ins.
- Keeping project files as the final overlay preserves repository-local configuration and avoids an irreversible migration of user-authored files.
- Protecting internal workers prevents user edits from breaking compaction, title generation, and scheduled execution.

## Consequences

- Schema v6, migration backup coverage, new local RPCs, catalog composition, and responsive management UI must ship together.
- Built-in definitions may evolve in later application versions only for users without an override for that ID.
- Custom modes referenced by historical sessions require explicit reassignment before deletion.
- Downgrading to v5 ignores persisted Agent-mode edits.

## Rejected alternatives

- Edit global Markdown/JSON files from the renderer: file ownership, concurrent edits, symlink handling, and validation errors would cross the privilege boundary and still not provide safe referenced-mode deletion.
- Store one catalog JSON blob in application config: per-mode uniqueness, deletion checks, migrations, and bounded queries would be opaque.
- Copy every built-in into SQLite during migration: copied defaults would stop receiving safe product updates and make reset semantics ambiguous.
- Allow management of hidden workers: user changes could disable required background behaviors with no recoverable foreground mode.

## Verification

`AT-AGENT-001`, `AT-SESSION-001`, `CT-RELIABILITY-001`, and `AT-UI-001` cover composition, CRUD, reference refusal, restart, migration, picker integration, and responsive management states.
