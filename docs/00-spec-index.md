# Aivo specification index

## Current baseline

- Specification revision: `0.1.4-active`
- Product version: `0.2.1`
- Desktop shell: Electron + Vite/React
- Local runtime: Go
- Persistence schema: v1 baseline; explicit v2 transitions are required before schema changes
- Status: Aivo v2 preparation is active; P0 product decisions remain open
- Last updated: `2026-08-28`

## Routing order

1. Read `../AGENTS.md` first.
2. With a known Work ID, read its `change.yaml` and `change.md`, then follow only its declared routing metadata.
3. Without a Work ID, use this index and search `../changes/*/change.yaml` metadata to locate candidates.
4. Read primary specs, focused specs, ADRs, and Traceability by matching heading or stable ID. Expand context for high-risk or conflicting areas as required by `../AGENTS.md`.

## Document map

| File | Single responsibility | Status |
| --- | --- | --- |
| `01-product-definition.md` | Product promise, users, outcomes, and non-goals | Active |
| `02-scope-matrix.md` | Required/Partial/Future/Out boundaries and open decisions | Active |
| `03-functional-requirements.md` | Stable, testable product and non-functional behavior | Active |
| `04-architecture.md` | Ownership, dependency direction, and runtime data flow | Active |
| `05-data-model.md` | Persistence ownership, entities, schema, and migration rules | Active |
| `06-security-privacy.md` | Trust boundaries, secrets, logging, and privacy | Active |
| `07-test-release-plan.md` | Test layers, commands, and release gates | Active |
| `08-traceability.md` | Generated Requirement, test, ADR, active Work, and sealed-evidence index | Generated |
| `09-document-governance.md` | Work, ADR, release, routing, and archive rules | Active |

## Supporting and focused documents

- `provider-backend.md`: provider registry, authentication, runtime policy, and diagnostics
- `runtime-configuration.md`: project runtime configuration behavior
- `release-quality.md`: packaging, signing, and release operations
- `opencode-replacement-manual-acceptance.md`: retained manual acceptance procedure
- `../openspec/changes/aivo-v2/`: v2 baseline, proposal, inventory, migration plan, and decision evidence
- `../specs/README.md`: focused-spec ownership and creation rules

## Work, ADR, and Release entry points

- `../changes/README.md`
- `../changes/_template/`
- `../changes/archive.json`
- `../adr/_template.md`
- `../releases/README.md`
- `../releases/_template.md`

Work status is owned only by each `change.yaml`; run `pnpm docs:trace` to refresh the generated Traceability index. This index does not duplicate active or historical Work status.

## Authoritative code locations

| Contract or behavior | Implementation location |
| --- | --- |
| Domain models and lifecycle | `../core/domain/` |
| Application orchestration | `../core/app/` |
| Persistence | `../core/infra/persistence/` |
| Local HTTP transport | `../core/internal/transport/http/` |
| Core CLI entry point | `../core/cmd/aivo-core/` |
| Electron main and preload | `../apps/desktop/electron/` |
| Renderer routes and features | `../apps/desktop/src/routes/`, `../apps/desktop/src/features/` |
| Generated desktop/core bindings | `../apps/desktop/bridge/`, `../apps/desktop/bridge/go/` |
| Build and packaging automation | `../scripts/` |

## Open decisions

| ID | Question | Blocked area |
| --- | --- | --- |
| `OPEN-001` | Primary v2 user and top three jobs-to-be-done | Product definition and prioritization |
| `OPEN-002` | Primary launch object and relation among conversation, task, run, and terminal | Information architecture and contracts |
| `OPEN-003` | Exact v1 data compatibility promise and v1/v2 coexistence strategy | Persistence migration and rollout |
| `OPEN-004` | First releasable v2 slice | Delivery ordering |
| `OPEN-005` | Whether the built-in browser remains removed | Product scope |

Agents may work around these decisions but must not close them without an accepted Work that updates the owning specification.
