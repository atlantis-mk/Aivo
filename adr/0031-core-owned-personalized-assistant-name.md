# ADR-0031: Store the personalized assistant name in global application configuration

- Status: Accepted
- Date: 2026-08-26
- Related Work: `CHG-2026-059-personalized-assistant-name`
- Closes OPEN: none

## Context

Initialization needs to let the user name the software and then use that same name for the built-in Assistant and the desktop home presentation. Renderer-only storage would make the value unavailable to other trusted desktop consumers, create inconsistent defaults across windows, and bypass the existing Core-owned initialization transaction. The current `app_config` row already owns global initialization state and the initial workspace, but it has no bounded identity field.

## Decision

- Core MUST own one non-secret global `appName` value in `AppConfig` and persist it in `app_config.app_name`.
- The initialization contract MUST accept the name together with the confirmed initial workspace and MUST reject blank, control-character-containing, or overlong values before changing initialized state.
- The canonical default MUST be `Aivo`. Missing historical values and omitted compatible-client input MUST resolve to that default.
- Schema v10 MUST add the column only after producing or verifying a recoverable schema-v9 backup, and the migration MUST remain transactional and idempotent.
- The renderer MUST treat Core's returned value as authoritative for the built-in Assistant label, empty-home title, projectless application title, and browser-window document title.
- The personalized name MUST NOT rename stable Agent IDs, RPC methods, database/application-data paths, package identity, protocol headers, or other technical `Aivo` identifiers.

## Rationale

- Global application configuration already has the correct lifetime, trusted ownership, and multi-window distribution path.
- A compatibility default lets existing installations and older initialization callers upgrade without an extra blocking flow.
- Keeping technical identities stable avoids breaking persisted references, filesystem compatibility, and external contracts.

## Consequences

- `CompleteInitialization` and `AppConfig` gain an additive public field.
- SQLite advances to schema v10 and requires v9 backup, forward-migration, idempotence, and failure-recovery coverage.
- Desktop presentation reads one shared name while product/infrastructure branding remains unchanged.

## Rejected alternatives

- Renderer local storage: it is window-local adapter state and would duplicate Core-owned initialization truth.
- Reusing an existing JSON configuration column: it obscures ownership and couples identity to an unrelated feature payload.
- Renaming the built-in `assistant` mode ID: it would break stable references and changes behavior beyond presentation.

## Verification

`AT-IDENTITY-001` covers defaulting, validation, persistence, migration, initialization ordering, and consistent desktop presentation. `AT-UI-001` covers responsive, keyboard, and error states of the naming step.
