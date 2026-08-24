# Copy installed extensions into Aivo-managed storage

## Problem or goal

The first visual installer retained a reference to the user-selected development directory. That makes installed behavior depend on a mutable source tree and differs from the managed-copy lifecycle users expect from Chrome-style extension installation. Aivo must own an immutable installed copy while leaving the selected source untouched.

## Expected behavior

After native preview and exact confirmation, Core copies the complete bounded Manifest v2 package into Aivo-managed private storage, revalidates the copied integrity, and atomically publishes the confirmed generation. Runtime discovery, restart restoration, Views, and tools use only that managed path. Changing, moving, or deleting the original source after successful installation has no effect. Update installs a newly confirmed managed generation. Uninstall removes the managed package and record but never the original selected source.

## Non-goals

No Chrome Web Store, CRX/ZIP installer, remote registry, signature/notarization claim, dependency installation, content scripts, automatic source watching, silent update, model-initiated install, cross-device synchronization, or arbitrary Chrome extension compatibility.

## Impact

Core gains bounded package copying, staging, verification, atomic publication, linked-record promotion, and managed cleanup. SQLite schema v4 distinguishes historical `linked` rows from `managed` rows. The desktop copy explains source versus managed package ownership. No production dependency, provider, MCP, prompt, tool protocol, View protocol, or platform scope changes.

## Implementation constraints

ADR-0002 and ADR-0009 own trust, files, and persistence. Copying cannot start executable code. The destination must remain beneath the exact Host-owned extension root, reject symlinks and unsupported files, preserve required executable modes, and reuse the existing file/byte bounds. Core must reload and compare the copied package before atomic publication. Failed staging, verification, persistence, enablement, migration, or uninstall must not expose a partial generation or delete user source. Old generations may remain for rollback while running and are collected only after safe restart or uninstall. Models, renderer content, and extension Web content cannot choose destinations or mutate managed packages.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `EXT-MANAGED-DOC-001` | `REQ-EXTENSION-004` | Accepted managed-copy Work, ADR, Requirement, data/security/test/traceability contract | `AT-EXTENSION-004` | Completed |
| `EXT-MANAGED-STORE-001` | `REQ-EXTENSION-004`, `NFR-RELIABILITY-001` | Schema v4 mode, v3 backup, managed root and linked-row promotion | `AT-EXTENSION-004`, `CT-RELIABILITY-001` | Completed |
| `EXT-MANAGED-COPY-001` | `REQ-EXTENSION-004`, `NFR-SECURITY-001` | Bounded copy, copied-integrity verification, atomic publication and managed uninstall | `AT-EXTENSION-004`, `CT-SECURITY-001` | Completed |
| `EXT-MANAGED-DESKTOP-001` | `REQ-EXTENSION-004`, `NFR-UI-001` | Source/managed ownership and uninstall copy in the visual flow | `AT-EXTENSION-004`, `AT-UI-001` | Completed |
| `EXT-MANAGED-QA-001` | `NFR-RELIABILITY-001`, `NFR-UI-001` | Focused migration/copy/failure tests, manual responsive acceptance and repository gates | `AT-EXTENSION-004`, `CT-RELIABILITY-001`, `AT-UI-001` | Completed |

## Acceptance and evidence

- Preview remains non-executing and a changed source still rejects an old confirmation.
- Confirmation copies every validated package file into a private Aivo directory, rejects symlinks/limits/unsupported files, reloads the copy, and publishes only an exact integrity match.
- Moving, editing, or deleting the original source after install does not change or disable the managed installation.
- Restart restores only the persisted managed generation and safely removes abandoned staging directories; it never falls back to the original source.
- Update publishes a new exact generation without exposing partial files; failed update retains a safe actionable state.
- Disable stops runtime without deleting the package. Uninstall stops runtime, removes all owned generations and the row, and preserves the user-selected source.
- Exact unchanged schema v3 linked rows promote to managed copies. Missing, changed, invalid, or failed rows remain disabled and visible for explicit reinstall.
- Schema v3-to-v4 creates/verifies a backup, is transactional/idempotent, and documents downgrade restoration.
- Applicable gates are focused Go/Node/renderer tests, `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, `pnpm build`, manual wide/narrow acceptance, and `git diff --check`.

Verification evidence on 2026-08-06:

- Focused Core tests passed managed-copy installation, source independence, managed-package tamper refusal, verified linked-row promotion, source preservation, owned-path uninstall, out-of-root deletion refusal, persistence round trip, schema v1/v2/v3 migration backup, and invalid-backup refusal cases.
- `pnpm test:core` passed every Go package, including deterministic teardown of retired extension generations.
- `pnpm scripts:test` passed all archive, desktop extension installation/runtime, settings/search, permission, activation-scope, and example extension tests.
- `pnpm lint` passed with only the repository's existing Fast Refresh export warnings in shared UI and root route files.
- `pnpm build` passed TypeScript compilation and the Vite production build.
- `pnpm docs:check` and `git diff --check` passed.
- In-app browser QA at 1280 x 800 and 390 x 844 showed the managed badge, managed installation path, native installation ownership copy, controls, and truncation without horizontal overflow or browser errors.
- A real schema v3 development database migrated to v4, its existing extension row promoted from a user source path to `~/.aivo/extensions/<id>/<integrity>`, its verified `.v3.bak` retained schema v3, and the published generation/root contained no owner-writable file or directory.

## Security and data lifecycle

The source path is transient native installation input and is not retained after a successful managed install. Managed packages are private local application data and contain user-supplied executable code, so their paths and contents never enter model context, logs, diagnostics, analytics, or extension Web content. Uninstall may delete only a validated descendant of the Host-managed extension root. Staging and abandoned generations have deterministic cleanup.

## Compatibility and migration

Fresh databases start at schema v4. Existing schema v3 databases receive a verified `.v3.bak` before `install_mode` is added with historical rows marked `linked`. Startup promotes only exact confirmed linked content and rewrites the row to the managed path/mode after copied-integrity verification. Older binaries do not understand schema v4; downgrade requires closing Aivo and restoring the verified v3 backup. Schema v2 databases can migrate directly through the current migration transaction and receive the applicable pre-migration backup.

## Bug root cause (type=bug only)

N/A.
