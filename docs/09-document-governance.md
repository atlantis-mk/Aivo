# AI documentation, Work, and release governance

This file preserves only what an agent needs to implement, verify, and evolve Aivo correctly without chat history. Meeting notes, staffing, time tracking, and performance reporting do not belong in this system.

## 1. Document responsibilities

- `AGENTS.md`: highest-priority execution rules and prohibitions.
- `docs/`: current product, scope, Requirements, architecture, data, security, test, and traceability truth.
- `specs/`: focused cross-module behavior that cannot be expressed by a Requirement and code contract alone.
- `adr/`: significant technical reasons and rejected alternatives that current code cannot explain.
- `changes/<WORK-ID>/`: one change's increment, tasks, tests, and evidence.
- `changes/archive.json`: immutable file set, digest, and sealing time for completed Work.
- `releases/vX.Y.Z.md` plus Git tag: what was actually delivered, compatibility, and historical snapshot.
- `openspec/changes/aivo-v2/`: retained preparation and migration evidence; accepted current behavior must be promoted into the primary specs above.

One rule has one primary specification owner. Work, Traceability, and Releases reference behavior; they do not restate it.

## 2. Work proportionality and creation threshold

Work is required when a product decision, contract, risk, migration, or cross-task coordination must be preserved, not because of change category or diff size.

A change may proceed without Work only when it is inside accepted scope and behavior, local and easily reversible, introduces no product or architecture choice, crosses no high-risk or public contract boundary, and can be fully implemented and verified in the current task. Direct changes may add or update focused regression tests, fixtures, or snapshots that prove an existing expectation; test changes do not automatically require Work.

Copy/visual polish, restoration of accepted responsive or accessibility behavior, an ordinary bug with a clear local root cause, internal refactoring, type/null fixes, test strengthening, non-release development tooling, and local semantics-preserving performance improvements may qualify. This is not a whitelist.

Create or reuse Work when any of these applies:

- A primary spec, Requirement, Scope, ADR, product decision, or security decision changes.
- Security/trust, secrets, data ownership, persistence/schema, public API/RPC/IPC, compatibility/migration, platform/scope, release/rollback, or irreversible behavior changes.
- Production dependencies or licenses, or cross-module/platform/version coordination, are involved.
- A bug is severe, recurring, security/data-loss related, unclear in root cause, or needs durable remediation evidence.
- Verification cannot finish in the current task or another agent needs preserved plans, risks, decisions, or unfinished state.

Low-risk Work keeps the common metadata, routing, state machine, and gates while allowing one short paragraph or explicit N/A per body section. High-risk, cross-boundary, and long-lived changes require full detail. Never combine unrelated work into a catch-all package.

## 3. Minimal Work structure

```text
changes/<WORK-ID>/
  change.yaml
  change.md
```

`change.yaml` contains routing and state metadata. `change.md` contains the problem, behavior increment, non-goals, impact, constraints, tasks, acceptance, evidence, security/data lifecycle, and compatibility. Add `design.md` or an ADR only for complex security, schema, IPC/API, migration, or architecture work.

## 4. Context routing and controlled search

- `requirements`, `adrs`, `context_refs`, and `related_changes` form the declared initial reading set.
- `context_refs` uses repository-root-relative paths and may append `#<heading or stable ID>` to constrain reading. It must not point to another Work.
- `related_changes` is the only default route to another Work body; `supersedes` also permits reading the superseded Work.
- Known Work ID: read its YAML/body first, then expand one routing layer. Unknown Work ID: read the spec index and search only Work filenames and YAML metadata first.
- A search hit, shared ID, module, or path does not create a Work dependency. Before reading undeclared Work, state the concrete unanswered question; if confirmed, add its ID to `related_changes`.
- New and materially updated Work must include both routing lists, even when empty.
- Security/trust, credentials, persistence/schema, authorization, irreversible migration, platform boundaries, or discovered conflicts require expansion to every directly relevant primary spec and ADR.
- Work status is owned only by `change.yaml`. Indexes, Traceability, and Releases must not duplicate an active-Work status table.

