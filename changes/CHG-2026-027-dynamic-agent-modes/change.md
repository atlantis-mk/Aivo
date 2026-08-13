# Manage dynamic Agent modes

## Problem or goal

Agent modes are currently hard-coded in Core and can only be overlaid indirectly through runtime configuration files. Users need to see and manage selectable modes in the existing “Extensions and MCP” surface, create their own modes, edit built-in modes, inspect complete definitions, and delete user-created modes without editing files.

## Expected behavior

`REQ-AGENT-001` makes Core the global source of truth for durable Agent-mode definitions. The management page lists visible built-ins and user-created modes, supports validated create/read/update/delete operations, marks origin, and immediately refreshes the conversation picker. Editing a built-in stores an override without mutating code defaults; deleting a user mode is refused while a session references it. Hidden internal worker modes remain code-owned and are not manageable. Project runtime files remain a higher-precedence project-only compatibility overlay.

## Non-goals

No provider, tool, permission, extension, MCP, Skill, Electron privilege, cloud sync, account, or release packaging redesign. This Work does not expose hidden summary/title/scheduler workers, edit project files, or add arbitrary scripts to a mode.

## Impact

Core adds bounded mode persistence, catalog composition, CRUD service methods, typed local RPCs, schema-v6 migration, and restart-safe resolution. The renderer adds an Agent modes tab, searchable list, editor, validation/error/loading/empty states, and refreshes the existing mode picker through current list calls. SQLite stores definitions and timestamps but no secrets. Existing v5 global-tool work remains intact; the migration builds from it. Cancellation and process ownership are unaffected.

## Implementation constraints

Core owns validation and effective composition. IDs are stable lowercase mode identifiers. Built-in defaults remain in code; saving the same ID writes an override, while deletion of a built-in restores its default rather than removing required behavior. Custom deletion is transactional and refuses referenced modes. At least one visible primary/all mode must remain available. Hidden internal IDs are protected. Global persisted definitions are applied before project runtime overlays. Failure to read durable definitions fails catalog management/resolution rather than silently using a contradictory renderer cache.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `AGENT-MODE-DOC-001` | `REQ-AGENT-001` | Work, Requirement, ADR, scope, architecture, data, test, and traceability agree | `AT-AGENT-001` | Completed |
| `AGENT-MODE-STORE-001` | `REQ-AGENT-001`, `NFR-RELIABILITY-001` | Schema-v6 migration and bounded transactional definition store | `AT-AGENT-001`, `CT-RELIABILITY-001` | Completed |
| `AGENT-MODE-CORE-001` | `REQ-AGENT-001`, `REQ-SESSION-001` | Effective catalog and typed CRUD RPCs drive runtime resolution | `AT-AGENT-001`, `AT-SESSION-001` | Completed |
| `AGENT-MODE-UI-001` | `REQ-AGENT-001`, `NFR-UI-001` | Responsive Agent-mode list/editor in Extensions and MCP | `AT-AGENT-001`, `AT-UI-001` | Completed |
| `AGENT-MODE-QA-001` | all | Focused and full gates plus wide/narrow UI evidence | all | Completed |

## Acceptance and evidence

- Fresh state lists current visible built-ins with built-in origin and no persisted duplicate rows.
- Create, inspect, update, restart, search, and delete of a custom mode preserve all supported fields and refresh the conversation picker.
- Editing a built-in persists an override; deleting that override restores the exact code default.
- Invalid/reserved/duplicate identifiers, blank names/prompts, invalid numeric bounds, unsupported mode/scope, excessive definitions, unavailable toolsets, and hidden internal IDs are refused with actionable errors.
- Deleting a referenced custom mode is refused without partial mutation; repeated save/delete is deterministic.
- Project runtime definitions retain project-only higher precedence and are not silently rewritten by global CRUD.
- Schema v5 receives a verified backup before v6; fresh, upgrade, repeated-open, invalid-backup, and downgrade effects are covered.
- `pnpm docs:check`, focused Core/renderer tests, `pnpm test:core`, `pnpm lint`, `pnpm build`, and `git diff --check` provide automated evidence. Wide and narrow management screenshots are required before verification.

Verification evidence on 2026-08-07:

- `pnpm docs:check` passed with 64 Markdown files, 28 YAML files, 19 Requirements, 19 Test IDs, 13 ADRs, and 27 Work Packages before this Work was sealed.
- `pnpm scripts:test` passed all script, desktop model, search, activation, extension example, and governance tests, including Agent-mode origin/prompt/model/toolset search.
- `pnpm test:core` passed every Go package. Focused Agent-mode tests cover built-in origin, custom create/update/delete, restart persistence, project overlay precedence, referenced deletion refusal, built-in override/reset, protected worker IDs, and validation; persistence tests cover v5-to-v6 backup and invalid-backup refusal.
- `pnpm lint` passed with only the repository's existing Fast Refresh export warnings in shared UI and root route files.
- `pnpm build` passed TypeScript compilation and the Vite production build; `git diff --check` passed.
- In-app browser QA at 1280 x 900 and 390 x 844 verified the searchable built-in/custom cards, origin and override badges, create/edit/delete/reset controls, and scroll access to every editor field. A discovered narrow-screen footer overlap was corrected by clipping the editor scroll region and then reverified. Browser logs contained no warnings or errors.
- Screenshots: `evidence/agent-modes-wide.png`, `evidence/agent-modes-narrow.png`, `evidence/agent-mode-editor-narrow.png`, and `evidence/agent-mode-editor-narrow-scrolled.png`.

## Security and data lifecycle

Mode display metadata, prompts, model references, sampling options, toolsets, and permission scope are non-secret local configuration owned by Core. Renderer submissions are untrusted and fully revalidated. The table stores no credentials, user conversation content, tool payloads/results, private file contents, or provider responses; operations log no raw prompt body.

## Compatibility and migration

Schema v6 adds `agent_mode_definitions` after creating or verifying the v5 database backup. Missing rows preserve code defaults. Built-in edits are full durable replacements that can be removed to restore defaults. A v5 binary ignores the new table and therefore uses code/runtime-file definitions; users must be warned that Agent-mode edits disappear during downgrade. Project `.aivo` definitions remain supported and override the global effective catalog only for that project.

## Bug root cause (type=bug only)

N/A.
