# Adopt a sidebarless chat workspace

## Problem or goal

The project chat screen reserves a persistent left column for projects and conversations even when the user is focused on composing or reading a task. Replace the persistent sidebar with a full-width chat workspace whose navigation is available from compact top-bar entry points, and align the empty state with Aivo's accepted HIG-derived visual scale.

## Expected behavior

`REQ-WORKSPACE-001` defines a sidebarless default chat workspace. The desktop shell uses a full-width top bar for new conversation, history, project context, plugins, and settings entry points. The empty state keeps a compact title and supporting prompt actions in the main canvas, while the functional composer remains near the bottom and retains attachments, permission mode, agent mode, project selection, model selection, microphone, and submit behavior. `CHG-2026-004-chat-canvas-only-shell` subsequently removes the right activity and bottom terminal panel compositions while leaving their underlying runtimes unchanged.

## Non-goals

No change to conversation, project, provider, permission, tool, terminal, persistence, API/RPC/IPC, or model-selection contracts. No new backend service, database migration, dependency, cloud behavior, built-in browser, or native AppKit shell. The selected visual target is the second ImageGen option displayed on 2026-08-01; it does not authorize unrelated project-page redesigns.

## Impact

The Electron renderer project workspace layout, top bar, empty state, and composer positioning are affected. The Go core, Electron main/preload, generated bridge bindings, schemas, credentials, provider execution, tools, MCP/LSP, worktrees, release packaging, and supported platform scope are unaffected.

## Implementation constraints

Use the semantic HIG theme from `CHG-2026-002-desktop-hig-theme`, existing Hugeicons assets, and existing feature composition outside `apps/desktop/src/components/ui`. Preserve keyboard focus, responsive overflow, conversation/history reachability, project selection, and visible task/permission states. The persistent left sidebar must not consume layout width in the default chat state.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `WORKSPACE-DOC-001` | `REQ-WORKSPACE-001` | Primary Requirement and traceability row | `AT-WORKSPACE-001` | Completed |
| `WORKSPACE-SHELL-001` | `REQ-WORKSPACE-001` | Full-width shell and compact top navigation | `AT-WORKSPACE-001` | Completed |
| `WORKSPACE-EMPTY-001` | `REQ-WORKSPACE-001` | Empty state and composer match the selected option | `AT-WORKSPACE-001` | Completed |
| `WORKSPACE-QA-001` | `REQ-WORKSPACE-001` | Wide/narrow screenshots, interaction checks, lint/build evidence | `AT-WORKSPACE-001` | Pending |

## Acceptance and evidence

- No persistent left sidebar or sidebar-width gap is visible on the chat workspace.
- New conversation, history, plugins, settings, project selection, model selection, permission mode, agent mode, attachment, microphone, and submit entry points remain reachable or retain their existing behavior.
- The empty state follows the selected option's hierarchy: compact top toolbar, centered title/supporting actions, and bottom-centered composer.
- Existing conversation, loading, error, permission, question, and cancellation states remain layout-compatible; auxiliary panel acceptance is superseded by `CHG-2026-004-chat-canvas-only-shell`.
- Wide and narrow screenshots show no horizontal overflow or obscured persistent controls; keyboard focus remains visible.
- The history popover contains only a compact, outlined two-line conversation list. It merges ordinary and project conversations, shows the associated project on the second line when present, keeps status or relative time right-aligned, and gives the active conversation a semantic selected background.
- `pnpm docs:check`, `pnpm lint`, and `pnpm build` pass without new warnings.
- Failure, cancellation, repetition, timeout, teardown, persistence, migration, rollback, provider, and security behavior are unchanged; applicable paths are N/A beyond renderer acceptance.

Implementation evidence recorded on 2026-08-01: `pnpm build`, `pnpm lint`, and `pnpm docs:check` completed successfully; lint reported only the repository's existing Fast Refresh warnings. A 1536×1024 implementation capture is stored at `artifacts/design-qa/workspace-option-2-1536x1024.png`. The history popover was subsequently reduced to a conversation-only shadcn `Item` list using `variant="outline"`; it includes all unarchived root conversations, conditionally shows the associated project, and represents the active conversation with `aria-current="page"` plus a muted background. Its shadcn `ScrollArea` now grows with the number of rows and is capped at `min(72vh, 560px)`, avoiding empty vertical space while keeping long histories scrollable. The Worktree dialog and more-actions menu retain their callbacks, including pin/unpin and archive. Title-bar dragging now belongs to the header container instead of a trailing overlay, while the left, center, right, and floating control groups explicitly remain no-drag and above the drag surface so all controls can receive pointer events. Browser inspection confirmed the popover geometry, absence of horizontal overflow, and absence of console errors, but its local browser state contained no conversations, so populated rows and the selected state still require verification in the Electron app. `WORKSPACE-QA-001` remains pending and this Work stays `Implementing`. After user verification and final evidence, move to `Verified` and run `pnpm work:archive -- CHG-2026-003-sidebarless-chat-workspace`.

## Security and data lifecycle

No secret, credential, prompt payload, project data, persistence, logging, clipboard, crash, or backup behavior changes. Existing renderer-to-privileged-service boundaries remain intact.

## Compatibility and migration

No data, schema, configuration, API, RPC, IPC, or dependency migration. Rollback is the renderer composition revert.

## Bug root cause (type=bug only)

N/A.
