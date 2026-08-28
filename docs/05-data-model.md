# Aivo data model and persistence

## Ownership

Domain meaning belongs in `core/domain`; persistence mappings and database access belong in `core/infra/persistence`. HTTP DTOs, renderer types, and provider payloads are adapters and must not silently redefine persisted or domain semantics.

## Current entity groups

- Projects and project runtime configuration
- Global application configuration, including the personalized assistant name, initial workspace for unscoped conversations, and the explicit default permission mode for future coding conversations
- Conversations, turns, messages, and retained output
- Sessions, agent execution state, tool calls, and permission/question records
- Provider accounts, model preferences, usage, and non-secret configuration
- Skills, Manifest v2 extensions, and MCP configuration/state
- Managed local extension installations, confirmed integrity, desired enabled state, and safe restoration status
- Global Agent-mode definitions that create user modes or override visible code-owned built-ins
- Managed global prompt working files, validated content-addressed revisions, and an active manifest
- Worktrees and process/terminal associations

The code schema remains the exhaustive owner. This document owns persistence lifecycle and compatibility rules, not a duplicated field list.

Global tool visibility is a non-secret Core-owned preference. Missing state means every registered tool is visible for future automatic and manual selection. The set is normalized and bounded. A visibility change does not rewrite conversation execution metadata or revoke an already selected tool.

Session execution metadata owns two independent bounded canonical-name sets: `rememberedDeferredTools` for manual conversation activation and `autoSelectedTools` for the current automatic selection, plus `autoSelectedToolsInitialized` to distinguish an initialized empty set from a legacy conversation. Automatic replacement writes the complete set atomically. Existing warm-selection metadata is ignored; no existing row is rewritten, and older binaries ignore the additive keys.

## Schema versions

- The archived v1 baseline reports schema version 1 and uses GORM migration behavior.
- Schema version 2 adds `app_config.initial_workspace_path` as the global owner of the user-confirmed directory used by unscoped coding contexts. The derived initialization default is returned at runtime and is not persisted until confirmation.
- Schema version 3 adds `extension_installs` as the durable owner of a user-confirmed Manifest v2 installation, validated manifest snapshot, integrity, desired enabled state, and safe status. It stores no credentials or runtime message data.
- Schema version 4 adds `extension_installs.install_mode`. Historical rows begin as `linked`; successful new and promoted installations use `managed` and point only to a Host-owned integrity-addressed package generation.
- Schema version 5 adds `global_tool_preferences`, keyed by canonical tool name. Rows represent explicit future-selection visibility overrides; missing rows mean visible. The table stores no tool payload, result, credential, or session state.
- Schema version 6 adds `agent_mode_definitions`, keyed by normalized mode ID. Rows own complete user-created definitions or overrides for visible code defaults; missing built-in rows mean the current code default. Hidden internal worker IDs are never persisted through management operations.
- Schema version 7 removes `toolsets` members from every `agent_mode_definitions.definition` JSON object. Global rows own mode metadata and behavior settings but not capability sets; built-in capabilities remain code-owned and custom global modes receive the safe runtime default.
- Schema version 8 recognizes at most 16 normalized `subagents` IDs in each Agent-mode definition payload. Existing rows remain byte-compatible and begin with an empty delegation allowlist; no SQL column or legacy row rewrite is required.
- Schema version 9 replaces persisted Agent-mode prompt bodies with stable `promptId` references after validated Markdown materialization. Built-in prompt bodies remain embedded defaults; managed files and their active manifest own user text.
- Schema version 10 adds `app_config.app_name` as the global owner of the bounded non-secret name used for the built-in Assistant and desktop home presentation. Missing and empty historical values resolve to `Aivo`; stable technical identifiers remain unchanged.
- Schema version 11 adds `app_config.default_permission_mode` as the global owner of the non-secret initial permission mode for future coding conversations. Missing, empty, and unknown historical values resolve to `request_approval`; each new coding conversation writes its own session permission rule so a later preference change cannot change existing authority.
- Legacy `plugin_installs` rows may remain after upgrade for downgrade preservation, but current application code never queries, mutates, restores, or executes them. Removing the table requires a later explicit migration.
- Every later schema transition must increment an explicit numeric version.
- A transition must define forward steps, transaction boundaries, indexes, compatibility behavior, removal conditions, and rollback or compensation.
- Schema or serialized-format changes require a `migration` Work and, when ownership or irreversibility changes, an ADR.

