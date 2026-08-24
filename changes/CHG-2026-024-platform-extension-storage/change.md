# Move managed extensions into platform application data

## Problem or goal

The managed installer currently stores packages in `~/.aivo/extensions`, which is technically private but unusually visible beside the legacy database. The user selected a Chrome-style application-data hierarchy so ordinary users do not encounter or edit managed packages during normal home-folder navigation.

## Expected behavior

Default Aivo installations use the platform application-data root with a Chrome-like `Aivo/Default/Extensions/<id>/<integrity>` layout. On macOS this resolves to `~/Library/Application Support/Aivo/Default/Extensions`. Runtime and persistence use only the new path. An exact existing package beneath `~/.aivo/extensions` is copied and reverified into the new root during startup, its row is updated, and only then is its old Aivo-owned directory removed.

## Non-goals

This Work does not claim that path obscurity is a security boundary, move the database or credentials, add multiple Aivo profiles, change package format, add signing, or hide installed paths from the owning operating-system user.

## Impact

The default persistence store selects a platform application-data package root. Explicit database stores used by tests and isolated runtimes retain a database-sibling root. Core recognizes only the exact former database-sibling managed layout as a migration source. No schema version, RPC, renderer, Manifest, View, tool, or provider contract changes.

## Implementation constraints

ADR-0010 owns the location decision. The new root remains mode `0700`; published generations remain read-only and integrity checked. Migration accepts only a validated `managed` record whose path exactly matches `<legacy-root>/<id>/<integrity>`, performs the existing staged copy and atomic publication flow, commits the new path before cleanup, and never recursively deletes an unresolved or user-selected path.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `EXT-LOCATION-DOC-001` | `REQ-EXTENSION-004` | Accepted platform application-data location and migration contract | `AT-EXTENSION-004` | Completed |
| `EXT-LOCATION-ROOT-001` | `REQ-EXTENSION-004`, `NFR-SECURITY-001` | Chrome-style default root with isolated-store compatibility | `AT-EXTENSION-004`, `CT-SECURITY-001` | Completed |
| `EXT-LOCATION-MIGRATE-001` | `NFR-RELIABILITY-001` | Verified old-root copy, row switch, and scoped old-root cleanup | `CT-RELIABILITY-001`, `CT-SECURITY-001` | Completed |
| `EXT-LOCATION-QA-001` | `NFR-RELIABILITY-001` | Focused migration tests, real development migration, and repository gates | `AT-EXTENSION-004`, `CT-RELIABILITY-001` | Completed |

## Acceptance and evidence

- macOS default storage resolves to `~/Library/Application Support/Aivo/Default/Extensions/<id>/<integrity>`.
- Default storage uses the platform configuration/application-data base on Windows and Linux with the same `Aivo/Default/Extensions` suffix.
- Fresh installs publish only beneath the new root and retain private/read-only/integrity protections.
- Exact legacy managed rows migrate through copied-package verification before the database path changes; cleanup happens only after persistence succeeds.
- Missing, changed, or out-of-root legacy packages remain disabled and are never deleted.
- Explicit temporary database stores remain isolated beside their database so automated tests never write to the user's application-data directory.
- Applicable gates are focused Go tests, `pnpm docs:check`, `pnpm test:core`, `pnpm lint`, `pnpm build`, and `git diff --check`.

Verification evidence on 2026-08-06:

- Focused tests passed platform-root resolution, explicit-store isolation, managed installation, linked-row promotion, and exact former-root relocation with original-source preservation and old-owned-directory cleanup.
- `pnpm test:core` passed every Go package.
- `pnpm lint` passed with only the repository's existing Fast Refresh export warnings in shared UI and root route files.
- `pnpm build` passed TypeScript compilation and the Vite production build.
- `pnpm docs:check` and `git diff --check` passed.
- The real development installation moved from `~/.aivo/extensions/<id>/<integrity>` to `~/Library/Application Support/Aivo/Default/Extensions/<id>/<integrity>`, remained enabled and ready, retained its integrity/read-only modes, updated the persisted path, and removed the now-empty former extension root.

## Security and data lifecycle

The deeper path reduces accidental discovery but is not access control. The owning OS user can still locate it. Mode restrictions, read-only generations, integrity verification, trust binding, and exact-root deletion checks remain the security controls. The former package directory is removed only after an exact owned-path migration succeeds.

## Compatibility and migration

Schema remains v4. Downgrade after filesystem migration may require reinstalling or copying a package back because older binaries expect the database-sibling managed root. The existing v3 database backup remains available for full schema downgrade, but does not contain package bytes.

## Bug root cause (type=bug only)

N/A.
