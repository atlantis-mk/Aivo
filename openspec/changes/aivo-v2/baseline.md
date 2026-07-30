# Aivo v1 baseline

## Protected source snapshot

- Archive branch: `codex/aivo-v1-archive`
- Archive tag: `aivo-v1-archive-2026-07-31`
- Archive commit: `bb528c8` (`Archive pre-v2 workspace state`)
- Upstream base before the archive: `75ac8a7` on `main`
- v2 branch: `codex/aivo-v2`, created from the archive commit

The archive commit contains the complete workspace state that existed before
v2 preparation: tracked modifications, untracked additions, and tracked
deletions. The snapshot changed 283 files with 11,630 insertions and 17,699
deletions relative to its upstream base.

The branch and tag are local until they are deliberately pushed to a remote.
No v2 preparation file is part of the archive commit.

## Restore procedures

Inspect the old version without changing the current branch:

```bash
git show aivo-v1-archive-2026-07-31
```

Create a new recovery branch from the immutable archive tag:

```bash
git switch -c recovery/aivo-v1 aivo-v1-archive-2026-07-31
```

Do not force-move or delete the archive branch or tag during v2 development.

## Toolchain baseline

- Node.js: `v24.12.0`
- pnpm: `11.1.1`
- Go: `go1.26.0 darwin/amd64`
- Desktop package version: `0.0.0`

## Verification baseline

Recorded on 2026-07-31 from commit `bb528c8`:

| Check | Result | Notes |
| --- | --- | --- |
| `pnpm test:core` | Pass | All Go packages pass; `core/domain` has no test files. |
| `pnpm lint` | Pass with warnings | 15 existing Fast Refresh warnings, mainly in shared UI primitives. |
| `pnpm build` | Pass with warnings | Vite reports large barrel-module and chunk-size warnings. |

These warnings are baseline debt, not v2 regressions. New warnings must not be
introduced. The shared files under `apps/desktop/src/components/ui` remain
read-only under the repository rules.

## Known baseline risks

- The largest generated renderer chunk is approximately 1.8 MB before gzip;
  v2 should introduce route or feature-level code splitting where useful.
- Persistence reports schema version 1 while relying heavily on GORM
  `AutoMigrate`; v2 migrations need explicit, testable version transitions.
- The existing transport is method-oriented RPC over local HTTP. New public
  contracts should have versioned resource boundaries without breaking the
  desktop during migration.
- The settings route currently provides navigation shell but no substantive
  settings content.
- The built-in browser implementation was removed in the archived workspace;
  reintroducing browsing is therefore an explicit v2 product decision.
