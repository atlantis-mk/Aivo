# ADR-0007: Use integrity-bound native visual installation for local extensions

- Status: Accepted
- Date: 2026-08-06
- Related Work: `CHG-2026-020-visual-extension-installation`
- Closes OPEN: none

## Context

The Extension Supervisor separates discovery, trust, and enablement but currently exposes those primitives directly. That is useful for tests and adapters, not a user installation design. Executable extensions have the user's OS authority, local source directories can change after selection, and in-memory registration does not survive restart. A native workflow must make the trust decision understandable without letting the renderer, model, or extension page grant authority.

## Decision

- Aivo MUST provide a native user-initiated directory picker and MUST NOT require users to type shell/RPC commands or raw paths to install a Manifest extension.
- Core MUST preview and validate the package without executing it, return bounded safe manifest/risk metadata, and compute the complete package integrity.
- Install confirmation MUST be bound to the exact canonical source path and integrity; Core MUST re-read the package and reject a mismatch before trust, enablement, or persistence.
- Executable runtime confirmation MUST state that the extension runs with the user's OS authority and is not sandboxed by Aivo.
- Core MUST persist the linked source root, validated manifest, confirmed integrity, desired enabled state, and safe status in SQLite schema v3.
- Restart restoration MUST revalidate Manifest v2 and integrity before trust or startup. Missing or changed packages MUST remain non-eligible and visible as errors until the user explicitly reviews them again.
- Models, extensions, extension Web views, and ordinary renderer content MUST NOT install, trust, update, or silently restore an unconfirmed integrity.
- Disable and uninstall MUST stop owned processes/streams/Views deterministically. Uninstall MUST delete only Aivo's installation record and MUST NOT delete the linked source directory.
- Schema v2-to-v3 migration MUST create or verify the recoverable v2 backup before a transactional table addition; downgrade MUST restore that backup.

## Rationale

- Native selection removes command-oriented setup while keeping filesystem access at the Electron boundary.
- Preview-then-confirm makes executable authority, permissions, and contributions visible before code runs.
- An integrity binding prevents mutable linked source from becoming a silent update channel.
- Persisting source references rather than copying developer packages keeps the first local workflow reversible and avoids package-manager/signing claims.

## Consequences

- Local extensions survive restart only while their source and confirmed integrity remain unchanged.
- Extension authors must explicitly reinstall/review after changing package contents.
- Schema v3, startup restoration, new Core RPCs, a preload picker, and responsive desktop states require migration/security/reliability tests.
- Managed package copies, signed distribution, remote registries, and automatic updates require later Work.

## Rejected alternatives

- Keep command/RPC registration: exposes implementation sequencing and provides no usable trust review or persistence.
- Persist only a path and trust every restart: mutable local files become an unreviewed code-update channel.
- Let the renderer read and hash packages: crosses the filesystem boundary and makes Core trust UI-authored metadata.
- Copy packages into app storage now: introduces ownership, update, signing, disk-cleanup, and rollback contracts beyond this local linked-source slice.
- Reuse legacy Plugin installation rows: conflates incompatible manifests, runtimes, lifecycle, and trust semantics.

## Verification

`AT-EXTENSION-004` covers preview, confirmation, persistence, list/control, restart restoration, and responsive UI. `CT-SECURITY-001` covers no-execution preview, integrity TOCTOU refusal, model/renderer denial, source privacy, and same-user authority wording. `CT-RELIABILITY-001` covers schema backup/rollback, missing/changed source recovery, startup failure, repeated operations, and deterministic teardown.
