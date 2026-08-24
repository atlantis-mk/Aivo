# Associate primary Agent modes with callable subagents

## Problem or goal

Agent modes can already be primary-only, subagent-only, or usable in both roles, and Core already owns bounded child-session delegation. However, a primary mode cannot declare which subagents it is allowed and expected to use, so the model receives no per-mode delegation catalog and the management UI cannot configure that relationship.

## Expected behavior

`REQ-AGENT-001` adds a bounded `subagents` association list to managed and project Agent definitions. A primary-capable mode may select visible modes that are subagent-capable; Core rejects self-links, duplicates, hidden/protected targets, missing targets, primary-only targets, and associations on subagent-only owners. The “Extensions and MCP” editor presents the eligible catalog as a scrollable checkbox list.

`REQ-SESSION-001` exposes `agent_delegate_task` to a model only when the effective current mode has valid associations. Its schema and system context name only those modes, and execution revalidates the association before forking. The model decides when delegation helps; configuration does not unconditionally fan out every prompt. Existing depth, concurrency, cancellation, permission, Tool Snapshot, child-session, and result-order behavior remains authoritative.

## Non-goals

This Work does not create a new Agent executor, grant custom modes broader toolsets, auto-run every associated mode, manage hidden workers, persist child prompts/results in the relationship, add graph visualization, or let the renderer authorize delegation.

## Impact

Core domain/catalog/management and project runtime definitions gain association IDs; schema v8 records the compatible payload contract after a verified v7 backup. Runtime Provider requests gain a narrowed delegation schema and prompt context only when associations exist. The renderer gains a responsive scrollable association chooser. Electron privileges, credentials, providers, extensions, MCP, LSP, terminals, worktrees, and release packaging are otherwise unchanged.

## Implementation constraints

Core owns normalization and referential validation. Associations are an allowlist, not merely presentation metadata. The maximum is 16 normalized unique IDs. Only modes with role `primary` or `all` may own associations; targets must be visible, manageable, not self, and role `subagent` or `all`. Saving a role change and deleting a custom target must refuse dangling references. Project overlays may replace the list explicitly and are validated in the effective catalog before execution. An empty or absent list omits the delegation tool and does not authorize legacy unrestricted delegation.

Schema v8 must create or verify a v7 backup before recording the new version. Existing rows remain byte-compatible and begin with no associations; migration is transactional and downgrade recovery restores the v7 backup. No prompt, tool payload/result, child output, credential, or private file content enters association data or logs.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `AGENT-LINK-DOC-001` | `REQ-AGENT-001`, `REQ-SESSION-001` | Requirement, ADR, architecture, data model, runtime configuration, tests, and traceability own the association contract | `AT-AGENT-001`, `AT-SESSION-001` | Completed |
| `AGENT-LINK-MIGRATION-001` | `REQ-AGENT-001`, `NFR-RELIABILITY-001` | Schema-v8 version transition with verified v7 backup and rollback evidence | `AT-AGENT-001`, `CT-RELIABILITY-001` | Completed |
| `AGENT-LINK-CORE-001` | `REQ-AGENT-001` | Bounded CRUD/catalog validation and reference-safe save/delete | `AT-AGENT-001` | Completed |
| `AGENT-LINK-RUNTIME-001` | `REQ-SESSION-001` | Narrowed prompt/schema/snapshot plus execution-time allowlist enforcement | `AT-SESSION-001`, `CT-RELIABILITY-001` | Completed |
| `AGENT-LINK-UI-001` | `REQ-AGENT-001`, `NFR-UI-001` | Scrollable checkbox association editor with empty/error/narrow states | `AT-AGENT-001`, `AT-UI-001` | Completed |
| `AGENT-LINK-QA-001` | all | Focused and repository gates plus wide/narrow screenshots | all | Completed |

## Acceptance and evidence

- Managed CRUD and project overlays round-trip bounded association IDs while invalid graphs fail with actionable errors.
- A referenced custom target cannot be deleted or changed to primary-only until references are removed; built-in reset restores its current code-owned association defaults.
- Provider requests omit `agent_delegate_task` for no associations and otherwise expose only the configured target IDs; forged or stale calls fail before creating a child session.
- Associated delegation retains depth, bounded parallelism, cancellation, result recording, and permission behavior.
- Schema v7 receives a verified backup before v8; repeated startup is deterministic, an invalid backup blocks version mutation, and v7 data remains recoverable.
- The editor is keyboard-operable, uses the shared checkbox and scroll-area primitives, and remains usable at wide and narrow sizes with long lists.
- `pnpm docs:check`, focused tests, `pnpm test:core`, `pnpm scripts:test`, `pnpm lint`, `pnpm build`, and `git diff --check` provide final evidence.

Verification evidence on 2026-08-07:

- Focused Core tests passed for association persistence/restart, self/duplicate/missing/hidden/role refusal, referenced delete and role-change refusal, project JSON/Markdown overlay, prompt/schema narrowing, mode-default Tool Snapshot activation, and forged-target refusal before child creation.
- Schema-v7 fixtures received a verified v7 backup before v8; payload preservation, repeated startup, invalid-backup refusal, and existing v6-to-v7 cleanup regression passed.
- `pnpm test:core` passed every Go package. `pnpm scripts:test` passed governance, desktop model, and extension tests. `pnpm docs:check` and `git diff --check` passed.
- `pnpm lint` passed with only the repository's existing shared-UI Fast Refresh warnings. `pnpm build` passed TypeScript compilation and the Vite production build.
- Local browser QA at 1440×900 and 420×760 confirmed independent outer-form and inner-association scrolling, a 12-item scroll to the final option, no horizontal overflow, visible footer actions, and no console errors.
- Wide screenshot: `artifacts/design-qa/agent-subagent-associations-wide-2026-08-07.png`. Narrow screenshot: `artifacts/design-qa/agent-subagent-associations-narrow-2026-08-07.png`.

## Security and data lifecycle

Associations contain only normalized non-secret mode IDs. Core revalidates both configuration and each call; renderer state and model arguments cannot expand the allowlist. Child prompts/results retain their existing bounded session/run storage and are not copied into the association row. Logs and diagnostics do not add prompt, result, credential, or private-content data.

## Compatibility and migration

Schema v8 adds no SQL column and preserves existing definition JSON bytes, but establishes `subagents` as a supported stored and RPC member. Existing rows and project files without the member behave as an empty allowlist. A v7 binary must use the verified v7 backup for downgrade; its newer-schema guard refuses the v8 database. Project files may add `subagents` without global CRUD rewriting them.

## Bug root cause (type=bug only)

N/A.
