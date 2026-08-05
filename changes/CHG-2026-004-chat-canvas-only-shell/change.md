# Remove auxiliary chat workspace panels

## Problem or goal

The sidebarless chat shell still mounts a resizable bottom terminal and right activity panel with corresponding top-right controls. Remove those auxiliary surfaces so the workspace is a single chat canvas.

## Expected behavior

`REQ-WORKSPACE-001` defines a chat workspace without persistent left, right, or bottom panels. The main canvas and compact top bar remain, while terminal, right-panel, maximize, and resize controls are absent. Existing agent tool execution and terminal runtime contracts are unchanged.

## Non-goals

No deletion or modification of terminal, tool-activity, persistence, process, API/RPC/IPC, or permission services. No change to the conversation, history, composer, provider, worktree, plugin, or settings flows.

## Impact

Only Electron renderer composition is affected. The Go core, Electron main/preload, generated bindings, schemas, credentials, provider execution, tool lifecycle, terminal processes, persistence, packaging, and platform scope are unchanged.

## Implementation constraints

Unmount the bottom and right panel compositions rather than changing their services. Remove their triggers and resize handles from the chat shell. Preserve the top bar, conversation canvas, composer, history, project context, permissions, and responsive overflow.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `CANVAS-DOC-001` | `REQ-WORKSPACE-001` | Requirement, scope, and traceability reflect the canvas-only shell | `AT-WORKSPACE-001` | Completed |
| `CANVAS-SHELL-001` | `REQ-WORKSPACE-001` | Right/bottom panels, controls, and resize handles are not mounted | `AT-WORKSPACE-001` | Completed |
| `CANVAS-QA-001` | `REQ-WORKSPACE-001` | Lint/build and user visual verification | `AT-WORKSPACE-001` | Pending |

## Acceptance and evidence

- The project chat workspace contains no right activity panel, bottom terminal panel, related resize handles, or their top-right buttons.
- The main chat canvas consumes the available workspace.
- New conversation, history, project context, plugins, settings, Worktree, conversation actions, and composer controls remain reachable.
- Existing terminal and tool runtime services, process ownership, cancellation, teardown, persistence, and security behavior are unchanged because their UI compositions are only unmounted.
- `pnpm docs:check`, `pnpm lint`, and `pnpm build` pass without new warnings; final visual verification remains user-owned.

Implementation evidence recorded on 2026-08-01: the chat shell no longer mounts `ProjectWorkspaceRightSidebar`, `ProjectWorkspaceBottomPanel`, their providers, resize handles, or floating terminal/right-panel controls. `pnpm build`, `pnpm lint`, `pnpm docs:check`, and `git diff --check` passed; lint reported only the repository's existing Fast Refresh warnings. `CANVAS-QA-001` remains pending for user visual verification, so this Work stays `Implementing`.

## Security and data lifecycle

No secret, prompt, project data, persistence, logging, clipboard, crash, or backup behavior changes. Existing privileged-service boundaries remain intact.

## Compatibility and migration

No data, schema, configuration, API, RPC, IPC, or dependency migration. Rollback remounts the renderer panel composition.

## Bug root cause (type=bug only)

N/A.
