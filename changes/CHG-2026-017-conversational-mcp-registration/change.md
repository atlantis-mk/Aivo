# Register MCP tool sources through Agent conversation

## Problem or goal

Users can configure MCP sources through settings, but cannot ask the Agent to register one conversationally and have it remain eligible in later conversations. The goal is a bounded Host-owned flow in which the Agent translates an explicit user request into an exact MCP registration proposal, the privileged Host presents and authorizes that proposal, and a successfully probed source becomes globally installed and enabled without converting one conversation's manual activation into a global default.

## Expected behavior

- `REQ-EXTENSION-002`: a coding Agent can list MCP source summaries and propose registration of one `stdio`, `streamable_http`, or `sse` source through namespaced Host-owned tools.
- A proposal is non-mutating, contains no raw secret, is bound to the originating turn, expires, and returns the exact command or origin, roots, credential references, requested global enablement, and security-relevant capabilities for native confirmation.
- The model cannot approve, trust, bind credentials, persist, enable, or start the proposed source. A privileged confirmation carrying the exact proposal identity is required.
- Confirmation saves the source disabled, resolves only Host-owned credential references, probes it under cancellation and bounded timeouts, and marks it ready and enabled only after successful capability discovery. Failure leaves no eligible tools and a safe actionable error.
- A ready enabled source is globally installed and therefore eligible in later conversations. Its tools still enter each model request only through the bounded Host pre-call resolver and immutable Tool Snapshot.
- `REQ-SESSION-001`: manual tool activation remains session-scoped; registration does not populate `SessionActiveTools` for other conversations.

## Non-goals

- Generating, compiling, downloading, or executing arbitrary tool source code authored by the model.
- Installing generic Manifest v1 extensions or plugins from paths, URLs, or package registries.
- Letting the model supply raw credentials, approve trust, bypass confirmation, or make a tool mandatory in every model request.
- Adding a global manual-active-tool preference, changing the four unqualified core tools, or changing the MCP protocol.
- Removing or editing registered sources conversationally; destructive lifecycle operations remain in settings.

## Impact

- Go domain/application: add proposal DTOs, an ephemeral turn-owned proposal store, a trusted built-in `aivo.tools` extension, exact validation, confirmation orchestration, and idempotency.
- MCP adapter/persistence: reuse `MCPManager.Save`, probe, capability replacement, and existing `mcp_servers`, `mcp_tools`, prompt, and resource tables. No schema change.
- HTTP/RPC: add a privileged confirmation method. Agent proposal execution remains inside the frozen extension-tool contract.
- Renderer: reuse the native pending-permission interaction for exact confirmation and show a bounded success/failure result; no extension Web view is required.
- Credentials: proposal DTOs carry only environment-variable names or secure-store references. Raw values never enter model-visible arguments, SQLite, renderer persistence, or diagnostics.
- Providers, Execution Environments, worktrees, LSP, release packaging, and dependencies are otherwise unchanged.

## Implementation constraints

- The trusted extension contributes Provider-safe namespaced tools only and does not alter the four unqualified primitive registry.
- Proposal validation is Host-owned and repeated at confirmation. A proposal is immutable, tied to session/turn/tool-call identity, single-use, short-lived, and invalidated by cancellation or source-state conflict.
- `stdio` confirmation displays and authorizes the exact executable, arguments, working directory, roots, and referenced environment names before any child process starts. Remote confirmation displays and authorizes the normalized HTTPS origin and credential references before any connection starts.
- Only supported transports are accepted. Remote URLs require HTTPS except explicitly loopback development endpoints. Commands must be absolute paths or simple executable names without shell evaluation; arguments remain an array and are never joined into a shell command.
- The commit sequence is recoverable: persist disabled, probe, then enable/ready. Probe failure retains a disabled error record for diagnosis or compensates by restoring an existing record exactly; it never exposes partial capabilities as eligible.
- Existing global source enablement controls eligibility across restarts. Existing session-pinned and one-shot draft activation semantics remain unchanged.
- Public errors distinguish invalid proposal, expired/replayed proposal, authorization required, conflict, dependency unavailable, cancelled, and internal failure without leaking secret or private payloads.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `REG-CONTRACT-001` | `REQ-EXTENSION-002` | Define proposal, confirmation, safe result, expiry, and error contracts plus `aivo.tools` manifest | `AT-EXTENSION-002` | Complete |
| `REG-HOST-001` | `REQ-EXTENSION-002`, `NFR-SECURITY-001` | Implement Host validation, turn-owned proposal lifecycle, exact authorization, and idempotent commit | `AT-EXTENSION-002`, `CT-SECURITY-001` | Complete |
| `REG-MCP-001` | `REQ-EXTENSION-001`, `NFR-RELIABILITY-001` | Persist disabled, probe, enable/ready, rollback on failure, and reload into later catalogs | `AT-EXTENSION-001`, `CT-RELIABILITY-001` | Complete |
| `REG-UI-001` | `REQ-EXTENSION-002`, `NFR-SECURITY-001` | Present exact native confirmation and bounded success/failure state | `AT-EXTENSION-002`, `CT-SECURITY-001` | Complete |
| `REG-SCOPE-001` | `REQ-SESSION-001` | Prove global installed-source eligibility without cross-session manual activation | `AT-SESSION-001`, `AT-EXTENSION-002` | Complete |