## 5. State machine

- `Draft`: scope discovery and reversible investigation only.
- `Accepted`: behavior, non-goals, risks, and acceptance are approved; implementation may be prepared.
- `Implementing`: accepted behavior is merged into primary specs and Traceability; product code is changing.
- `Verified`: implementation and applicable automated/platform acceptance have recorded evidence and await sealing.
- `Released`: compatible historical completed state only; new releases do not rewrite Work to this state.
- `Rejected`: not implemented, with the reason preserved.

Normal path: `Draft -> Accepted -> Implementing -> Verified -> sealed`. Rejection ends at `Rejected -> sealed`. A merge is not verification. Actual delivery belongs to a Release record and Git tag.

## 6. Completed Work sealing

- `Verified`, `Rejected`, and historical `Released` are completed. Run `pnpm work:archive -- <WORK-ID>` in the same task; completion is not final until the manifest is written.
- The manifest records Work ID, status, timestamp, complete directory file set, and per-file SHA-256. Work stays in place so references remain valid.
- After sealing, never modify, delete, or add files under that Work directory and never delete, replace, or recalculate its existing manifest entry. Corrections and supplemental evidence require a new Work.
- `changes/archive.json` is the single machine-readable archive owner. File permissions or a second archive directory are not integrity controls.
- Local `pnpm docs:check` compares against `HEAD`. CI sets `AIVO_ARCHIVE_BASE_REF` to a trusted base commit so the same commit cannot rewrite a sealed body and its digest.

## 7. Implementation flow

1. Read Work metadata and body, then the routed Requirement, spec, ADR, and Test context.
2. Freeze scope, behavior, non-goals, risks, compatibility, and acceptance before `Accepted`.
3. Merge accepted behavior into the owning primary specs and Traceability, then move to `Implementing`.
4. Implement a vertical slice, including failures, cancellation, repeated execution, timeouts, lifecycle cleanup, migration/rollback, and security where applicable.
5. Record focused and full command evidence, CI/commit/build references, platform acceptance, and screenshots in the Work.
6. Mark `Verified` only after all applicable evidence passes, finish the body, and immediately seal the Work as its last mutation.
7. Release by referencing sealed Work in a Release record and creating the same-name Git tag; do not modify Work.

Bug Work must first reference an existing Requirement and capture minimum reproduction, expected/actual behavior, affected versions/surfaces, root cause, why tests missed it, a pre-fix failing/post-fix passing regression test, verification platforms, and fix version. If no Requirement exists, determine whether this is a specification omission, a behavior change requiring CHG, or by-design rejection.

## 8. IDs, ADRs, and history

- Work, Requirement, Test, ADR, and OPEN IDs are never reused or redefined.
- Superseded or removed Requirements remain with `Superseded by <ID>` or `Removed in vX.Y.Z`.
- Completed and rejected Work remains for evidence and becomes permanently read-only when sealed.
- Governance changes use `type: governance` and increment the specification revision.
- Create or revise an ADR for persistence ownership/schema migration strategy, Electron privilege boundaries, public API/RPC/IPC contracts, provider credential ownership, plugin/MCP trust, sandbox/command authorization, platform scope, or irreversible migration.
- Git tags freeze full historical snapshots; do not maintain duplicate versioned spec trees.

## 9. Release minimum

A Release records version, date, Git tag, Electron/desktop/core/schema/contract builds, sealed delivered Work, compatibility/migration, test and platform evidence, known issues, rollback limits, and compensation. It never redefines a Requirement or changes sealed Work.

## 10. Completion gates

- Work YAML parses and uses only template fields; routes, selectors, ADRs, and related Work resolve.
- Primary specs own final behavior and Traceability maps every Requirement and Test ID.
- Bug evidence includes the reproduction and regression result.
- Failure, cancellation, repetition, timeout, teardown, migration, rollback, security, and platform effects are verified or explicit N/A.
- All completed Work is sealed; archived file sets, digests, and previous manifest entries remain unchanged.
- Documented `pnpm` commands exist.
- Release records reference only sealed Work and pair with a Git tag.
