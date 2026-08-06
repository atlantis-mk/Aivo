# ADR-0009: Install extensions into Host-managed package storage

- Status: Accepted
- Date: 2026-08-06
- Related Work: `CHG-2026-023-managed-extension-packages`
- Supersedes: ADR-0007 linked-source persistence decision
- Closes OPEN: none

## Context

ADR-0007 established native preview, integrity confirmation, and durable extension restoration, but deliberately retained user-owned source directories. That linked model is useful for development yet leaves availability tied to source location and differs from Chrome-style managed installation. The user has selected a managed-copy lifecycle for the product installer.

## Decision

- Native selection and non-executing preview remain bound to the exact canonical source path and complete package integrity.
- After confirmation, Core MUST copy the bounded package into a private Host-owned root organized by extension ID and integrity generation.
- Core MUST reject symlinks and unsupported files, copy into a unique staging directory, reload the copied Manifest v2 package, require the confirmed integrity, and publish with a same-filesystem atomic rename before discovery or execution.
- Runtime discovery, persistence, restoration, and View/resource resolution MUST use only the managed copy after installation; the selected source path is not retained in the successful record.
- Editing, moving, or deleting the original source MUST NOT affect a successful installation. Updates require a new preview/confirmation and create a new managed generation.
- Disable MUST retain the managed package. Uninstall MUST stop runtime, remove only the exact managed extension directory plus persistence record, and preserve the original source.
- Schema v4 MUST distinguish historical `linked` rows from `managed` rows. Startup MAY promote an unchanged exact linked row by the same copy-and-verify flow; a missing or changed linked source MUST remain disabled for explicit reinstall.
- Abandoned staging directories and non-current generations MUST be cleaned only when no runtime can own them. A running update may retain the prior generation until safe restart or uninstall.
- Renderer, models, and extension content MUST NOT select the managed destination, write managed packages, mint trust, or request deletion outside the exact managed root.

## Rationale

- A Host-owned immutable generation gives installation stable behavior independent of a development workspace.
- Copy verification closes source-to-destination races and atomic rename prevents partial packages from becoming discoverable.
- Integrity-addressed generations make updates, rollback retention, and cleanup explicit.
- Distinguishing historical linked rows permits a safe one-way migration without silently trusting changed source.

## Consequences

- Aivo owns disk cleanup for installed packages and must treat uninstall as a destructive but tightly scoped operation.
- Installed packages consume additional local disk space and source edits require explicit update installation.
- Schema v4, filesystem migration, staged cleanup, rollback tests, and revised UI wording are required.
- Managed storage does not imply publisher identity, signature verification, Chrome API compatibility, or sandboxing.

## Rejected alternatives

- Keep linked sources as the product default: mutable availability and lifecycle do not match the selected install model.
- Copy without revalidation: a source race could publish code different from the confirmed preview.
- Copy directly into the final directory: crashes or copy failures could expose a partial package.
- Store one mutable directory per ID: loses generation identity and makes update rollback/cleanup ambiguous.
- Let Electron perform the copy: moves package trust and persistence sequencing out of Core.

## Verification

`AT-EXTENSION-004` covers native preview, managed install/update/list/control/uninstall, source independence, restart restoration, and schema v3 promotion. `CT-SECURITY-001` covers copy TOCTOU, path containment, symlink refusal, private permissions, and deletion scoping. `CT-RELIABILITY-001` covers staging failure, atomic publication, migration backup/rollback, abandoned cleanup, repeated operations, and runtime teardown.
