# ADR-0034: Persist explicit permission-mode preference

- Status: Accepted
- Date: 2026-08-28
- Related Work: `CHG-2026-063-persist-permission-mode-preference`
- Closes OPEN: none

## Context

The desktop permission selector updates only the active session rule. A user who selects full access before creating a conversation, or creates another conversation later, is returned to request approval even though their explicit choice was meant to be retained. Carrying a global setting into existing sessions would silently expand authority.

## Decision

- Aivo MUST persist the native user's selected `request_approval` or `full_access` value as a non-secret global default.
- Aivo MUST initialize each newly created coding conversation with an independent session-level permission rule derived from that default.
- Changing the default MUST NOT modify, remove, or elevate the permission mode of an existing conversation.
- Missing, stale, or invalid stored default values MUST resolve to `request_approval`.

## Rationale

The persistent default matches the user's explicit selection while retaining session rules as the execution authority. Copying the value at session creation avoids retroactively increasing the authority of an existing conversation.

## Consequences

- Schema version 11 adds one non-secret `app_config` column and uses the existing recoverable migration flow.
- Renderer state uses the persisted default whenever no existing conversation is active.
- Core, persistence migration, and desktop build checks must cover persistence, new-session inheritance, compatibility fallback, and unchanged existing sessions.

## Rejected alternatives

- Keep the choice renderer-only: new conversations and restarts lose the explicit preference.
- Reuse the latest session's mode: results vary by history and does not survive an empty workspace.
- Apply preference changes to all sessions: silently escalates authority for existing work.

## Verification

`AT-TOOL-001`, `CT-SECURITY-001`, and `CT-RELIABILITY-001` cover explicit mode validation, default inheritance without escalation, persistence compatibility, and migration backup/failure behavior. `AT-UI-001` covers the desktop selector state and saved new-conversation behavior.
