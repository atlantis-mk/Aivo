# AI documentation, Work, and release governance

This file is the canonical owner of Aivo documentation governance. The system optimizes for compact current truth and executable constraints; Git owns ordinary change history.

## 1. Document responsibilities

- `AGENTS.md`: short execution-critical rules and prohibitions.
- `docs/`: current product, scope, Requirements, architecture, data, security, global test policy, and generated Traceability.
- `specs/`: focused current behavior that a Requirement and code contract cannot express clearly.
- `adr/`: only significant durable decisions and rejected alternatives that code cannot explain.
- `changes/<WORK-ID>/change.yaml`: temporary cross-task state or controlled-boundary coordination.
- `changes/archive.json`: compatibility registry for legacy sealed Work only.
- `releases/vX.Y.Z.md` plus the matching Git tag: delivered versions and release evidence.

Requirements own current behavior, code owns exhaustive contracts, tests own executable acceptance, ADRs own exceptional reasoning, and Git owns ordinary history. Do not copy those facts into Work.

## 2. Governance lanes

### Direct is the default

Use Direct change for ordinary features, behavior changes, specification edits, bugs with a clear root cause, refactoring, UI work, tests, developer tooling, and reversible dependency-free improvements that can finish in the current task. Update the owning current spec and tests together when behavior changes. No Work, approval state, evidence file, or archive record is created.

### Work is the exception

Create Work only when at least one condition applies:

- Unfinished context, risks, or next actions must survive across tasks.
- An open product or architecture decision needs explicit user approval before implementation.
- Security/trust, secrets, data ownership, persistence/schema, public API/RPC/IPC, compatibility/migration, production dependency/license, irreversible behavior, platform/scope, or release/rollback boundaries change.
- A severe, recurring, security/data-loss, or unclear-root-cause bug needs durable coordination.
- Cross-module/platform/version work cannot be completed and verified as one coherent task.

A behavior or documentation change alone does not justify Work. Significant persistence, privilege, public-contract, credential, plugin/MCP trust, sandbox/authorization, platform, or irreversible decisions also require an ADR. Never combine unrelated work.

## 3. New Work schema

Schema-v2 Work is one `change.yaml` containing identity, goal, current state, routing, controlled boundaries, risks, and next actions. It has no mandatory body, task table, acceptance prose, command transcript, evidence JSON, profile, specification-delta field, or per-Work hash.

```yaml
schema: "2"
id: "CHG-YYYY-NNN-change-name"
title: "Change title"
type: "feature"
status: "Draft"
spec_revision: "0.1.4-active"
target_release: null
goal: "Why this state must survive the current task"
requirements: []
tests: []
adrs: []
context_refs: []
related_changes: []
boundaries: []
risks: []
next: []
```

Use an ADR, focused spec, or optional design document for complex durable reasoning; do not inflate Work YAML into a second specification.

## 4. Context routing

- `requirements`, `adrs`, `context_refs`, and `related_changes` are the initial reading set.
- `context_refs` uses repository-root-relative paths and optional `#<heading or stable ID>` selectors; it never points to another Work.
- A search hit, shared ID, path, or module is not a dependency. Read undeclared Work only to resolve a concrete conflict or controlled boundary.
- Known Work ID: read its YAML and optional legacy body, then one declared routing layer. Unknown ID: start at `docs/00-spec-index.md` and search filenames/YAML metadata only.
- Archived and Done Work is history, not default current context.

## 5. Lifecycle

New Work uses `Draft -> Active -> Done`; a proposal may end at `Rejected`.

- `Draft`: discovery and proposal; controlled implementation has not started.
- `Active`: `pnpm work:start -- <WORK-ID>` records the explicit decision to proceed and refreshes Traceability.
- `Done`: `pnpm work:finish -- <WORK-ID>` runs applicable checks, changes only the status, refreshes Traceability, and validates the result.

Command results remain in the task/CI output and eventual Git history. New Work creates no `verification.json` or archive digest. Once a Done Work is committed, Git preserves it; a later cleanup may delete the small YAML in a separate reviewable commit when nothing references it.

## 6. Legacy compatibility

Work without `schema: "2"` keeps the lifecycle and fields of the governance revision under which it was accepted. Existing Draft, Accepted, Implementing, Verified, Released, and Rejected states remain readable. Verified, Released, and Rejected legacy Work still requires its historical archive entry, and every existing archive directory/digest remains permanently immutable.

Do not migrate or rehash legacy sealed Work. Active legacy Work may finish through the compatible `work:finish` path, which records and seals it according to its original contract.

## 7. Generated Traceability

`docs/03-functional-requirements.md` owns Requirement text and Test IDs. Work YAML owns temporary routing. `pnpm docs:trace` generates active Work and completed evidence into `docs/08-traceability.md`; `pnpm docs:check` fails when it is stale.

Traceability is an index, not an evidence store. Feature-specific acceptance lists and command results do not belong there.

## 8. IDs, current truth, and history

- Work, Requirement, Test, ADR, and OPEN IDs are never reused or redefined.
- Current specs describe only current accepted behavior; Git shows how they changed.
- Superseded Requirements keep a short replacement/removal pointer only when current compatibility requires it.
- Git commits and tags replace narrative process archives for ordinary changes.
- Protected branches and signed release tags should be enabled before legacy archive compatibility is considered removable.

## 9. Release minimum

A Release records version, date, matching Git tag, component/schema/contract builds, delivered Done or legacy sealed Work when applicable, compatibility/migration, platform evidence, known issues, rollback limits, and compensation. It never redefines a Requirement.

Release readiness is an explicit operator decision. Documentation and test evidence remain reviewable guidance but do not authorize or block publication. Publication automation may still fail closed when it cannot bind the version, build required artifacts, preserve immutable bytes, or publish a self-consistent update contract.

## 10. Completion gates

- `pnpm docs:trace` produces no diff and `pnpm docs:check` validates schemas, routing, IDs, commands, generated Traceability, releases, and historical archives.
- `pnpm scripts:test` applies when governance or release scripts change.
- `pnpm test:core`, `pnpm lint`, and `pnpm build` apply to their changed runtime surfaces.
- Migration, security, public-contract, dependency, platform, and release-mechanism checks remain fail-closed for the Work that changes them.
