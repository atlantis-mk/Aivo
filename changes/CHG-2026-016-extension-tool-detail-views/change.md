# Embed extension tool pages in the contextual inspector

## Problem or goal

Manifest v2 lets an extension associate isolated Web views with contributed tools, but the desktop currently opens those views only as separate windows and does not connect tool calls to the contextual right-side inspector. A registered tool page should behave like a custom tool detail: open in the existing inspector when the tool runs, close with that contextual surface, and open again when the user selects the tool later.

## Expected behavior

`REQ-EXTENSION-001` requires the Host to derive an optional bounded view reference from a trusted Manifest v2 `views[].tools` association; extension-returned data cannot choose another extension, route, or privileged surface. `REQ-TOOL-002` requires a referenced `tool-detail` view, or a tool-associated `page` fallback, to open in the inspector's overlapping detail layer when the call becomes visible. While the detail layer remains open, selecting or automatically receiving another call with the same Host-resolved extension ID, View ID, and surface reuses the existing WebContents and delivers a bounded revisioned context update without navigation or loading fallback. Closing the detail/inspector, changing conversation, or switching View identity tears down the Web contents, and reopening recreates it. Missing, stopped, removed, or failed views fall back to the native safe summary.

## Non-goals

No built-in browser, arbitrary URL embedding, persistent workspace panel, new manifest version, extension-authored privileged IPC, renderer access to backend URLs or tokens, schema migration, or change to tool execution authority. Dialog, settings, notification, and standalone page-window behavior remain available through their existing Host surface.

## Impact

Core adds Host-owned optional view metadata to extension tool-call records. Electron main/preload add bounded mount, context update, resize, close, and close-notification IPC for one isolated `WebContentsView` per owning main window while retaining standalone windows. The extension preload exposes a revisioned `onContextChanged` event for the same mount. React composes the Web surface into the existing inspector, separates View identity from selected-call context, and preserves responsive native fallback. Persistence stores optional metadata in the existing tool result map; HTTP RPC, database schema, providers, MCP, LSP, worktrees, dependencies, and platform scope do not change.

## Implementation constraints

The Manifest v2 association plus ADR-0002 and ADR-0005 remain authoritative. Tool-detail mapping is deterministic, prefers a declared `tool-detail` surface over `page`, and never trusts an extension-returned identity. The embedded Web contents uses a unique in-memory partition, sandboxing, context isolation, restrictive CSP, no Node integration, blocked permissions/window creation/arbitrary navigation, bounded bridge context, and explicit Core open/close lifecycle accounting. Same-identity context updates are bound to the live mount ID, monotonic revision, and existing bounded safe fields; they cannot change backend, token, actions, route, origin, extension, View, or surface. Renderer layout owns bounds; stale asynchronous mounts and repeated close/update calls are harmless. Native detail remains usable whenever the Web view is unavailable.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `EXT-VIEW-REF-001` | `REQ-EXTENSION-001`, `NFR-SECURITY-001` | Host-derived bounded view references for associated extension tool calls | `AT-EXTENSION-001`, `CT-SECURITY-001` | Completed |
| `EXT-VIEW-HOST-001` | `REQ-EXTENSION-001`, `NFR-RELIABILITY-001` | Isolated embedded WebContents lifecycle and bounded IPC | `AT-EXTENSION-001`, `CT-RELIABILITY-001` | Completed |
| `EXT-VIEW-UI-001` | `REQ-TOOL-002`, `NFR-UI-001` | Automatic and repeatable custom tool detail inside the responsive inspector | `AT-TOOL-002` | Completed |
| `EXT-VIEW-CONTEXT-001` | `REQ-TOOL-002`, `REQ-EXTENSION-001`, `NFR-SECURITY-001`, `NFR-RELIABILITY-001` | Same-identity WebContents reuse with revisioned bounded context updates | `AT-TOOL-002`, `AT-EXTENSION-001`, `CT-SECURITY-001`, `CT-RELIABILITY-001` | Completed |
| `EXT-VIEW-QA-001` | `NFR-SECURITY-001`, `NFR-RELIABILITY-001`, `NFR-UI-001` | Focused tests plus docs/core/lint/build and Electron syntax gates | `AT-TOOL-002`, `AT-EXTENSION-001`, `CT-SECURITY-001`, `CT-RELIABILITY-001` | Pending |

