# ADR-0001: Persist the initial workspace in application configuration

- Status: Accepted
- Date: 2026-08-03
- Related Work: `CHG-2026-006-initial-workspace-directory`
- Closes OPEN: none

## Context

Unscoped coding conversations need a privileged, durable filesystem root even though they are not associated with a project. The current implementation derives a managed root from the environment or home directory and allocates one directory per session. The derived root remains a useful initialization default, but it must not be treated as confirmed or created before setup completes. Renderer-local state would not protect other clients or runtime entry points, and project runtime configuration would incorrectly allow project layers to redefine a global initialization choice.

## Decision

- The Go application and persistence layers MUST own one global `initialWorkspacePath` in `AppConfig`.
- The Go application MUST expose `~/Documents/Aivo-Workspaces` (or the existing environment override) as the initialization default without persisting or creating it before confirmation. The default directory name uses a hyphen rather than whitespace so terminal entry and path search do not require space escaping.
- Initialization MUST let the user accept that default or choose another directory, resolve the confirmed path to an absolute path, ensure the configured root exists as a directory, persist it, and only then mark initialization complete.
- An unscoped coding session MUST use the configured directory in its coding context but MUST NOT be persisted as belonging to a project.
- Aivo MUST create the configured root with private permissions when it is missing, but MUST NOT create per-session child directories, delete the root, or migrate content into it.
- SQLite schema v1 MUST be backed up and advanced explicitly to v2 before adding the configuration column. Migration or backup failure MUST stop startup without mutating user data further.
- The renderer MUST obtain the default through `AppConfig`, use the existing privileged directory-picker IPC only when the user chooses another path, and MUST NOT perform filesystem validation or mutation itself.

## Rationale

- Application configuration is the existing owner of initialization state and other global runtime defaults.
- Keeping project association empty preserves temporary-conversation navigation semantics while a coding context supplies the tool boundary.
- A single shared user-confirmed directory matches the requested lifecycle and removes hidden per-session filesystem allocation while retaining a convenient default.
- An additive column gives the path an explicit, queryable owner; hiding it inside unrelated serialized provider/tool configuration would create an unsafe implicit contract.

## Consequences

- The local `AppConfig` and `CompleteInitialization` RPC contracts change together.
- A fresh setup can be completed with the suggested default without opening the directory picker; selecting another path replaces the suggestion before confirmation.
- Schema migration gains backup, version, retry, integrity, and downgrade verification obligations.
- Existing installations without a configured path can continue project-scoped work, but new unscoped coding conversations fail with setup guidance until configuration is completed.
- Multiple unscoped conversations share one filesystem root and therefore do not receive filesystem isolation from one another; conversation and execution records remain session-isolated.

## Rejected alternatives

- Keep allocating per-session directories: contradicts the requested shared directory and continues hidden lifecycle behavior.
- Store the path in renderer local storage: bypasses privileged service ownership and does not cover non-renderer session creation.
- Store the path in project runtime configuration: makes a global initialization choice layerable by projects and mixes application setup with agent runtime policy.
- Reuse an unrelated JSON column: avoids a migration only by obscuring ownership and compatibility.

## Verification

`AT-PROJECT-002` covers initialization validation, shared unscoped workspace resolution, exact-root recreation, explicit project precedence, and no per-session directory creation. `CT-RELIABILITY-001` covers schema v1 backup, v2 migration, idempotence, refusal on invalid backup, and recovery compatibility. `AT-UI-001` covers responsive setup states and keyboard focus.
