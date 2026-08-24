# Add visual local extension installation

## Problem or goal

Manifest v2 extensions currently expose only low-level `DiscoverExtension`, `TrustExtension`, and `EnableExtension` RPCs. A user must manually supply paths and sequence privileged calls, no native screen explains what will execute, and registration disappears after restart. Aivo needs one visual local-extension workflow that makes selection, inspection, trust, enablement, restoration, and failure visible.

## Expected behavior

`REQ-EXTENSION-004` adds a software-owned extension section. The user chooses a local extension directory through the native OS picker. Core performs a non-executing preview and returns the validated Manifest v2 identity, runtime, contributions, declared permissions/requirements, source path, and integrity. Only an explicit confirmation bound to that exact path and integrity may install and optionally enable it. The linked installation is persisted and restored after restart only while its current integrity still matches the confirmed value.

## Non-goals

No Chrome Web Store, remote package registry, archive download/extraction, source copying, dependency installation, shell command entry, model-initiated install/trust, silent source update, automatic directory scanning, credential entry, signing/notarization claim, or arbitrary Chrome extension compatibility.

## Impact

React gains a responsive extension list and install-review dialog. Electron main/preload gains a directory-picker capability only. Go domain/application code gains preview/install/list/enable/uninstall orchestration and restart restoration. SQLite schema v3 adds `extension_installs`; no provider, prompt, MCP, LSP, terminal, worktree, or extension runtime-message protocol changes. No production dependency is added.

## Implementation constraints

ADR-0002 and ADR-0007 own trust and persistence. Preview must validate and hash without executing extension code. Confirmation must include the exact preview integrity and Core must re-read/re-hash before mutation to prevent selection-to-install changes. Executable extensions run with the user's OS authority and the UI must say so. Source changes, missing sources, failed startup, and migration failure cannot silently produce eligible tools. Disable/uninstall must deterministically stop owned runtime state; uninstall removes only Aivo's record and never deletes the source directory. Models and extension Web content cannot invoke install/trust APIs.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `EXT-INSTALL-DOC-001` | `REQ-EXTENSION-004` | Accepted Work, ADR, Requirement, data/security/test/traceability contract | `AT-EXTENSION-004` | Completed |
| `EXT-INSTALL-STORE-001` | `REQ-EXTENSION-004`, `NFR-RELIABILITY-001` | Schema v3 table, backup migration, durable linked-install CRUD and startup restoration | `AT-EXTENSION-004`, `CT-RELIABILITY-001` | Completed |
| `EXT-INSTALL-CORE-001` | `REQ-EXTENSION-004`, `NFR-SECURITY-001` | Non-executing preview and integrity-bound install/enable/uninstall RPCs | `AT-EXTENSION-004`, `CT-SECURITY-001` | Completed |
| `EXT-INSTALL-DESKTOP-001` | `REQ-EXTENSION-004`, `NFR-UI-001` | Native folder selection, manifest/risk review, installed list and enable controls | `AT-EXTENSION-004`, `AT-UI-001` | Completed |
| `EXT-INSTALL-QA-001` | `NFR-RELIABILITY-001`, `NFR-UI-001` | Focused tests, responsive states, and repository gates | `AT-EXTENSION-004`, `CT-RELIABILITY-001`, `AT-UI-001` | Completed |

## Acceptance and evidence

- Previewing a valid static/process/service/external Manifest v2 package executes nothing and returns only bounded safe metadata plus integrity.
- Invalid/v1/missing/escaping packages fail before trust or persistence; a changed package fails an old confirmation.
- Confirmed install persists the linked root and desired enabled state; restart restores only an exact integrity match.
- Missing or changed sources remain visible with an actionable error and no eligible tools or process start.
- Disable and uninstall stop the runtime; uninstall does not delete user files. Repeated operations are idempotent or return actionable conflicts.
- The desktop covers empty, selecting, previewing, executable warning, installing, success, validation error, restart-error, disabled, and narrow layouts with keyboard-accessible controls.
- Schema v2-to-v3 migration creates/verifies a backup, is transactional/idempotent, and documents downgrade restoration.
- Applicable gates are `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, `pnpm build`, targeted Electron/renderer tests, and `git diff --check`.

Implementation evidence recorded on 2026-08-06: Electron exposes a directory-only native picker; the desktop extension tab provides empty, preview, executable-authority warning, update, running, disabled/error, and uninstall states without raw path or command entry. Core canonicalizes the selected root, validates Manifest/API `2/2` without execution, rejects symbolic links and packages over 4,096 files or 64 MiB, hashes the complete package, and re-hashes at install and restart. Focused tests prove arbitrary unreferenced source drift invalidates confirmation, persistence/restoration works, source drift disables eligibility, uninstall preserves source, and schema v2-to-v3 creates and verifies its backup. Manual local acceptance installed `examples/extensions/ui-test` from the 640 px and default-width UI, observed its risk/capability preview and running list state, restarted the full development instance, and observed the persisted extension restore as enabled and `ready`. `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, `pnpm build`, targeted Node/Go tests, and `git diff --check` pass.

## Security and data lifecycle

The selected local path is private user data shown only in the native management UI and persisted locally; it is not sent to models, logs, analytics, or extension Web content. The persisted integrity is a non-secret trust binding. Aivo stores no extension credentials in this record. Runtime payloads retain existing bounds and teardown. Migration fixtures use synthetic paths and manifests.

## Compatibility and migration

Fresh databases start at schema v3. Existing schema v2 databases receive a verified `.v2.bak` before the transactional table addition. Schema v3 data is not read by older binaries; downgrade requires closing Aivo and restoring the verified v2 backup. Existing built-ins and already running in-memory extension APIs remain valid, but only installations created through the new confirmation flow are restored after restart.

## Bug root cause (type=bug only)

N/A.