## Migration safety

Before a migration mutates user data, Aivo must create or verify a recoverable backup. Tests use sanitized previous-version fixtures and cover success, repeated startup/idempotence, malformed input, partial failure, rollback, and application-version downgrade behavior.

The v1-to-v2 transition creates or verifies a consistent SQLite backup beside the database before adding the application-config column. The v2-to-v3 transition likewise creates or verifies the v2 backup before adding `extension_installs` transactionally. The v3-to-v4 transition creates or verifies the v3 backup before adding installation mode. The v4-to-v5 transition creates or verifies the v4 backup before adding global tool preferences transactionally. The v5-to-v6 transition creates or verifies the v5 backup before adding Agent-mode definitions transactionally. The v6-to-v7 transition creates or verifies the v6 backup before transactionally removing stored Agent-mode toolsets. The v7-to-v8 transition creates or verifies the v7 backup before transactionally recording association support without rewriting existing rows. The v8-to-v9 transition creates or verifies the v8 backup, publishes validated prompt files idempotently, then replaces Agent prompt bodies with references transactionally. The v9-to-v10 transition creates or verifies the v9 backup before adding the application-name column transactionally; existing rows resolve an empty value to the `Aivo` compatibility default. The v10-to-v11 transition creates or verifies the v10 backup before adding the default-permission-mode column transactionally; existing rows resolve a missing or invalid value to `request_approval`. A failed or invalid backup blocks schema mutation. Fresh databases start directly at v11. Legacy managed-workspace files are not moved or deleted; an existing unscoped coding-context record may be repointed to the configured initial workspace when that conversation next submits work.

Agent-mode rows store non-secret definition data without toolsets plus timestamps. Save replaces one normalized ID atomically, ignores removed legacy toolset input, and accepts only bounded valid subagent associations. Deleting a user-created row first checks durable session and managed association references in the same transaction; a referenced ID is retained and the operation fails. Deleting a built-in row means reset and never deletes the code default. Project runtime files are external configuration inputs and are not migrated into or mutated by this table.

An extension installation points to a private package generation below the platform application-data `Aivo/Default/Extensions` root; explicit isolated database stores may use a database-sibling root. Confirmation copies from the transient user-selected source into staging, validates the copied Manifest/integrity, and atomically publishes before persistence or runtime eligibility. Startup reads only the managed generation and requires an exact persisted integrity. Updates create a new confirmed generation; prior generations may remain until no runtime owns them and a safe restart or uninstall collects them. Uninstall deletes only the validated managed extension directory and row, never the original source. Historical schema v3 linked rows promote only when the old source still matches the confirmed integrity; exact packages in the former default database-sibling root similarly copy and verify before their persisted path changes and their old owned directory is removed.

Multi-step consistency uses explicit transactions with checked rollback. Query paths used by project lists, session timelines, tool activity, and worktree state require pagination/index review and must avoid N+1 behavior.

Agent project queries use bounded keyset pagination over the existing project update-time index. A conversation's first project association conditionally updates an empty `sessions.project_id` and its `coding_contexts` row in one transaction; the project ID is immutable afterward, concurrent attempts have one winner, and no schema transition is required.

Conversational MCP registration proposals are bounded ephemeral Core state owned by one session and turn; they are not persisted. After exact Host confirmation, MCP configuration and discovered capabilities reuse the existing MCP tables. A new source is saved disabled and becomes enabled/ready only after successful bounded discovery; failure must leave no eligible partial capability set. Durable proposal or authorization auditing requires a later explicit schema transition.

Desktop-entered MCP Bearer tokens use the existing Host secret store and existing persisted MCP access-token-reference field; SQLite stores only the reference. Environment-variable Bearer rows remain compatible. A save without a replacement retains the current reference, while replacement or authentication-mode changes compensate secret state if configuration persistence fails. No schema transition or historical row rewrite is required.

## Sensitive data

Provider secrets, MCP Bearer tokens, and OAuth tokens use approved secure storage and are represented in general DTOs only by safe status or reference data. A direct MCP Bearer value is a write-only privileged save input and is never part of the persisted or returned server configuration. Raw credentials, prompts, tool payloads, private file contents, and provider responses are not copied into analytics, crash reports, logs, migration fixtures, or diagnostics exports.
