# ADR-0017: Accept direct MCP bearer input through Host-owned secure references

- Status: Accepted
- Date: 2026-08-11
- Related Work: `CHG-2026-032-mcp-direct-bearer-credential`
- Closes OPEN: none

## Context

MCP Bearer authentication currently stores only an environment-variable name and resolves it inside the Go Core process. Desktop applications launched from Finder, Dock, Start Menu, or another graphical shell commonly do not inherit variables configured only in an interactive terminal. Requiring users to arrange a process environment outside Aivo makes otherwise valid remote MCP sources unusable from the desktop.

Accepting a raw Token as ordinary MCP configuration would copy it into renderer-managed server objects, SQLite, list DTOs, diagnostics, and exports. That conflicts with Aivo's Host-owned credential boundary. The existing provider and MCP OAuth paths already accept a secret once, store it through `SecretStore`, and retain only a reference.

## Decision

- The native desktop MCP settings flow MAY accept a raw Bearer token only as a transient write-only field on the privileged save request.
- Core MUST derive the credential reference from the normalized MCP server identity, write the value through `SecretStore`, and persist/return only the Host-owned reference or a non-secret configured status.
- MCP authorization MUST resolve the reference only for the owned request and MUST NOT expose the value to renderer persistence, model context, catalog metadata, SQLite, logs, diagnostics, crash output, fixtures, or edit/read DTOs.
- Saving without a new raw value MUST preserve an existing direct credential. Replacing it or switching authentication modes MUST use compensating cleanup so a failed configuration save restores the prior secret state.
- The existing environment-variable Bearer mode remains supported. OAuth token ownership and conversational MCP proposals remain unchanged; models may carry only credential references, never raw values.
- This change MUST reuse the existing persisted MCP access-token-reference field and MUST NOT introduce a schema migration.

## Rationale

- Direct entry makes graphical desktop launch behavior independent of the user's shell initialization.
- A write-only save field gives the user a practical setup path without making the server configuration itself secret-bearing.
- Reusing the existing secret store and reference field keeps ownership aligned with MCP OAuth and avoids duplicating persistence or migration state.
- Keeping environment references preserves automation and advanced local configurations.

## Consequences

- The renderer temporarily holds the value in password-input state and sends it over the local privileged RPC boundary; it must not persist, rehydrate, or display it after save.
- `SaveMCPServer` gains an additive secret-bearing input that requires explicit redaction discipline even though normal MCP configuration/list DTOs remain non-secret.
- Existing rows require no rewrite. A reference-backed Bearer row still needs a runtime containing this decision to resolve its token after rollback to older application code.
- Replacing the local restricted-file secret store with an OS keychain remains separate Work because it changes credential persistence ownership and platform behavior.

## Rejected alternatives

- Save the token in `headers.Authorization`, `env`, or a new SQLite column: these are general configuration fields and would expose or redact away the usable value.
- Require users to launch Aivo from a configured terminal: this does not solve normal desktop usage and creates platform-specific setup friction.
- Remove environment-variable support: it remains useful for managed and automated environments.
- Add an OS-keychain dependency in this slice: valuable, but broader than the MCP desktop usability problem and requires separate platform acceptance.

## Verification

`AT-EXTENSION-001` verifies direct/reference Bearer resolution and compatibility. `CT-SECURITY-001` verifies write-only transport, reference-only persistence/output, redaction, and cleanup. `CT-RELIABILITY-001` verifies repeated saves and compensation. `AT-UI-001` verifies password masking, non-prefill, validation, reset, and responsive states.
