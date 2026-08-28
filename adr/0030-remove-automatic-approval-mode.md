# ADR-0030: Remove the automatic approval permission mode

- Status: Accepted
- Date: 2026-08-26
- Related Work: `CHG-2026-058-remove-auto-approval`
- Closes OPEN: none

## Context

The permission selector currently offers request approval, automatic approval, and full access. Automatic approval is an intermediate authorization policy that silently allows selected write operations while its label does not enumerate that authority. Keeping it also requires every privileged operation to maintain a distinct and easily misunderstood policy branch.

Removing the selector alone would leave the persisted and RPC-level `auto_approve` value capable of granting authority. Existing local databases may contain that value, so removal needs an explicit fail-closed compatibility rule without a persistence migration.

## Decision

- Aivo MUST expose and accept only `request_approval` and `full_access` as permission modes.
- Core MUST reject new attempts to set `auto_approve` or any other unknown permission mode.
- A historical persisted `auto_approve` mode rule MUST be interpreted as `request_approval` when permission state is read or evaluated. It MUST NOT preserve any automatic allow behavior.
- The desktop MUST omit the automatic-approval option and MUST normalize a stale or unknown mode response to `request_approval`.
- Exact user decisions, remembered denials, non-bypassable safety checks, and the explicit full-access mode remain unchanged.

## Rationale

A two-mode contract makes the authorization choice explicit: operations either follow approval policy or use the separately labeled full-access policy. Fail-closed interpretation prevents a stale database or caller from retaining an authorization path that the current product no longer offers.

## Consequences

- The permission-mode contract is intentionally narrowed during development.
- No schema migration is required because mode state is stored as permission rules and historical values can be interpreted safely at read and evaluation time.
- Historical automatic-approval sessions begin requesting approval on their next applicable operation.
- Core and desktop tests must prove option removal, RPC refusal, and historical-value fallback.

## Rejected alternatives

- Hide only the desktop option: stale renderers, direct RPC callers, and stored rules could retain the removed authority.
- Delete historical rows: unnecessary persistence mutation would add backup and rollback requirements when safe interpretation is sufficient.
- Map historical automatic approval to full access: this would expand authority without a new explicit user choice.

## Verification

`AT-TOOL-001` and `AT-PROJECT-003` verify the two-mode contract, removed-value refusal, historical fail-closed fallback, and unchanged explicit full-access behavior. `CT-SECURITY-001` verifies that the removed mode cannot grant execution through persistence or RPC input. `AT-UI-001` verifies the two-option responsive selector.
