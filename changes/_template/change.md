# Change title

> Documentation proportionality: detail follows the decisions, risks, and evidence that must be preserved—not UI/Bug/refactor labels or diff size. A low-risk Work may use one short paragraph or explicit N/A per section.

## Problem or goal

Describe the trigger, context, and user outcome. For a Bug include minimum reproduction, environment, expected behavior, and actual behavior.

## Expected behavior

List observable behavior and Requirement IDs. A Bug must restore an existing Requirement rather than changing its meaning.

## Non-goals

Name adjacent behavior that this Work will not implement.

## Impact

Cover renderer, Electron main/preload, Go domain/app/persistence/transport, public API/RPC/IPC, schema/data, providers, skills/plugins/MCP, LSP, terminals/processes, worktrees, security, performance, dependencies, and release gates. State “none” for applicable high-risk areas with no impact.

## Implementation constraints

Record the owner, dependency direction, failure, cancellation, repetition, timeout, teardown, compatibility, and recovery requirements. Use `design.md` or an ADR for complex decisions; do not duplicate primary specs.

List direct primary-spec paths and optional `#<heading or stable ID>` selectors in `context_refs`. Other Work belongs only in `related_changes`.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `<TASK-ID>` | `<REQ/NFR-ID or N/A>` | `<output>` | `<AT/CT-ID>` | Pending |

## Acceptance and evidence

- Cover happy path, boundaries, refusal, failure, cancellation, repetition, timeout, cleanup, compatibility, migration/rollback, and regression as applicable.
- List applicable operating systems, architectures, package/install cases, and wide/narrow UI states.
- Record command output, CI, commit/build, screenshots, or platform evidence after implementation.
- After final evidence and `Verified` or `Rejected`, run `pnpm work:archive -- <WORK-ID>`; this directory then becomes permanently read-only.

## Security and data lifecycle

Describe secret/private-data ownership, process and DTO flow, persistence, cleanup, and log/diagnostic/clipboard/crash/backup effects.

## Compatibility and migration

Describe schema/data, API/RPC/IPC, settings, upgrade/downgrade, rollback, and irreversible compensation. Write “none” when unaffected.

## Bug root cause (type=bug only)

Record root cause, affected versions, why existing tests missed it, the regression test that failed before and passed after, and the fix version.
