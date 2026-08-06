# Aivo data model and persistence

## Ownership

Domain meaning belongs in `core/domain`; persistence mappings and database access belong in `core/infra/persistence`. HTTP DTOs, renderer types, and provider payloads are adapters and must not silently redefine persisted or domain semantics.

## Current entity groups

- Projects and project runtime configuration
- Global application configuration, including the initial workspace for unscoped conversations
- Conversations, turns, messages, and retained output
- Sessions, agent execution state, tool calls, and permission/question records
- Provider accounts, model preferences, usage, and non-secret configuration
- Skills, Manifest v2 extensions, and MCP configuration/state
- Managed local extension installations, confirmed integrity, desired enabled state, and safe restoration status
- Worktrees and process/terminal associations

The code schema remains the exhaustive owner. This document owns persistence lifecycle and compatibility rules, not a duplicated field list.

## Schema versions

- The archived v1 baseline reports schema version 1 and uses GORM migration behavior.
- Schema version 2 adds `app_config.initial_workspace_path` as the global owner of the user-confirmed directory used by unscoped coding contexts. The derived initialization default is returned at runtime and is not persisted until confirmation.
- Schema version 3 adds `extension_installs` as the durable owner of a user-confirmed Manifest v2 installation, validated manifest snapshot, integrity, desired enabled state, and safe status. It stores no credentials or runtime message data.
- Schema version 4 adds `extension_installs.install_mode`. Historical rows begin as `linked`; successful new and promoted installations use `managed` and point only to a Host-owned integrity-addressed package generation.
- Legacy `plugin_installs` rows may remain after upgrade for downgrade preservation, but current application code never queries, mutates, restores, or executes them. Removing the table requires a later explicit migration.
- Every later schema transition must increment an explicit numeric version.
- A transition must define forward steps, transaction boundaries, indexes, compatibility behavior, removal conditions, and rollback or compensation.
- Schema or serialized-format changes require a `migration` Work and, when ownership or irreversibility changes, an ADR.

## Migration safety

Before a migration mutates user data, Aivo must create or verify a recoverable backup. Tests use sanitized previous-version fixtures and cover success, repeated startup/idempotence, malformed input, partial failure, rollback, and application-version downgrade behavior.

The v1-to-v2 transition creates or verifies a consistent SQLite backup beside the database before adding the application-config column. The v2-to-v3 transition likewise creates or verifies the v2 backup before adding `extension_installs` transactionally. The v3-to-v4 transition creates or verifies the v3 backup before adding installation mode. A failed or invalid backup blocks schema mutation. Fresh databases start directly at v4. Legacy managed-workspace files are not moved or deleted; an existing unscoped coding-context record may be repointed to the configured initial workspace when that conversation next submits work.

An extension installation points to a private package generation below the platform application-data `Aivo/Default/Extensions` root; explicit isolated database stores may use a database-sibling root. Confirmation copies from the transient user-selected source into staging, validates the copied Manifest/integrity, and atomically publishes before persistence or runtime eligibility. Startup reads only the managed generation and requires an exact persisted integrity. Updates create a new confirmed generation; prior generations may remain until no runtime owns them and a safe restart or uninstall collects them. Uninstall deletes only the validated managed extension directory and row, never the original source. Historical schema v3 linked rows promote only when the old source still matches the confirmed integrity; exact packages in the former default database-sibling root similarly copy and verify before their persisted path changes and their old owned directory is removed.

Multi-step consistency uses explicit transactions with checked rollback. Query paths used by project lists, session timelines, tool activity, and worktree state require pagination/index review and must avoid N+1 behavior.

Agent project queries use bounded keyset pagination over the existing project update-time index. A conversation's first project association conditionally updates an empty `sessions.project_id` and its `coding_contexts` row in one transaction; the project ID is immutable afterward, concurrent attempts have one winner, and no schema transition is required.

Conversational MCP registration proposals are bounded ephemeral Core state owned by one session and turn; they are not persisted. After exact Host confirmation, MCP configuration and discovered capabilities reuse the existing MCP tables. A new source is saved disabled and becomes enabled/ready only after successful bounded discovery; failure must leave no eligible partial capability set. Durable proposal or authorization auditing requires a later explicit schema transition.

## Sensitive data

Provider secrets and OAuth tokens use approved secure storage and are represented in general DTOs only by safe status or reference data. Raw credentials, prompts, tool payloads, private file contents, and provider responses are not copied into analytics, crash reports, logs, migration fixtures, or diagnostics exports.
