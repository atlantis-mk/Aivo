# Aivo test and release plan

This document owns global test layers and release policy. Requirement-specific acceptance belongs with the Requirement and executable tests; command results stay in task/CI output rather than Work documents.

## Test layers

| Layer | Purpose | Primary command or evidence |
| --- | --- | --- |
| Documentation | Schema, routing, IDs, generated Traceability, commands, and immutable archives | `pnpm docs:trace`, `pnpm docs:check` |
| Governance and release scripts | Work lifecycle, archive, packaging, publication, and policy behavior | `pnpm scripts:test` |
| Go unit/integration | Domain, application, transport, persistence, concurrency, providers, tools, and lifecycle | `pnpm test:core` |
| Desktop static/build | Type safety, lint, renderer/main/preload build | `pnpm lint`, `pnpm build` |
| Focused acceptance | Requirement behavior, regression, failure, cancellation, repetition, timeout, cleanup, and security paths | Stable `AT-*` and `CT-*` IDs in `docs/03-functional-requirements.md` |
| UI acceptance | Changed wide/narrow layout, keyboard, loading, empty, error, permission, overflow, and cancellation states | Focused checks and screenshots in the owning Work |
| Package and platform | Installer contents, launch, bundled Core, signing/notarization, and OS integration | Target-OS package command and `pnpm smoke:release` |
| Release publication | Version/tag binding, complete artifacts, immutable R2 objects, digest agreement, stable-manifest ordering, and GitHub assets | Release tests plus the operator-triggered native workflow |

## Work verification

- Run focused tests while implementing, then every repository gate selected by the changed surfaces.
- New or changed behavior updates its stable Requirement and Test ID directly in the same task; ordinary behavior changes do not require Work. Run `pnpm docs:trace` instead of editing Traceability.
- Bug Work records a reproduction that fails before the fix and passes after it, plus root cause and why prior tests missed it.
- Cover applicable success, boundary, refusal, failure, cancellation, repetition, timeout, dependency loss, teardown, compatibility, migration/rollback, security, and UI states.
- Schema-v2 `pnpm work:finish -- <WORK-ID>` runs applicable checks, marks Done, and refreshes Traceability without copying command evidence or creating hashes.
- Legacy Work retains its historical verification and archive contract.
- Target-OS packaging, signing, notarization, publication, and updater-channel checks are Release evidence unless the Work changes that mechanism.

## Release gates

The checks below are an operator-owned verification checklist, not blockers in the stable publication workflow. The operator decides when evidence is sufficient and may trigger publication directly. Native packaging plus tag/version, complete-asset, immutable-object, digest, and stable-manifest ordering/readback checks remain executable requirements because they produce and protect published artifacts.

| Gate | Requirement |
| --- | --- |
| GATE-1 | `pnpm docs:check` and `pnpm scripts:test` pass when governance or publication scripts change |
| GATE-2 | `pnpm test:core` passes when Core behavior changes |
| GATE-3 | `pnpm lint` and `pnpm build` pass when desktop behavior changes |
| GATE-4 | Applicable provider, UI, extension, and end-to-end acceptance passes |
| GATE-5 | Target-OS package and `pnpm smoke:release` pass |
| GATE-6 | Migration backup, forward migration, failure recovery, and rollback pass when data changes |
| GATE-7 | Release record references only schema-v2 Done or legacy sealed Work and the same-name Git tag exists |
| GATE-8 | R2 and GitHub assets share normalized names, sizes, and SHA-256 digests; the stable manifest is published last and verified by readback |
| GATE-9 | Packaged update check/download/handoff passes on each supported target; unsigned artifacts remain explicit and non-silent |

macOS signing/notarization and packaging details remain in `docs/release-quality.md`. Cross-compilation cannot satisfy target-OS acceptance.

## CI archive baseline

CI must set `AIVO_ARCHIVE_BASE_REF` to a trusted target-branch or push-before commit. Local checks default to `HEAD`; sealed history remains immutable even when governance schemas evolve.
