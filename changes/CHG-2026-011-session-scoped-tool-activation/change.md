# Keep manual tool activation scoped to one conversation

## Problem or goal

In desktop acceptance, manually enabling a Chrome/MCP tool in one conversation makes that tool active in every subsequently created conversation. Expected behavior is conversation isolation. Actual behavior occurs because saving an existing session's active tools also writes the same names to a renderer-global `aivo:default-active-tool-names:v1` preference, and new-session submission copies that preference into each new backend session.

## Expected behavior

- `REQ-SESSION-001`: manually changing tools in an existing conversation changes only that session's active-tool list.
- When the tool dialog is used before a conversation exists, the selection is an in-memory draft for the next created conversation and is consumed exactly once.
- A later new conversation starts without prior manual tools unless the user selects tools for that conversation.
- Existing legacy global default-tool state is discarded and cannot silently reactivate tools after upgrade or restart.
- `REQ-EXTENSION-001`: backend session-pinned, mode, warm, and auxiliary activation sources retain their accepted semantics; this Work removes only renderer-driven cross-session propagation.

## Non-goals

- Do not remove explicit session pinning, bounded same-session warm leases, mode defaults, or auxiliary activation.
- Do not change plugin/MCP installation, enablement, trust, permission, execution, or skill activation behavior.
- Do not add a new global “always enable this tool” product preference.

## Impact

- Renderer state and new-conversation submission change; a pure activation-scope helper and regression test are added.
- Go domain/app, persistence schema, HTTP/RPC shapes, Electron privilege boundaries, Provider adapters, MCP transport, extension protocol, dependencies, packaging, and platform scope are unchanged.
- The obsolete renderer-only localStorage entry is removed; backend active-tool lists already belong to session execution state and need no migration.

## Implementation constraints

- Existing-session saves must call `SetSessionActiveTools` without mutating the one-shot new-conversation draft.
- A draft selected with no active session must stay memory-only, be captured and cleared before the first new-session RPC attempt, and never be reused by another session.
- Clearing the legacy key must not touch unrelated project preferences or backend session metadata.
- Failure to apply the captured draft leaves the created session active and produces the existing error path; it must not restore a global default that can leak later.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `BUG-SCOPE-001` | `REQ-SESSION-001` | Separate existing-session saves from one-shot new-session draft activation | `AT-SESSION-001` | Complete |
| `BUG-SCOPE-002` | `REQ-EXTENSION-001` | Remove global persistence and discard the legacy default-tool key | `AT-EXTENSION-001` | Complete |
| `BUG-SCOPE-003` | `NFR-RELIABILITY-001` | Add pre-fix failing/post-fix passing scope and single-consumption regression | `CT-RELIABILITY-001` | Complete |

## Acceptance and evidence

- A regression fails before the fix because no session/draft scope model exists and the renderer persists a global default.
- Post-fix tests prove existing-session saves have session scope, no-session saves create a draft, consumption returns the normalized selection once and an empty remainder, and stale duplicates/whitespace are normalized.
- Source/build verification proves the existing-session branch no longer updates draft state, the new-session path clears before applying, and the legacy localStorage key is removed rather than loaded.
- Backend session activation isolation, warm leases, mode/auxiliary activation, Provider calls, cancellation, timeouts, teardown, and permissions remain covered by existing tests.
- Persistence migration/rollback, dependency loss, installer/signing, package smoke, and new UI screenshots are N/A; the visible dialog layout is unchanged.
- Pre-fix evidence on 2026-08-03: `node --experimental-strip-types --test src/features/projects/project-tool-activation-scope.test.ts` failed with `ERR_MODULE_NOT_FOUND` because no session/draft activation-scope implementation existed. This was recorded before product-code changes.
- Post-fix focused evidence: `node --experimental-strip-types --test tests/project-tool-activation-scope.test.ts` passed four tests covering existing-session scope, no-session draft scope, normalized single consumption with an empty remainder, and removal of the exact legacy localStorage key.
- Source verification found no remaining `defaultActiveToolNames`, `setDefaultActiveToolNames`, or write path for `aivo:default-active-tool-names:v1`; the key remains only as the explicit removal target. TypeScript build proves the pending-state setter is threaded through new-session creation and the existing-session dialog branch does not update it.
- `pnpm test:core` passed on 2026-08-03 for Core app, CLI, persistence, and HTTP transport packages, retaining backend session pinning, warm-lease, snapshot, cancellation, and extension lifecycle coverage.
- Repository gates passed on 2026-08-03: `pnpm docs:check`, `pnpm scripts:test`, `pnpm lint`, `pnpm build`, and `git diff --check`. Lint reported only existing Fast Refresh warnings; build reported only existing large-barrel and chunk-size advisories.
- Verification platform: macOS 14.8.7, Darwin 23.6.0 x86_64, Node.js 24.12.0.

## Security and data lifecycle

Tool names are the only affected data. The legacy renderer-local list is deleted. New-conversation draft names live only in renderer memory until consumed; active names remain stored only in their backend session execution state. No credentials, arguments, results, prompts, or private filesystem data are added to storage or logs.

## Compatibility and migration

On first load of the corrected renderer, `aivo:default-active-tool-names:v1` is removed. Existing backend conversations retain their explicit active tools. There is no schema/API/RPC migration. Rollback can recreate the global propagation bug but cannot recover the intentionally discarded obsolete preference.

## Bug root cause (type=bug only)

Affected version: `0.0.0-development`. `submitActiveToolNames` updated `defaultActiveToolNames` even after saving an existing session, the Zustand preference persisted it globally, and `submitPrompt` copied it into every new session. Existing tests covered backend session pinning but not the renderer's cross-session orchestration or localStorage lifecycle. The fix replaces the persisted global default with a memory-only pending list, keeps existing-session saves isolated, consumes and clears the pending list during the next session creation, and removes the obsolete storage key. The focused regression failed before the scope implementation existed and passed afterward. Fix version: `0.0.0-development`.
