# Generate MCP descriptions from discovered tools

## Problem or goal

An MCP functional description is useful for source-group auxiliary selection, but users currently have to summarize a server's tool catalog manually. The MCP edit flow needs an optional auxiliary-model action that reads the complete current discovered tool catalog and drafts a concise functional description.

## Expected behavior

For `REQ-EXTENSION-001`, the edit form can request a generated description for an existing MCP source. Core loads every currently stored discovered tool for that source, sends only bounded tool names and descriptions to the configured auxiliary model, and returns one bounded plain-text description. The renderer replaces only the draft description; persistence still requires the existing explicit save action. An empty catalog, missing auxiliary model, oversized catalog, cancellation, or provider failure produces an actionable error and does not synthesize or persist content.

## Non-goals

This Work does not probe or start an MCP source, generate extension descriptions, auto-save the result, change automatic-selection membership, expose raw MCP configuration or credentials, or let the auxiliary model call tools.

## Impact

The React MCP edit form, desktop Core client, Go application service, domain RPC DTOs, and local HTTP RPC dispatch change. The existing MCP description persistence field is reused. Electron privilege boundaries, preload IPC, database schema, MCP execution, dependencies, packaging, and release formats do not change.

## Implementation constraints

Core owns catalog lookup, safe projection, bounds, prompt construction, auxiliary-model invocation, output normalization, and refusal. Every discovered tool must be represented; bounds fail the request rather than silently dropping tools. Tool text is untrusted data and cannot become instructions. Endpoint, transport command/arguments, environment, headers, roots, authentication data, credential references, and other server configuration must not enter the provider request. The request inherits cancellation, declares no executable tools, performs no persistence, and returns at most the accepted MCP description limit.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `TASK-035-01` | REQ-EXTENSION-001 | Core generation service and typed RPC read the complete current tool catalog and return a safe bounded draft | AT-EXTENSION-001, CT-SECURITY-001, CT-RELIABILITY-001 | Completed |
| `TASK-035-02` | REQ-EXTENSION-001 | Existing-server edit form generates, reports progress/failure, and changes only the unsaved description draft | AT-EXTENSION-001 | Completed |

## Acceptance and evidence

- Tests must prove all stored tools appear in the provider prompt, sensitive MCP configuration does not, no Provider tools are declared, and the persisted MCP source is unchanged.
- Tests must cover empty catalogs, missing auxiliary configuration, catalog bounds, malformed/oversized model output, provider failure, and cancellation where practical.
- Desktop verification is `pnpm lint` and `pnpm build`; the user retains final visual acceptance of the edit form.
- After final evidence and `Verified`, run `pnpm work:archive -- CHG-2026-035-mcp-description-generation`.

Implementation evidence on 2026-08-11: focused Core tests prove complete deterministic tool-name/description projection, exclusion of server display/configuration values, absence of executable Provider tools, draft normalization, UTF-8 byte bounds, empty/oversized-catalog refusal, missing auxiliary-model refusal, and unchanged persisted description. The focused desktop draft test passes. The MCP edit dialog gives its Radix `ScrollArea` a bounded flex viewport so long generated/configuration content scrolls while the header and save controls remain fixed. `pnpm docs:check`, `pnpm test:core`, `pnpm lint`, and `pnpm build` pass; lint reports only existing Fast Refresh warnings. The broader `pnpm scripts:test` run passes this Work's tests but remains red on the unrelated existing `extension-install-picker.test.ts` regex, which spans into the later combined composer file-or-directory picker and mistakes its `openFile` option for the extension picker. The user retained final visual acceptance, so this Work remains `Implementing` and unsealed.

## Security and data lifecycle

Tool names and descriptions are model-visible private metadata for this explicit user action. They are held only for the request and are not logged by this operation. MCP transport configuration, filesystem roots, environment, headers, authorization material, secure-store references, and raw provider responses remain outside renderer results, logs, and generated prompts. Only the user-approved existing save flow persists the returned description.

## Compatibility and migration

The new local RPC is additive during development. Existing MCP rows and blank descriptions remain valid, and no schema migration or rollback data transformation is required.
