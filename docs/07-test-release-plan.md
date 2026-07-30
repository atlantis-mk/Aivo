# Aivo test and release plan

## Test layers

| Layer | Purpose | Primary command or evidence |
| --- | --- | --- |
| Documentation | Routing, IDs, traceability, commands, and immutable Work archives | `pnpm docs:check` |
| Governance scripts | Work archive behavior and tamper detection | `pnpm scripts:test` |
| Go unit/integration | Domain, services, transport, persistence, concurrency, providers, tools | `pnpm test:core` |
| Desktop static/build | Type safety, lint, renderer/main/preload build | `pnpm lint`, `pnpm build` |
| Provider diagnostics | Configured provider auth and backend behavior | provider smoke command in `docs/provider-backend.md` |
| UI acceptance | Wide/narrow layout, keyboard, loading/error/cancel/permission behavior | Screenshots and `docs/opencode-replacement-manual-acceptance.md` as applicable |
| Package smoke | Installer contents, launch, bundled core, health | `pnpm smoke:release` after platform package command |

## Work verification

- Run focused tests while implementing, then all applicable repository gates.
- New or changed Requirements need stable Test IDs and an updated row in `08-traceability.md`.
- A Bug Work records a reproduction test that fails before the fix and passes after it.
- Failure, cancellation, repeated execution, timeout, dependency loss, teardown, compatibility, migration, rollback, and security paths are tested or explicitly marked N/A in the Work.
- UI Work includes wide and narrow screenshots when behavior or layout changes.
- `Verified` requires command/platform evidence in `change.md`, followed immediately by Work archive sealing.

## Release gates

| Gate | Requirement |
| --- | --- |
| GATE-1 | `pnpm docs:check` and `pnpm scripts:test` pass |
| GATE-2 | `pnpm test:core` passes |
| GATE-3 | `pnpm lint` and `pnpm build` pass without new warnings |
| GATE-4 | Applicable provider and end-to-end/manual acceptance passes |
| GATE-5 | Target-OS package and `pnpm smoke:release` pass |
| GATE-6 | Migration backup, forward migration, failure recovery, and rollback pass when data changes |
| GATE-7 | Release record references only sealed Work and the same-name Git tag is created |

macOS signing/notarization and platform packaging details remain in `docs/release-quality.md`. Cross-compilation cannot satisfy target-OS package acceptance.

## CI archive baseline

CI must set `AIVO_ARCHIVE_BASE_REF` to a trusted target-branch commit or push-before commit. This prevents a single commit from rewriting a sealed Work and recalculating its digest. A local check defaults to `HEAD`.
