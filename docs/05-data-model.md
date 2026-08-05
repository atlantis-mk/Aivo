# Aivo data model and persistence

## Ownership

Domain meaning belongs in `core/domain`; persistence mappings and database access belong in `core/infra/persistence`. HTTP DTOs, renderer types, and provider payloads are adapters and must not silently redefine persisted or domain semantics.

## Current entity groups

- Projects and project runtime configuration
- Global application configuration, including the initial workspace for unscoped conversations
- Conversations, turns, messages, and retained output
- Sessions, agent execution state, tool calls, and permission/question records
- Provider accounts, model preferences, usage, and non-secret configuration
- Skills, plugins, and MCP configuration/state
- Worktrees and process/terminal associations

The code schema remains the exhaustive owner. This document owns persistence lifecycle and compatibility rules, not a duplicated field list.

## Schema versions

- The archived v1 baseline reports schema version 1 and uses GORM migration behavior.
- Schema version 2 adds `app_config.initial_workspace_path` as the global owner of the user-confirmed directory used by unscoped coding contexts. The derived initialization default is returned at runtime and is not persisted until confirmation.
- Every later schema transition must increment an explicit numeric version.
- A transition must define forward steps, transaction boundaries, indexes, compatibility behavior, removal conditions, and rollback or compensation.
- Schema or serialized-format changes require a `migration` Work and, when ownership or irreversibility changes, an ADR.

## Migration safety

Before a migration mutates user data, Aivo must create or verify a recoverable backup. Tests use sanitized previous-version fixtures and cover success, repeated startup/idempotence, malformed input, partial failure, rollback, and application-version downgrade behavior.

The v1-to-v2 transition creates or verifies a consistent SQLite backup beside the database before adding the application-config column. A failed or invalid backup blocks schema mutation. Fresh databases start directly at v2. Legacy managed-workspace files are not moved or deleted; an existing unscoped coding-context record may be repointed to the configured initial workspace when that conversation next submits work.

Multi-step consistency uses explicit transactions with checked rollback. Query paths used by project lists, session timelines, tool activity, and worktree state require pagination/index review and must avoid N+1 behavior.

Agent project queries use bounded keyset pagination over the existing project update-time index. A conversation's first project association conditionally updates an empty `sessions.project_id` and its `coding_contexts` row in one transaction; the project ID is immutable afterward, concurrent attempts have one winner, and no schema transition is required.

## Sensitive data

Provider secrets and OAuth tokens use approved secure storage and are represented in general DTOs only by safe status or reference data. Raw credentials, prompts, tool payloads, private file contents, and provider responses are not copied into analytics, crash reports, logs, migration fixtures, or diagnostics exports.
