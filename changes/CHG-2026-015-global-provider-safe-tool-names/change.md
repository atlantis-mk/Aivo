# Make every canonical tool name Provider-safe

## Problem or goal

`CHG-2026-010` preserved dotted canonical tool names and added a per-request Provider wire codec. That leaves two identities for one tool and lets invalid names enter registration, snapshots, activation, history, and UI before being rewritten at the Provider boundary. Some supported model Providers accept only ASCII letters, digits, `_`, and `-` in function names. The approved replacement is one global naming contract: every executable tool identity is Provider-safe when registered, and Provider adapters send and receive that exact name without encoding, decoding, escaping, or alias lookup.

## Expected behavior

- `REQ-PROVIDER-001`: every non-hosted tool has one canonical name matching `^[A-Za-z0-9_-]+$` with the common 64-byte limit; Provider adapters serialize that exact name and observe the same name in responses and streams.
- `REQ-EXTENSION-001`: Manifest v1 rejects contributed tool names outside the global contract. Built-in and MCP adapter tool names use `_`-separated canonical identities; an MCP server's original tool name remains separate and is used only for the upstream MCP call.
- Registry registration is the authoritative enforcement boundary, so invalid or oversized names cannot enter catalogs, activation state, Tool Snapshots, permission evaluation, execution, UI emission, or new persisted history.
- `NFR-RELIABILITY-001`: collisions are rejected rather than hidden behind request-local aliases, and repeated/multi-turn Provider requests retain one stable identity.

## Non-goals

- Do not rename extension IDs, source IDs, capabilities, display labels, MCP upstream method/tool names, or hosted Provider tools.
- Do not add compatibility aliases, Provider-specific escaping, lossy runtime rewriting, or inference for unknown returned names.
- Do not change extension trust, activation, permission, credential, cancellation, execution, or persistence schema behavior.

## Impact

- Go domain/application: central tool-identifier validation, safe built-in/MCP/Manifest names, and removal of the Provider codec.
- Providers: declarations, assistant history, final responses, and streaming deltas use the canonical name unchanged.
- Extensions: Manifest v1's contributed-tool naming contract becomes ASCII Provider-safe and bounded; MCP adapters retain upstream raw names separately.
- Renderer/Electron, HTTP/RPC/IPC, persistence schema, credentials, processes, worktrees, dependencies, packaging, and platform scope are unchanged.

## Implementation constraints

- The registry owns the global executable-name invariant and must validate both single and batch registration before mutation.
- Provider code must not maintain a canonical-to-wire or wire-to-canonical table.
- Generated MCP names must be deterministic and use only the global character set. Registration collision errors remain authoritative and must not silently select another identity.
- Historical dotted tool calls remain generic display evidence and gain no executor alias. Development data that must continue executing is recreated with the new canonical names; no persistence migration is introduced.
- Hosted Provider tools keep their native Provider-owned serialization.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `TOOL-NAME-001` | `REQ-EXTENSION-001` | Enforce one bounded Provider-safe canonical name at registry and Manifest boundaries | `AT-EXTENSION-001` | Complete |
| `TOOL-NAME-002` | `REQ-EXTENSION-001` | Rename built-in and generated MCP tool identities while retaining upstream MCP names | `AT-EXTENSION-001` | Complete |
| `TOOL-NAME-003` | `REQ-PROVIDER-001` | Remove Provider request/response name codecs and serialize canonical names unchanged | `AT-PROVIDER-001` | Complete |
| `TOOL-NAME-004` | `NFR-RELIABILITY-001` | Cover invalid names, length boundaries, collisions, history, responses, streams, and repeated requests | `CT-RELIABILITY-001` | Complete |

## Acceptance and evidence

- A pre-change regression proves the registry accepts a dotted name that violates the new invariant.
- Focused tests prove the allowed character set and 64-byte boundary, invalid Manifest refusal, pre-mutation batch refusal, deterministic MCP naming, and collision rejection.
- Provider request-body tests prove declarations and historical calls are unchanged canonical names; final and streamed calls reach Host consumers unchanged; no codec or alias map remains.
- Unknown/invalid Provider-returned names continue to fail through the authoritative Tool Snapshot/registry path and are not guessed.
- Repetition and multi-turn behavior are covered. Cancellation, timeout, teardown, dependency loss, security, persistence migration/rollback, UI screenshots, installer/signing, and package smoke are unchanged or N/A for this contract reset.
- Applicable gates: focused Go tests, `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, `pnpm build`, and `git diff --check`.
- Pre-change evidence on 2026-08-04: `go test ./app -run '^TestRegistryRejectsProviderUnsafeToolNames$' -count=1` failed because the Registry accepted dotted, space-containing, Unicode, and 65-byte names.
- Post-change focused evidence passed for Registry character/length enforcement, atomic batch refusal, Manifest v1 refusal, deterministic bounded MCP canonical naming, collision refusal, separate upstream MCP identity, unchanged Provider declarations/history/final responses/stream deltas, and two repeated requests.
- `pnpm test:core` passed on 2026-08-04 for Core application, CLI, persistence, and HTTP transport packages.
- Repository gates passed on 2026-08-04: `pnpm docs:check`, `pnpm scripts:test`, `pnpm lint`, `pnpm build`, and `git diff --check`. Lint reported only existing Fast Refresh warnings; build reported only existing large-barrel and chunk-size advisories.
- Verification platform: macOS Darwin 23.6.0 x86_64 with Go 1.26.0 darwin/amd64. Provider smoke requiring configured external credentials, UI screenshots, installer/signing, and package smoke are N/A because this change adds no provider configuration, visual behavior, or packaging surface.

## Security and data lifecycle

The change handles identifiers only. It adds no secrets, payload logging, persistence table, credentials, artifacts, network calls, or new authority. One canonical identity continues to drive snapshots, permissions, execution, and recorded results.

## Compatibility and migration

This supersedes the development-only dotted-name/wire-alias contract in `CHG-2026-010`. Built-in and Manifest/MCP canonical names change incompatibly before release. Historical rows remain renderable but old dotted names are not executable and are not rewritten at Provider boundaries. Rollback restores the old canonical names and codec; user filesystem data is unaffected.

## Bug root cause (type=bug only)

N/A; this is an approved contract replacement.