## Acceptance and evidence

- Happy path covers conversational proposal, exact user confirmation, MCP capability discovery, ready state, restart, and a later conversation resolving one contributed tool.
- Validation covers missing identifiers, invalid names, unsupported transports, shell-shaped commands, malformed/credential-bearing URLs, non-HTTPS remote origins, unsafe headers/environment values, roots, bounds, and duplicate source identity.
- Authorization covers no confirmation, altered proposal, wrong session/turn, replay, expiry, denial, cancellation, and concurrent confirmation.
- Failure and recovery cover unavailable executable/server, timeout, malformed MCP response, capability-name collision, persistence failure, partial capability replacement, and retry without eligible partial tools.
- Security acceptance proves no raw secret reaches the model, renderer persistence, SQLite configuration fields, logs, diagnostics, crash output, or fixtures; no stdio process or remote connection starts before confirmation.
- Scope acceptance proves manual active-tool names remain isolated while installed enabled MCP sources are globally eligible and request-scoped resolver selection remains bounded.
- Applicable gates are `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, `pnpm build`, responsive desktop screenshots, and `git diff --check`.

### Verification evidence

- `core/app/mcp_registration_test.go` covers Host-owned intent selection, no process or network activity before approval, mandatory confirmation under Full Access, denial, replay, unsafe configuration, disabled-on-failure behavior, successful global eligibility, session activation isolation, restart reload, and concurrent single-winner commit.
- `apps/desktop/tests/project-permission-approval-model.test.ts` proves that native confirmation exposes the exact MCP source and global effect without a reusable grant. Browser-controlled Vite acceptance at 1280x720 and 390x844 confirmed the full-access warning, exact command/cwd/roots, independently required approval, scroll containment, and reachable deny/approve actions; the temporary preview DOM and local storage were removed afterward.
- `pnpm diagnostics` passed on 2026-08-06: documentation checks, 4 archive-script tests, 9 desktop model tests, all Go packages, lint, TypeScript compilation, and the Vite production build. Lint reported only the repository's existing Fast Refresh warnings in shared UI and route files; Vite reported only existing barrel/chunk-size advisories.
- `git diff --check` passed after the verified implementation and documentation updates.

## Security and data lifecycle

Unapproved proposals live only in bounded Core memory, are owned by one turn, carry configuration and references rather than raw secrets, and are deleted on denial, expiry, cancellation, successful consumption, or service shutdown. Approved MCP configuration and discovered public capability metadata reuse existing SQLite ownership. OAuth and bearer material remains in the secure store. Logs and persisted tool-call history contain only safe proposal identity, source display metadata, state, and sanitized error codes.

## Compatibility and migration

No schema migration is required. Existing MCP sources continue unchanged, and the new built-in extension plus confirmation RPC are additive development-version contracts. Rollback removes the conversational entry point while leaving already registered MCP sources manageable through existing settings. If durable proposal/audit persistence becomes required, implementation stops and creates a separate versioned migration Work.

## Bug root cause (type=bug only)

N/A; this is new behavior.
