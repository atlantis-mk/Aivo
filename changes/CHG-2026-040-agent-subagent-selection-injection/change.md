# Configure subagent injection from Agent creation

## Problem or goal

Subagent associations are currently configured with a large checkbox list after an Agent exists, while the model-facing association explanation is exposed as the managed `protocol.subagents` prompt. Users need to choose callable subagents while creating or editing a primary-capable Agent, and the association explanation should be generated from that validated configuration rather than maintained as a standalone protocol document.

## Expected behavior

`REQ-AGENT-001` presents eligible subagent-capable modes in a keyboard-operable multi-select dropdown for primary-capable Agent creation and editing. `CreateAgentPrompt` persists the selected role and bounded association IDs with the new Agent. `REQ-PROMPT-001` no longer lists or accepts `protocol.subagents` as a managed prompt; Core deterministically generates the association explanation from the effective validated IDs and injects it only when the list is non-empty.

The existing ADR-0015 allowlist, Provider-schema narrowing, execution-time revalidation, concurrency, cancellation, permission, and child-session behavior remain unchanged.

## Non-goals

This Work does not automatically run all selected subagents, change the maximum of 16 associations, add nested delegation, broaden subagent permissions, remove `agent_delegate_task`, change project configuration, or delete a user's dormant former override file.

## Impact

The renderer replaces the association checkbox list with a dropdown and adds role/association fields to Agent creation. The typed `CreateAgentPrompt` input, Core creation service, generated system prompt, embedded prompt catalog, prompt-registry compatibility handling, and focused tests change. Persistence schema, Electron privileges, provider transports, tools, extensions, MCP, LSP, terminals, worktrees, credentials, and dependencies are unchanged.

## Implementation constraints

Core remains authoritative for role, target eligibility, association bounds, referential validation, delegation-tool exposure, and execution. The renderer submits IDs but cannot grant delegation authority. Generated association text names only the validated effective list and is not editable prompt content. A dormant `protocol.subagents` override is ignored and omitted from the catalog without destructive file deletion. Existing Agent definitions and project `subagents` arrays remain compatible.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `AGENT-SUBSELECT-DOC-001` | `REQ-AGENT-001`, `REQ-PROMPT-001` | Primary specs and Traceability own generated association injection and dropdown creation behavior | `AT-AGENT-001`, `AT-PROMPT-001` | Completed |
| `AGENT-SUBSELECT-CORE-001` | `REQ-AGENT-001`, `REQ-PROMPT-001` | Typed creation input persists role/associations and Core injects deterministic generated context without a managed protocol prompt | `AT-AGENT-001`, `AT-PROMPT-001` | Completed |
| `AGENT-SUBSELECT-UI-001` | `REQ-AGENT-001`, `NFR-UI-001` | Agent creation/editing uses an accessible multi-select dropdown with empty, disabled, long-list, and narrow states | `AT-AGENT-001`, `AT-UI-001` | Completed |
| `AGENT-SUBSELECT-QA-001` | all | Focused tests, repository gates, and wide/narrow UI evidence | all | Completed |

## Acceptance and evidence

- A newly created primary/all Agent persists selected valid subagents; a subagent-only Agent cannot own associations.
- The editor dropdown shows only visible subagent-capable non-self candidates, supports multiple selections, and clearly reports the selected count.
- `protocol.subagents` is absent from the prompt catalog and cannot become active from an old override.
- Non-empty associations produce deterministic model context naming only selected IDs; empty associations omit both context and delegation authority.
- Forged/stale target refusal, concurrency, cancellation, repetition, timeout, teardown, permissions, and result ordering retain existing coverage.
- `pnpm docs:check`, focused tests, `pnpm test:core`, `pnpm scripts:test`, `pnpm lint`, `pnpm build`, `git diff --check`, and wide/narrow screenshots provide final evidence.

Verification evidence on 2026-08-23:

- Focused Core tests passed for managed-protocol retirement, dormant override preservation/refusal, deterministic generated association context, delegation schema narrowing, and Agent creation role/association persistence with failure rollback.
- `pnpm test:core` passed every Go package; `pnpm scripts:test` passed governance, desktop model, and extension tests; `pnpm docs:check` and `git diff --check` passed.
- `pnpm lint` passed with only the repository's existing shared-UI Fast Refresh warnings. `pnpm build` passed TypeScript compilation and the Vite production build.
- Browser QA selected `Research` and `Review` without closing the dropdown, confirmed checked accessibility states and the `已选择 2 个子 Agent` summary, and repeated the selection using Tab, Enter, ArrowDown, and Space. No console errors remained after the QA favicon was made explicit.
- Wide evidence at a 1440×900 page viewport: `evidence/agent-subagent-select-wide.png`. Narrow evidence at a 500×760 page viewport: `evidence/agent-subagent-select-narrow.png`; the dialog remained within the viewport and `document.body.scrollWidth` equaled `clientWidth`.

## Security and data lifecycle

Associations contain only normalized Agent IDs. Core validates them before persistence and execution; generated text cannot authorize an unlisted target. No prompt body, child result, credential, private file content, raw tool payload, or new diagnostic data crosses this flow. Former override files remain local and dormant rather than being destructively removed.

## Compatibility and migration

No schema transition is required. Existing global and project association arrays retain their meaning. The local `CreateAgentPrompt` RPC adds optional `mode` and `subagents` fields; omitted fields preserve the existing safe `all` role with an empty list. `protocol.subagents` is retired from the active catalog, while a pre-existing override file remains recoverable but ignored. Downgrade restores the previous catalog behavior without data conversion.

## Bug root cause (type=bug only)

N/A.
