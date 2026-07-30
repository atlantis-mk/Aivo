# Aivo data model and persistence

## Ownership

Domain meaning belongs in `core/domain`; persistence mappings and database access belong in `core/infra/persistence`. HTTP DTOs, renderer types, and provider payloads are adapters and must not silently redefine persisted or domain semantics.

## Current entity groups

- Projects and project runtime configuration
- Conversations, turns, messages, and retained output
- Sessions, agent execution state, tool calls, and permission/question records
- Provider accounts, model preferences, usage, and non-secret configuration
- Skills, plugins, and MCP configuration/state
- Worktrees and process/terminal associations

The code schema remains the exhaustive owner. This document owns persistence lifecycle and compatibility rules, not a duplicated field list.

## Schema versions

- The archived v1 baseline reports schema version 1 and uses GORM migration behavior.
- Every v2 schema transition must increment an explicit numeric version.
- A transition must define forward steps, transaction boundaries, indexes, compatibility behavior, removal conditions, and rollback or compensation.
- Schema or serialized-format changes require a `migration` Work and, when ownership or irreversibility changes, an ADR.

## Migration safety

Before a migration mutates user data, Aivo must create or verify a recoverable backup. Tests use sanitized previous-version fixtures and cover success, repeated startup/idempotence, malformed input, partial failure, rollback, and application-version downgrade behavior.

Multi-step consistency uses explicit transactions with checked rollback. Query paths used by project lists, session timelines, tool activity, and worktree state require pagination/index review and must avoid N+1 behavior.

## Sensitive data

Provider secrets and OAuth tokens use approved secure storage and are represented in general DTOs only by safe status or reference data. Raw credentials, prompts, tool payloads, private file contents, and provider responses are not copied into analytics, crash reports, logs, migration fixtures, or diagnostics exports.
