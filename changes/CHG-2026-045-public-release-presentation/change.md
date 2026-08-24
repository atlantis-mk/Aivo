# Present public releases for end users

## Problem or goal

The GitHub Release body currently exposes an internal verification record as its primary content. Present releases with a descriptive bilingual title, user-facing highlights, a system/chip/format download table, integrity guidance, and concise installation notes while retaining governance and verification details.

## Expected behavior

`NFR-RELEASE-001` owns the release contract. GitHub uses the first H1 in the same-name Release record as its title and the complete record as its body. The public sections come first; internal compatibility, verification, and known-issue evidence remain available afterward.

## Non-goals

Renaming immutable v0.1.0 assets, changing R2 keys, adding electron-updater YAML, republishing package bytes, signing packages, or changing automatic-update trust are excluded.

## Impact

Release Markdown, the tagged workflow title derivation, focused publication tests, and the existing v0.1.0 GitHub Release metadata change. Product code, packages, digests, R2 manifests, schemas, APIs, credentials, and signing are unchanged.

## Implementation constraints

The same checked-in Release record remains the single notes owner. Workflow title extraction must accept exactly one non-empty first H1 and fail closed otherwise. Direct download links must use the matching tag and existing digest-bound asset names. Unsigned-package warnings remain visible.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| RELEASE-UI-01 | NFR-RELEASE-001 | User-facing Release template and v0.1.0 notes | CT-RELEASE-001 | Complete |
| RELEASE-UI-02 | NFR-RELEASE-001 | Workflow derives the GitHub Release title from the Release record | CT-RELEASE-001 | Complete |
| RELEASE-UI-03 | NFR-RELEASE-001 | Existing v0.1.0 GitHub Release metadata updated without asset mutation | CT-RELEASE-001 | Complete |

## Acceptance and evidence

- Tests cover title extraction and notes-file ownership.
- `pnpm docs:check` and `pnpm scripts:test` pass.
- GitHub v0.1.0 retains the exact seven existing assets and digests while its name/body match the checked-in record.

Verified on 2026-08-25: documentation checks passed across 93 Markdown and 47 YAML files; all script and desktop tests passed; the GitHub API confirmed the Release title/body match `releases/v0.1.0.md`; and before/after asset comparison confirmed all seven names, sizes, and SHA-256 digests are unchanged.

## Security and data lifecycle

No secrets, private data, downloads, or package bytes are added. Release text references public GitHub assets and the existing checksum manifest.

## Compatibility and migration

No product compatibility or migration impact. Existing download URLs, R2 keys, hashes, and automatic-update metadata remain unchanged.
