# Configure one initial workspace for unscoped conversations

## Problem or goal

When a coding conversation has no selected project, Aivo currently creates a new date/session directory under `Documents/Aivo Workspaces` and may recreate that directory after deletion. Initialization does not establish where unscoped work runs. Replace that implicit lifecycle with one explicit, persisted initial workspace confirmed during setup, with the existing managed root offered as the default.

## Expected behavior

`REQ-PROJECT-002` requires initialization to offer `~/Documents/Aivo Workspaces` as a default, let the user accept it or select another local directory, and finish only after confirming that path. Every temporary or otherwise projectless coding conversation uses that same directory as its tool/runtime workspace while remaining unscoped in project navigation. Aivo must not create a per-conversation directory. If the configured root is missing, Aivo recreates that root at the same path; a non-directory or unconfigured path produces an actionable error.

## Non-goals

No per-conversation subdirectory, project import, project association, worktree policy change, cloud sync, account behavior, or migration of files from legacy managed workspaces. Editing the initial workspace after setup is not added in this slice.

## Impact

The renderer gains a final initialization step, displays a backend-provided default, and reuses the existing privileged directory picker for replacement. Go domain/application code derives the default and validates and persists the confirmed path, the local RPC accepts a typed initialization input, and unscoped session creation resolves the configured path instead of allocating a directory. SQLite `app_config` advances from schema v1 to v2 with a pre-migration backup. Provider credentials, model payloads, plugins/MCP, LSP ownership, worktrees, dependencies, and platform scope are unchanged.

## Implementation constraints

Go application services own the default-path derivation, path validation, and directory creation; the renderer only displays the suggestion and invokes the privileged picker when the user wants another path. The confirmed path must resolve to an absolute directory path. The service does not persist or create the default before confirmation, then creates the configured root with private permissions when it is missing, but never allocates a session child directory. Projectless sessions must not be persisted as project-owned merely because their coding context uses the initial workspace. Migration follows `ADR-0001`, creates or verifies a recoverable v1 backup before mutation, runs transactionally where SQLite permits, is idempotent, and leaves the additional column compatible with the previous application version.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `INITIAL-WORKSPACE-DOC-001` | `REQ-PROJECT-002` | Requirement, ADR, schema lifecycle, and traceability are current | `AT-PROJECT-002` | Completed |
| `INITIAL-WORKSPACE-MIGRATION-001` | `NFR-RELIABILITY-001` | Schema v2 path field with verified pre-migration backup and retry safety | `CT-RELIABILITY-001` | Completed |
| `INITIAL-WORKSPACE-CORE-001` | `REQ-PROJECT-002` | Initialization validation and shared unscoped coding context | `AT-PROJECT-002` | Completed |
| `INITIAL-WORKSPACE-UI-001` | `REQ-PROJECT-002` | Final responsive setup step with directory selection and errors | `AT-UI-001` | Completed |
| `INITIAL-WORKSPACE-QA-001` | `REQ-PROJECT-002` | Automated gates plus wide/narrow setup acceptance | `AT-PROJECT-002` | Pending |

## Acceptance and evidence

- Initialization displays the managed-root default without creating it, can finish with that default or a user-selected directory, and reports selection, creation, or save failures without losing the current step.
- Two unscoped coding conversations receive the same configured coding-context path and no child directory is created.
- Explicitly project-scoped conversations continue to use their selected project and remain isolated.
- Deleting the configured directory causes Aivo to recreate only that configured root. Replacing it with a non-directory causes an actionable failure.
- A schema v1 fixture is backed up before the new column is added, migrates to version 2, preserves configuration, and reopens idempotently. An invalid existing backup blocks migration before schema mutation.
- Upgrade keeps existing initialized state but requires initial-workspace configuration before a new unscoped coding conversation can be created. Downgrade tolerates the additive column but does not understand or enforce the new behavior.
- Cancellation is the directory-picker cancel path. Repetition is repeated completion with the same valid path. Runtime cancellation, timeout, teardown, providers, secrets, and worktrees are otherwise unchanged.
- `pnpm docs:check`, `pnpm test:core`, `pnpm lint`, and `pnpm build` pass. Wide and narrow setup states require screenshot acceptance before verification.

Automated evidence recorded on 2026-08-03:

- `pnpm docs:check` passed.
- `pnpm test:core` passed, including default-path suggestion/confirmation, initial-workspace creation/recreation, unscoped/project separation, explicit-project precedence, and schema-v1 backup/migration cases.
- `pnpm lint` passed with non-blocking Fast Refresh warnings in shared UI/route files.
- `pnpm build` passed with non-blocking large-barrel and chunk-size advisory warnings.

Per the user's verification preference, wide/narrow setup acceptance remains pending. This Work therefore remains `Implementing`; it may move to `Verified` and be sealed only after that acceptance is recorded.

## Security and data lifecycle

The confirmed absolute path is private local metadata stored in SQLite and returned in the local `AppConfig` DTO. The unconfirmed default is derived locally and returned only as a suggestion. Neither is a secret, but paths must not be added to logs, diagnostics, analytics, crash output, clipboard, or committed fixtures. The confirmed value becomes the filesystem/tool authorization root for unscoped conversations, so automatic creation is limited to that exact root. Migration backups contain the private local database and remain beside it with restrictive permissions; they are never exported automatically.

## Compatibility and migration

Schema v2 adds nullable/text `app_config.initial_workspace_path`. Before changing an existing schema v1 database, Aivo creates or verifies a consistent `v1` SQLite backup beside the database. Fresh databases are created directly at v2. The transition is idempotent; failed backup validation or migration leaves the v1 database usable. The additive column is tolerated by the previous binary, while rollback requires restoring the v1 backup to remove v2-only configuration semantics. Legacy per-session workspace files are never moved or deleted automatically; an existing unscoped coding-context record is repointed to the configured initial workspace only when that conversation next submits work.

## Bug root cause (type=bug only)

N/A.
