# Enforce global tool enablement during automatic injection

## Problem or goal

The global tool settings switch currently writes current-session activation, while Host automatic resolution independently considers every registered eligible tool. A user can therefore see a tool switched off in the global settings surface and still have it injected into a later Provider request.

## Expected behavior

Global tool configuration owns whether an exact canonical tool name is eligible anywhere. A globally disabled tool remains visible and re-enableable in global settings, but is absent from conversation activation choices, auxiliary resolution, automatic and warm activation, Provider declarations, and Tool Snapshots. Conversation switches remain session-scoped and cannot override global disablement. Missing configuration keeps registered tools enabled for compatibility.

## Non-goals

No extension/MCP/Skill lifecycle redesign, permission-policy change, credential change, tool-name alias, destructive data migration, or change to historical tool-call rendering.

## Impact

Core persists a bounded disabled-name set in a dedicated global preference table, exposes one typed local RPC, annotates the global catalog, and enforces the set before Host resolution and immutable snapshot assembly. The renderer global tool tab uses that RPC; the conversation dialog continues using session activation APIs. Electron remains a typed pass-through. SQLite advances from schema v4 to v5 with the required backup and transaction behavior. Provider adapters, secrets, processes, packaging, and platform scope are unchanged.

## Implementation constraints

Core is authoritative. Names must satisfy the existing Provider-safe canonical contract. Writes are idempotent, sorted, and bounded. Configuration-read failure must not silently expose a possibly disabled tool. Source-level extension/MCP/Skill enablement remains an additional prerequisite. Re-enabling restores eligibility but does not rewrite any conversation selection. Runtime execution remains snapshot-bound.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `GLOBAL-TOOL-DOC-001` | `REQ-TOOL-001`, `REQ-EXTENSION-001` | Requirement, ADR, architecture, data, security, and traceability agree | `AT-TOOL-001`, `AT-EXTENSION-001` | Completed |
| `GLOBAL-TOOL-CORE-001` | `REQ-TOOL-001`, `NFR-SECURITY-001` | Persistent global state filters catalog, resolver, Provider tools, and snapshot | `AT-TOOL-001`, `CT-SECURITY-001` | Completed |
| `GLOBAL-TOOL-UI-001` | `REQ-SESSION-001` | Global and conversation switches call distinct APIs | `AT-SESSION-001` | Completed |
| `GLOBAL-TOOL-QA-001` | all | Focused and full gates plus global/session UI acceptance | all | Pending |

## Acceptance and evidence

- A fresh or legacy configuration reports registered tools enabled.
- Disabling one exact tool persists across restart, leaves it visible as disabled in global settings, hides it from conversation selection, and excludes it from auxiliary candidates, automatic/warm/manual assembly, Provider declarations, and Tool Snapshot.
- Re-enabling restores catalog and automatic eligibility without changing session selections.
- Source-disabled extensions/MCP/Skills remain unavailable regardless of per-tool state.
- Invalid, bridge, unknown, duplicate, excessive, repeated, concurrent, cancelled, and failed updates are safely handled; secrets and payloads are unaffected.
- Focused Core tests, renderer model coverage, `pnpm test:core`, `pnpm lint`, `pnpm build`, `pnpm docs:check`, and `git diff --check` provide automated evidence. Interactive global/session distinction remains required before verification.

Implementation evidence recorded on 2026-08-07: schema v5 adds the bounded `global_tool_preferences` override table with verified v4 backup, transaction, invalid-backup refusal, idempotent reopen, persistence, and re-enable coverage. `SetGlobalToolEnabled` is distinct from session activation; the global renderer tool tab calls it and reloads the authoritative catalog, while the conversation dialog retains its session APIs. Core annotates disabled entries for management, filters them before Host candidates and automatic selection, and assembles Provider tools and Tool Snapshots only from the filtered specs. Tool replay applies the same global filter, and snapshot-bound runtime execution rejects a stale or forged call that was not advertised in the current turn. Focused app/persistence tests, `pnpm test:core`, `pnpm lint`, `pnpm build`, `pnpm docs:check`, and `git diff --check` passed. Lint and build retained only the repository's existing Fast Refresh, large-barrel, and chunk-size warnings. Interactive Electron switch/restart acceptance remains pending, so this Work stays `Implementing`.

## Security and data lifecycle

Only canonical disabled tool names and timestamps are persisted in a non-secret global preference table. No prompt, payload, result, credential, filesystem content, or provider response is added to persistence or logs. Global disablement is a deny boundary reinforced by immutable Tool Snapshot execution.

## Compatibility and migration

Schema v5 adds `global_tool_preferences` after creating or verifying the v4 database backup. No row means enabled; disabled canonical names receive one bounded override row. Downgrade to a v4 binary leaves the unknown table intact but ignores it and may re-expose tools, so rollback must be described to users who rely on global disablement.

## Bug root cause (type=bug only)

N/A.
