# ADR-0005: Reuse an embedded extension View while its Host-owned context changes

- Status: Accepted
- Date: 2026-08-06
- Related Work: `CHG-2026-016-extension-tool-detail-views`
- Closes OPEN: none

## Context

The contextual inspector automatically selects the latest extension-backed tool call. Each call has a new operation ID, and the current renderer therefore tears down the existing isolated `WebContentsView`, creates a new ephemeral session, reloads the same logical URL, and briefly displays a loading state. This discards harmless page state and visibly flashes even when the extension ID, View ID, surface, origin, actions, and security policy are unchanged. Tool-call context still must remain Host-owned and bounded; an extension page cannot choose another call, origin, or privileged surface.

## Decision

- The desktop MUST identify an embedded View instance by its owning window plus Host-resolved extension ID, View ID, and surface, not by tool-call operation ID.
- While that identity remains unchanged and the detail layer stays open, the existing isolated session and `WebContentsView` MUST remain mounted when the selected tool call changes.
- The renderer MAY submit only the existing bounded operation, session, turn, and tool-name context fields through a dedicated update IPC bound to the current mount ID.
- Electron main MUST replace the stored context atomically, increment a monotonic per-mount revision, and notify only that isolated View WebContents through the versioned preload bridge.
- The extension preload MUST expose an additive `onContextChanged` subscription whose payload matches `getContext` and includes the revision. It MUST NOT expose arbitrary IPC or backend/token data.
- A stale mount ID, closed/replaced View, oversized context, different extension/View/surface, or late update MUST be ignored or rejected without affecting the current View.
- Switching View identity, selecting Back, closing the inspector, changing conversation, extension stop/failure, renderer loss, or explicit guest close MUST retain deterministic teardown and recreate-on-reopen behavior.

## Rationale

- Reusing the same WebContents removes reload flashes and preserves local UI state while keeping one isolated View per owning window.
- A Host-owned revisioned context event gives pages an explicit latest-wins update path without navigation, ambient data, or privileged renderer access.
- Mount-bound updates prevent stale React effects from retargeting a replacement View.

## Consequences

- Extension pages that want live tool-call switching should subscribe to `onContextChanged`; pages that only call `getContext` remain compatible but keep their initial content.
- React must separate View identity lifecycle from selected-call context lifecycle.
- Electron main and both preloads gain additive IPC/bridge surface that requires bounds, stale-update, teardown, and same-identity regression coverage.
- Different View identities still reload by design.

## Rejected alternatives

- Recreate the View for every operation: simple ownership but causes visible flashing and loses page state.
- Encode the operation ID in the View URL: navigation still reloads and exposes Host context to URL/history surfaces.
- Let the extension poll arbitrary Aivo state: expands authority and weakens operation ownership.
- Keep one global View across conversations: violates conversation teardown and increases stale-context risk.

## Verification

`AT-TOOL-002` verifies stable same-identity presentation and different-identity teardown. `AT-EXTENSION-001` verifies revisioned bounded context delivery and bridge compatibility. `CT-SECURITY-001` verifies mount ownership, payload bounds, and no endpoint/token exposure. `CT-RELIABILITY-001` verifies latest-wins updates, stale mount refusal, and deterministic close/reopen behavior.
