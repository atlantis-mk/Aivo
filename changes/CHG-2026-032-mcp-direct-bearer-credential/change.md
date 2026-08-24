# Configure MCP bearer credentials directly in desktop settings

## Problem or goal

The desktop MCP form accepts only an environment-variable name for Bearer authentication. An Electron app launched outside a terminal commonly does not inherit shell-only variables, so a correctly entered variable name still fails at probe time. A native user needs to enter a token value directly without persisting that value in renderer storage or SQLite.

## Expected behavior

- `REQ-EXTENSION-001`: the desktop MCP form offers both direct Bearer-token entry and the existing environment-variable reference mode.
- Direct entry sends the raw value only in the one privileged `SaveMCPServer` request. Core stores it through the existing `SecretStore`, persists only a generated Host reference, clears the request value, and uses the resolved value only while authorizing the MCP request.
- MCP list/save results expose only whether a credential reference exists; editing never returns or pre-fills the raw token. Leaving the direct-token field empty preserves an existing stored credential.
- Environment-variable Bearer and OAuth behavior remain compatible.
- Save failure restores the prior secret state or removes a newly created secret reference; probe failures remain actionable without including the token.

## Non-goals

- Exposing raw credentials to the Agent, MCP catalog, renderer persistence, diagnostics, logs, JSON export, or edit forms.
- Replacing the existing local `SecretStore`, adding an OS-keychain dependency, changing OAuth authorization, or removing environment-variable mode.
- Adding a persistence column or schema migration.

## Impact

- Renderer: adds an explicit Bearer-token mode with a password input and transient component state; import accepts an explicit top-level `bearerToken` value only for immediate save.
- Go domain/app/RPC: adds one write-only raw-token field to `SaveMCPServerInput`, prepares and resolves a Host-owned reference, and never includes raw values in `MCPServerConfig` output.
- Persistence: reuses the existing MCP access-token-reference field; no schema or row migration.
- MCP runtime: Bearer authorization prefers the resolved secure-store token, while existing `bearerTokenEnv` remains supported.
- Electron privilege boundary, providers, extensions, skills, LSP, terminals, worktrees, dependencies, and release packaging are unchanged.

## Implementation constraints

- Core validates the source ID and auth mode before writing the secret. The reference is derived from the normalized server ID and cannot be supplied by the renderer.
- A raw token must never be copied into `MCPServerConfig`, returned DTOs, persistence fields, error strings, diagnostics, logs, fixtures, or screenshots.
- Secret write plus configuration save is compensating: a failed save restores the previous value/reference state. Repeated saves without a new raw token retain the current reference.
- Switching away from direct Bearer removes the unused direct bearer reference only after the configuration save succeeds.
- Existing environment-variable configurations remain byte-compatible and continue to resolve in Core.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `MCP-CRED-CONTRACT-001` | `REQ-EXTENSION-001`, `NFR-SECURITY-001` | Add write-only save input and safe credential-reference output behavior | `AT-EXTENSION-001`, `CT-SECURITY-001` | Complete |
| `MCP-CRED-CORE-001` | `REQ-EXTENSION-001`, `NFR-RELIABILITY-001` | Store, resolve, preserve, replace, and compensate bearer secrets | `AT-EXTENSION-001`, `CT-RELIABILITY-001` | Complete |
| `MCP-CRED-UI-001` | `NFR-UI-001`, `NFR-SECURITY-001` | Add direct-token mode, password input, validation, and non-prefill behavior | `AT-UI-001`, `CT-SECURITY-001` | Complete |
| `MCP-CRED-VERIFY-001` | `NFR-SECURITY-001`, `NFR-RELIABILITY-001` | Run focused/full tests, lint/build, secret review, and responsive acceptance | `AT-EXTENSION-001`, `AT-UI-001`, `CT-SECURITY-001`, `CT-RELIABILITY-001` | In progress |

## Acceptance and evidence

- Happy path proves a direct token is stored only by reference and is injected into a Streamable HTTP/SSE Authorization header during probe and later calls.
- Compatibility proves environment-variable Bearer and OAuth sources remain unchanged.
- Boundary tests cover blank token, invalid source ID, repeated save without replacement, replacement, mode switching, secret-store failure, persistence failure compensation, and sanitized probe errors.
- UI acceptance covers direct/reference selection, password masking, disabled submit for a missing new credential, existing-credential preservation, dialog reset, JSON import, and wide/narrow layouts.
- Applicable gates are `pnpm docs:check`, focused Go and desktop tests, `pnpm test:core`, `pnpm lint`, `pnpm build`, `git diff --check`, and a repository secret review.

### Verification evidence

- `core/app/mcp_bearer_secret_test.go` passes focused coverage for reference-only persistence/output, blank-edit preservation, runtime Authorization injection, environment-mode cleanup, and secret compensation after configuration-save failure.
- `apps/desktop/tests/extension-settings-mcp-credential.test.ts` passes direct-token/reference validation, direct-vs-environment normalization, and existing-reference edit state. `pnpm lint` and `pnpm build` pass; lint reports only the repository's pre-existing shared-UI Fast Refresh warnings, and Vite reports existing barrel/chunk-size advisories.
- `pnpm test:core` passes every Go package. `pnpm docs:check` passes with 17 ADRs and 32 Work Packages. `git diff --check` passes.
- In-app browser acceptance on 2026-08-11 covers 1280x900 and 390x844. The direct Bearer form remains contained and readable, the credential field is `type=password` with `autocomplete=new-password` and `spellcheck=false`, and there are no page console errors.
- `pnpm scripts:test` runs the new MCP tests successfully but the full command remains red because the pre-existing `extension-install-picker.test.ts` regular expression scans beyond the `aivo:select-extension-directory` handler into the separate combined file-or-directory handler and sees its `openFile`. The actual extension handler still declares only `openDirectory`. This unrelated dirty-worktree failure prevents moving this Work to `Verified` and sealing it in this task.

## Security and data lifecycle

The raw token exists only in the password input's transient React state, the local privileged RPC request body, and short-lived Core memory. Core writes it to the existing restricted local secret store under a derived `mcp-auth/<server>/access-token` reference, clears it from the save path, and persists/returns only the reference or configured status. The value is never enumerated, logged, diagnosed, exported, backed up with SQLite, included in model context, or shown after save. Replacement and failed saves clean up or restore secret-store state deterministically.

## Compatibility and migration

No schema migration. Existing `bearerTokenEnv` rows and OAuth token references remain valid. The additive local RPC input is ignored by older clients; rolling back the feature leaves secure-store entries inert but existing reference-backed sources require the new runtime to resolve direct Bearer credentials.

## Bug root cause (type=bug only)

N/A; this security Work adds an approved desktop credential-entry path while preserving the former environment-reference path.
