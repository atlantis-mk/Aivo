# Adopt AI document governance

> Documentation proportionality: this is a full governance Work because it changes repository-wide authority, routing, history, and release rules.

## Problem or goal

Aivo has a useful v2 preparation package and supporting operational documents but no single current-spec index, stable Requirement/Test traceability, routed Work metadata, ADR template, Release record, or machine-enforced completed-Work integrity. Adopt the proven VaultMesh governance pattern while adapting it to Aivo's Electron/Go architecture and retaining existing evidence.

## Expected behavior

- `AGENTS.md` defines context routing, risk-based Work thresholds, authority, boundaries, and completion rules.
- `docs/00-spec-index.md` routes agents to current primary specs without duplicating Work state.
- The numbered primary documents from 01 through 09 own current product, scope, Requirements, architecture, data, security, tests, Traceability, and governance.
- Work, ADR, and Release templates create stable, proportional records.
- Documentation checks validate routes, IDs, traceability, commands, sealed Work, and Release references.
- Completed Work is sealed with a complete file set and SHA-256 manifest and becomes permanently read-only.

No product Requirement changes; this Work is repository governance.

## Non-goals

This Work does not resolve open Aivo v2 product decisions, rewrite existing product code, migrate persistence, release a build, or delete/restructure `openspec/changes/aivo-v2`.

## Impact

Documentation, repository scripts, root package commands, and CI change. Renderer, Electron runtime, Go runtime, local API, persistence schema, provider credentials, plugins/MCP, and packaged behavior do not change.

## Implementation constraints

- Preserve current Aivo-specific repository rules and existing evidence.
- Adapt names, architecture, commands, source paths, archive environment variable, and release fields; do not copy VaultMesh product assumptions.
- Use only Node standard-library dependencies for governance scripts.
- A check failure must identify the category and file/ID involved.
- Seal this Work only after all governance checks and script tests pass; sealing is the final mutation under this directory.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| GOV-001 | N/A | Aivo primary spec set and routing constitution | N/A | Complete |
| GOV-002 | N/A | Work/ADR/Release templates and archive manifest | N/A | Complete |
| GOV-003 | N/A | Documentation and archive validation scripts | N/A | Complete |
| GOV-004 | N/A | Package commands, README, and CI integration | N/A | Complete |
| GOV-005 | N/A | Passing validation and sealed Work evidence | N/A | Complete |

## Acceptance and evidence

- `pnpm docs:check` passed on 2026-07-31: 22 Markdown files, 2 Work YAML files including the template, 10 Requirements, 10 Test IDs, 1 routed Work, and no validation errors before sealing.
- `pnpm scripts:test` passed on 2026-07-31: 4/4 tests covered sealing, tamper detection, duplicate/incomplete refusal, trusted Git-baseline immutability, and an initial registry whose baseline predates the manifest.
- `git diff --check` passed on 2026-07-31.
- `pnpm test:core`, `pnpm lint`, and `pnpm build` are not required because product code is unchanged; CI continues to own them for product/release changes.
- Status is `Verified`; `pnpm work:archive -- CHG-2026-001-document-governance` is the final mutation for this directory.

## Security and data lifecycle

No runtime data or secret path changes. Archive records contain repository-relative filenames, status, UTC timestamp, and SHA-256 digests of Work documentation only. Scripts reject symlinks inside sealed Work.

## Compatibility and migration

Existing `openspec/changes/aivo-v2` remains unchanged and is referenced as preparation evidence. Existing supporting docs remain available. New Work begins at `0.1.0-active`; there are no historical completed Work Packages to bulk-migrate.

## Bug root cause (type=bug only)

N/A.
