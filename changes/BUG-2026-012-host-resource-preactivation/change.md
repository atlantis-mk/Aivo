# Restore Host pre-call activation for skills and extensions

## Problem or goal

After `CHG-2026-007-minimal-agent-tool-primitives` removed model-visible discovery tools, the primary Agent still receives stale instructions to call `skill` and `tool_resolve`, although neither tool is registered. The Host pre-call path resolves only tools already present in a workspace Registry. It never selects or injects Skill instructions, omits enabled legacy-plugin registrations from Agent registries, does not recover a missing MCP capability catalog before selection, and never supplies enabled extension context resources. As a result, the Agent can incorrectly report that no Skill/MCP/extension capability exists while only listing `read`, `bash`, `edit`, and `write`.

Minimum reproduction on `0.0.0-development` (observed on macOS): import and enable a UI-component Skill, then ask “当前有哪些 UI 组件技能”. The primary model sees the four primitives plus instructions naming unavailable discovery tools and replies that it cannot query Skills. Equivalent reproductions use an enabled plugin whose declared tool is absent from the workspace Registry, an enabled MCP server with no cached capability rows, or an enabled Manifest v1 static/context extension whose resource is never added to model context.

Expected: before every primary request, the Host builds one bounded eligible resource catalog, resolves relevant Skills/tools/context resources, loads only validated selections, then freezes the resulting Tool Snapshot and context. Actual: only already-registered tools participate, while Skill and context resolution remain disconnected and stale prompt instructions delegate the missing Host responsibility back to the primary model.

## Expected behavior

- `REQ-EXTENSION-001`: one Host-owned pre-call phase prepares eligible enabled Skill, plugin, Manifest v1 extension, and MCP contributions before the primary model request.
- Skill inventory questions receive a bounded catalog summary; ordinary matching requests receive the selected imported-and-enabled Skill instructions for the current request. Pending or disabled Skills are not activated implicitly.
- Enabled plugin and MCP catalogs are made available to the resolver without advertising every long-tail schema to the primary model. A missing MCP catalog may be refreshed with a bounded, cancellable probe; failure falls back to the remaining core/resources and records safe diagnostics.
- Manifest v1 context resources are selected and injected with explicit size bounds. Tool selections are validated against exact eligible registrations before their schemas enter the immutable Tool Snapshot.
- The primary prompt describes Host preactivation and never instructs the model to call removed discovery bridge tools.
- `REQ-SESSION-001`: automatically selected Skill/context resources apply to the current request only; explicit session-pinned Skills/tools retain their existing session semantics and do not leak across conversations.
- `NFR-RELIABILITY-001`: catalog preparation and resource loading honor cancellation, bounds, deterministic ordering, and partial-failure fallback.

## Non-goals

- Installing, importing, trusting, enabling, authorizing, or binding credentials through the resolver.
- Automatically importing pending Skill candidates or enabling disabled/error extensions and MCP servers.
- Automatically reading arbitrary MCP resources or invoking MCP prompts; MCP resource utility tools remain explicit selected capabilities.
- Changing Manifest v1, extension trust policy, permission policy, persistence schema, or Web-view isolation.
- Reintroducing `skill`, `tool_resolve`, or any other model-visible discovery bridge.

## Impact

- Go application orchestration: add a bounded Host pre-call resource result and inject its ephemeral context before provider invocation.
- Skills: reuse existing sanitized candidate selection and Skill rendering without persisting automatic selections as session activation.
- Plugins/MCP: restore enabled catalog preparation before Agent Registry assembly; only resolver-selected schemas reach the provider. MCP refresh can connect to an already enabled server but cannot enable or authorize it.
- Manifest v1 extensions: expose eligible context metadata for selection and bounded content loading; executable runtimes remain governed by existing trusted/enabled/ready lifecycle.
- Provider/Tool Snapshot: no wire-format or persistence change; the snapshot is still frozen after preparation.
- Renderer/Electron, HTTP/RPC/IPC, schema/data, credentials, Web UI, platform scope, dependencies, and release packaging: none.

## Implementation constraints