## Acceptance and evidence

- A running or completed extension tool with an associated `tool-detail` view automatically opens that page inside the selected activity's right-side detail layer; `page` is the fallback only when no `tool-detail` surface is declared.
- A newly selected call with the same extension/View/surface keeps the existing WebContents and page-local state, emits one latest-wins bounded context revision, and updates content without showing the mounting skeleton or navigating the logical URL; a different View identity still tears down and mounts a new isolated instance.
- Closing the inspector, changing conversation, selecting Back, and selecting the same tool again deterministically remove and recreate the embedded Web contents without leaked Core view counts or stale overlays.
- Bounds follow wide pushed-aside and narrow overlay layouts without covering the Host-owned title, Back, or Close controls; focus and native timeline behavior remain available.
- Malformed or extension-authored view references, unavailable runtimes, load failures, and removed historical views never expose backend origins/tokens and render the native bounded tool summary with an actionable unavailable state.
- Permissions, navigation, popups, Node integration, privileged renderer APIs, storage lifetime, CSP, context size, and lifecycle accounting retain ADR-0002 isolation.
- Applicable gates are focused Go/renderer tests, `node --check` for Electron scripts, `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, and `pnpm build`; manual wide/narrow Electron interaction remains required before `Verified`.

Implementation evidence recorded on 2026-08-06: Core derives the custom view from the exact frozen extension generation, prefers `tool-detail` over the `page` fallback, persists only the bounded Host reference, and excludes it from model-visible tool-result JSON. The renderer automatically selects the newest referenced call, mounts one isolated Electron `WebContentsView` inside the overlapping detail card, tears it down on Back/Close/unmount, and retains the native details as a failure fallback. Same-identity selection now leaves the mount effect intact, sends the latest bounded tool context through the current mount ID, and Electron main increments and publishes the context revision without resolving, navigating, opening, or closing the View. Activity transitions also derive the next selected View during render instead of rendering the stale selected-call ID as an intermediate null detail; this removes the one-frame unmount/close/remount cycle before the selection effect commits. The bundled UI test extension subscribes to that event and rejects stale revisions and stale HTTP state responses while retaining page-local interaction state. A real Electron smoke harness loaded extension HTML through the isolated custom-protocol session, verified Core open/close lifecycle counts, and passed after confirming teardown. Focused same-identity/context and atomic activity-selection tests, the complete Go suite, desktop script tests, Electron syntax checks, docs validation, lint, and production build pass; final wide/narrow visual interaction screenshots remain pending, so `EXT-VIEW-QA-001` and this Work remain `Pending`/`Implementing`.

## Security and data lifecycle

The persisted reference contains only extension ID, manifest view ID, selected Host surface, and optional safe title. Backend URLs, bearer tokens, credentials, raw Provider data, and ambient browser state stay in Electron main/Core memory. Every embedded view uses an ephemeral session whose storage is cleared on close. The bridge context and its revisioned updates contain only bounded operation/session/turn identifiers and the tool name; tool arguments and results are not passed to the Web view or logged by this feature.

## Compatibility and migration

No database migration. Existing calls without a view reference and extensions without associated views keep the native detail UI. Existing `openExtensionView` callers and standalone surfaces remain compatible. The additive context event does not change `getContext`; pages that do not subscribe retain their initial context until reopened. Rollback restores per-call remount behavior and removes the context-update IPC/event; stored references remain inert bounded map fields.

## Bug root cause (type=bug only)

N/A.
