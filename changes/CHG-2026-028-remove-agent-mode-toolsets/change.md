# Remove toolsets from managed Agent-mode data

## Problem or goal

The Agent-mode editor exposed a comma-separated toolset field and the same value was accepted and persisted by Core. The user requested removal from both the UI and backend data. Tool capability selection is policy, not general mode metadata, and should not be mutable through global Agent-mode CRUD.

## Expected behavior

`REQ-AGENT-001` continues to provide dynamic mode CRUD, but managed definitions and RPC payloads no longer expose, accept, search, render, or persist `toolsets`. Visible built-ins always retain their code-owned toolsets when overridden. New and existing custom global modes receive Core's safe default toolset at runtime. Existing project runtime files remain a compatibility overlay and may still narrow or select project-specific toolsets.

## Non-goals

This Work does not remove internal runtime toolset enforcement, code-owned built-in capability sets, project runtime-file compatibility, per-conversation tool controls, tool registration, permissions, extensions, Skills, or MCP.

## Impact

Core separates managed serialized data from runtime capability state, schema v7 sanitizes existing JSON payloads, and management RPC output omits toolsets. The desktop removes toolset fields, badges, and search matching. No Electron privilege, provider, secret, process, extension, MCP, or release-packaging boundary changes.

## Implementation constraints

Core remains the authority. Schema v7 must create or verify a v6 backup before rewriting rows transactionally. Built-in overrides reattach the exact current code default toolsets; custom definitions receive only `safe` unless a project runtime overlay supplies an accepted project-specific value. Unknown client `toolsets` input must be ignored and never persisted. Downgrade restores the v6 backup if the removed payload data is needed; v7 does not attempt to reconstruct historical custom values.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `AGENT-TOOLSET-DOC-001` | `REQ-AGENT-001` | Requirement, ADR, data model, test plan, and traceability own the narrower contract | `AT-AGENT-001` | Completed |
| `AGENT-TOOLSET-MIGRATION-001` | `REQ-AGENT-001`, `NFR-RELIABILITY-001` | Schema-v7 backup and transactional payload cleanup | `AT-AGENT-001`, `CT-RELIABILITY-001` | Completed |
| `AGENT-TOOLSET-CORE-001` | `REQ-AGENT-001` | Management DTO/storage omit toolsets while runtime restores code/safe capabilities | `AT-AGENT-001` | Completed |
| `AGENT-TOOLSET-UI-001` | `REQ-AGENT-001` | Editor, cards, and search expose no toolset data | `AT-AGENT-001`, `AT-UI-001` | Completed |
| `AGENT-TOOLSET-QA-001` | all | Focused migration/CRUD tests and repository gates | all | Completed |

## Acceptance and evidence

- Fresh and upgraded management RPC responses contain no `toolsets` property; submitted unknown `toolsets` input is not retained.
- Schema v6 receives a verified backup before v7 and every persisted definition JSON object loses its `toolsets` member transactionally.
- Built-in overrides retain exact code toolsets at runtime; custom global definitions use `safe`; project runtime overlays remain higher precedence.
- Agent-mode cards, search, and editor show no toolset data.
- Repeated migration/open is deterministic; invalid backup blocks mutation; downgrade recovery uses the v6 backup.
- `pnpm docs:check`, focused tests, `pnpm test:core`, `pnpm scripts:test`, `pnpm lint`, `pnpm build`, and `git diff --check` provide evidence.

Verification evidence on 2026-08-07:

- Focused Core tests passed for managed-mode serialization, ignored legacy input, built-in capability restoration, custom safe defaults, project overlay precedence, physical store omission, schema-v6 backup, schema-v7 cleanup, invalid-backup refusal, and malformed-row rollback.
- `pnpm test:core` passed every Go package on the final full rerun.
- `pnpm scripts:test` passed all governance, desktop, and example-extension tests; the Agent-mode search regression proves legacy toolset text is not searchable.
- `pnpm lint` passed with only the repository's existing Fast Refresh export warnings in shared UI and the root route.
- `pnpm build` passed TypeScript compilation and the Vite production build.
- `pnpm docs:check` and `git diff --check` passed. Source inspection found no toolset field, badge, draft member, service DTO member, or Agent-mode QA fixture payload in the management renderer.

## Security and data lifecycle

The migration deletes non-secret capability-policy metadata from current v7 rows after creating a verified v6 backup. No credential, prompt, conversation, tool payload/result, or private file data is added or logged.

## Compatibility and migration

Schema v7 removes `toolsets` keys from persisted Agent-mode JSON. A v6 binary should use the verified v6 backup for rollback; reading a v7 database is refused by the existing newer-schema guard. Older clients may submit the removed property, but Core ignores it. Project configuration remains compatible and separate from global managed data.

## Bug root cause (type=bug only)

N/A.