- The Host owns the phase and completes it before the primary provider call. The primary model cannot request discovery as a fallback.
- Resolver inputs contain only bounded, sanitized names/descriptions/source metadata. Exact candidate validation precedes Skill content reads, context reads, or tool-schema exposure.
- Automatic Skill/context activation is ephemeral. Existing explicit session activation remains the only durable activation state.
- Only trusted, enabled Manifest v1 extensions and enabled plugins/MCP servers are eligible. Selection grants no additional authority; normal permission checks still run at execution.
- MCP catalog refresh uses the request context and existing connection timeouts. One failing source must not hide core tools or contributions from other healthy sources.
- Context composition is deterministically ordered and bounded before insertion into model messages. Secret-bearing values and raw diagnostics are excluded.
- Existing `ADR-0002` already owns the boundary; no new ADR or schema migration is required.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `HOST-RESOLVE-001` | `REQ-EXTENSION-001` | Unified pre-call preparation includes enabled plugin, Manifest v1, and MCP catalogs before bounded tool selection | `AT-EXTENSION-001` | Complete |
| `HOST-SKILL-001` | `REQ-SESSION-001`, `REQ-EXTENSION-001` | Inventory summary and ephemeral selected Skill instructions are injected without a model discovery tool or cross-session mutation | `AT-SESSION-001`, `AT-EXTENSION-001` | Complete |
| `HOST-CONTEXT-001` | `REQ-EXTENSION-001` | Relevant enabled extension context resources are selected, validated, bounded, and injected | `AT-EXTENSION-001` | Complete |
| `HOST-PROMPT-001` | `REQ-EXTENSION-001` | Primary prompt no longer references removed discovery tools | `AT-EXTENSION-001` | Complete |
| `HOST-FAIL-001` | `NFR-RELIABILITY-001` | Cancellation, failed catalog refresh, invalid selections, ordering, and output bounds preserve a usable core request | `CT-RELIABILITY-001` | Complete |

## Acceptance and evidence

- A pre-fix regression test demonstrates that an enabled relevant Skill is absent from the first primary request and the prompt names unavailable tools; after the fix, the selected Skill content or bounded inventory is present without advertising a discovery tool.
- Plugin, Manifest v1 process/service/static contribution, and MCP fixtures demonstrate that eligible catalogs participate before selection while unrelated long-tail schemas remain absent.
- Disabled, untrusted, pending-import, invalid-selection, failing, timed-out, and cancelled sources do not become active and do not prevent the four core tools from being assembled.
- Repeated requests are deterministic; ephemeral automatic resources do not appear in another conversation. Explicit pinned/warm tool behavior remains covered.
- Applicable verification: focused Go tests, `pnpm docs:check`, `pnpm test:core`, `pnpm lint`, `pnpm build`, and `git diff --check`. UI screenshots, packaging, signing, migration, and target-OS installer acceptance are N/A because this bug changes no UI, package, platform contract, or persistence format.
- Pre-fix/post-fix regression: `TestFirstPrimaryRequestReceivesHostResolvedSkillInventoryAndFourCoreTools` would fail before the fix because no `<host_preactivated_resources>` inventory existed and the prompt referenced removed tools; it now proves the first primary request receives the bounded imported/enabled inventory with exactly the four core schemas. `TestHostUsesOneAuxiliaryResolutionForToolSkillMCPAndExtensionContextCandidates` now proves one auxiliary Host call selects Manifest and MCP tools plus a Skill and extension context before the primary request.
- Boundary coverage: focused tests pass for exact/invalid/duplicate resource selection, request-scoped repetition and cross-conversation isolation, stopped Manifest contexts, cached resource-only MCP adapters, failed plugin/MCP catalog preparation, cancellation fallback, context ordering/bounds, pinned/default/auto tool policies, and removal of stale prompt instructions.
- Verification on 2026-08-03, macOS 14.8.7 x86_64: focused Host resolver tests passed; `pnpm test:core` passed all Core packages; `pnpm docs:check` and `pnpm scripts:test` passed; `pnpm lint` passed with only pre-existing Fast Refresh warnings; `pnpm build` passed with only existing large-module/chunk advisories; `git diff --check` passed.
- UI screenshots, packaging, signing, migration, rollback execution, and target-OS installer evidence are N/A: no renderer, package, platform, persistence, or irreversible format changed. Rollback is the code reversal described below.

## Security and data lifecycle

Skill bodies and extension context are private local data and are sent to the configured provider only after exact eligible selection, with existing context bounds. Catalog summaries exclude secrets, raw tool payloads, credentials, and diagnostics. MCP/plugin/extension enablement and trust remain user-owned state; resolver output cannot mutate them. No new persisted data, logs, clipboard data, crash data, backups, or temporary artifacts are introduced.

## Compatibility and migration

No API/RPC/IPC, schema, settings, or stored activation migration. The fix restores the accepted `ADR-0002` behavior in the development contract. Rollback returns to four core tools with incomplete long-tail preactivation; it does not mutate user files or extension configuration.

## Bug root cause

The minimal-tool refactor implemented tool selection only after Registry construction, but did not implement the corresponding Host resource preparation for Skills/context or restore all enabled extension catalogs to that Registry. Legacy prompt text and separate auxiliary tool/Skill resolver helpers remained, and tests asserted individual catalogs/lifecycles rather than the first complete primary-request boundary. The fix replaces those duplicate auxiliary paths with one bounded Host resolver decision over exact tool and instruction/context candidates, restores eligible plugin/MCP catalog preparation, and freezes schemas only after validation. The affected version is `0.0.0-development`; the fix version is the next development build. The named first-request and unified-resolution regressions failed by construction against the pre-fix behavior and pass after the fix.
