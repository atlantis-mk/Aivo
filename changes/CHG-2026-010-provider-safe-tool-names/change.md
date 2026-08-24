# Encode canonical extension tool names for provider compatibility

## Problem or goal

On macOS acceptance, enabling the Chrome extension and asking which tabs are open causes the model request to fail before execution with HTTP 400: `tools[4].function.name` does not match the Provider pattern `^[a-zA-Z0-9_-]+$`. Aivo correctly keeps canonical namespaced extension identities such as `mcp.chrome.list_tabs`, but sends those names unchanged through Provider function declarations. Expected behavior is that the request succeeds and the returned call executes against the exact frozen canonical registration; actual behavior is that a Provider-invalid dot or other punctuation rejects the complete turn.

## Expected behavior

- `REQ-PROVIDER-001`: every local function declaration sent to a Provider uses a deterministic bounded wire name accepted by the common Provider function-name contract.
- `REQ-EXTENSION-001`: canonical extension/MCP registration names and Tool Snapshot identities remain unchanged; outbound declarations and prior assistant tool calls are encoded, while streamed and completed calls are decoded to the canonical name before UI emission, history recording, snapshot lookup, permission evaluation, or execution.
- Safe core names remain unchanged. Distinct canonical names never share one wire alias within a request, including sanitized-name collisions and length boundaries.
- `NFR-RELIABILITY-001`: repeated calls and multi-turn history retain stable aliases without introducing retries, stale mappings, or execution ambiguity.

## Non-goals

- Do not rename extension manifests, MCP registrations, persisted canonical tool history, or Tool Snapshot identities.
- Do not loosen registry collision checks or Provider validation, and do not infer unknown aliases returned by a Provider.
- Do not change extension activation, trust, permissions, credentials, cancellation, or tool execution semantics.

## Impact

- Go Provider adapter orchestration gains a per-request canonical/wire-name codec and focused tests.
- Renderer, Electron, persistence schema, local HTTP/RPC/IPC, provider credentials, MCP transport, extension protocol version, dependencies, packaging, and platform scope are unchanged.
- Existing canonical names in history remain compatible because they are encoded only in the outbound Provider copy and restored before Aivo records new results.

## Implementation constraints

- Canonical `ToolSpec.Name` remains the only registry, snapshot, permission, execution, and persisted-history identity.
- The codec is built from the exact route-specific Tool Snapshot input, keeps already compatible bounded names unchanged, and assigns deterministic collision-free aliases to incompatible names.
- Encoding applies to declarations and assistant tool calls sent back as conversation context. Decoding applies to synchronous responses and streaming deltas before callers observe them.
- Hosted Provider tools remain owned by their existing native serialization and are not remapped.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `BUG-NAME-001` | `REQ-PROVIDER-001` | Add deterministic Provider-safe aliases with pattern, length, and collision handling | `AT-PROVIDER-001` | Complete |
| `BUG-NAME-002` | `REQ-EXTENSION-001` | Encode declarations/history and decode response/stream names to canonical identity | `AT-EXTENSION-001` | Complete |
| `BUG-NAME-003` | `NFR-RELIABILITY-001` | Add pre-fix failing/post-fix passing multi-turn and repeated-mapping regressions | `CT-RELIABILITY-001` | Complete |

## Acceptance and evidence

- A regression fails before the fix because a namespaced Chrome/MCP tool is serialized with a dot and violates the Provider pattern.
- Post-fix tests prove safe core names are unchanged; punctuation, Unicode, length, and sanitized-name collisions produce valid unique aliases; declarations and historical calls use aliases; returned and streamed calls expose canonical names.
- Unknown Provider-returned names remain unchanged and fail through the existing authoritative snapshot/registry path rather than being guessed.
- Provider errors, cancellation, timeouts, fallback, execution teardown, permissions, and extension lifecycle remain unchanged and are covered by the existing suite.
- Persistence migration/rollback, dependency loss, UI screenshots, installer/signing, and package smoke are N/A for this Provider adapter correction.
- Pre-fix evidence on 2026-08-03: `go test ./app -run '^TestProviderToolNameCodec' -count=1` failed to build because `newProviderToolNameCodec` and the Provider limit did not exist. This was recorded before product-code changes.
- Post-fix focused evidence: the codec and end-to-end generation tests passed. They prove the common `^[a-zA-Z0-9_-]+$`/64-byte boundary, safe core-name preservation, punctuation/Unicode/length encoding, distinct deterministic aliases, Provider-safe namespaces and historical calls, final-response and streamed-delta decoding, and unchanged unknown names.
- Focused compatibility evidence passed for generation/fallback, OpenAI-compatible calls, Responses and Chat Completions request bodies, MCP registration utilities, and frozen Tool Snapshots.
- `pnpm test:core` passed on 2026-08-03 for Core app, CLI, persistence, and HTTP transport packages.
- Repository gates passed on 2026-08-03: `pnpm docs:check`, `pnpm scripts:test`, `pnpm lint`, `pnpm build`, and `git diff --check`. Lint reported only existing Fast Refresh warnings; build reported only existing large-barrel and chunk-size advisories.
- Verification platform: macOS 14.8.7, Darwin 23.6.0 x86_64, Go 1.26.0 darwin/amd64.

## Security and data lifecycle

The codec handles tool names only and does not persist an alias table, tool arguments, results, credentials, or private content. Aliases are deterministic hashes plus bounded safe prefixes and are not logged as a new event. Canonical trust, snapshot, permission, and execution checks remain authoritative.

## Compatibility and migration

No schema, settings, API/RPC/IPC, manifest, or protocol migration. Rollback reintroduces Provider rejection for namespaced extension tools but does not alter existing canonical registrations or history.

## Bug root cause (type=bug only)

Affected version: `0.0.0-development`. Extension/MCP names intentionally use dotted canonical namespaces, while Provider serializers copied `ToolSpec.Name` and historical `ChatToolCall.Name` directly into function-name fields. Existing serializer tests covered core underscore names and namespace grouping but never asserted the lowest-common Provider name pattern, long names, collisions, or round-trip restoration. The fix installs one route-local codec before every Provider call, sends safe names in declarations and history, and restores canonical names in streaming deltas and final responses before any Host consumer observes them. The regression failed before the codec existed and passed afterward. Fix version: `0.0.0-development`.
