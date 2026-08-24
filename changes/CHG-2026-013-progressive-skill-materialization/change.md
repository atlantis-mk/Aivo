# Materialize selected Skills progressively

## Problem or goal

Host pre-call resolution currently gives the auxiliary model sanitized Skill summaries, but after an exact Skill selection the Host immediately loads the complete Skill instructions. This skips the MCP-like progressive exposure boundary discussed with the user: selected Skills should first have their canonical summary added by the Host, while full instructions should be loaded only when the task requires the primary model to follow that Skill. The auxiliary model must select IDs and disclosure level, not generate summary text.

## Expected behavior

- `REQ-EXTENSION-001`: the auxiliary resolver returns exact selected resource keys plus an exact subset of selected Skill keys whose instructions are required.
- Every automatically selected Skill contributes a Host-owned canonical summary to the current primary request. Only Skills in the validated instruction subset additionally contribute their complete instructions and bounded resource listing.
- Skill inventory requests continue to receive bounded canonical summaries without loading Skill bodies. Extension context and selected tool schemas retain their existing behavior.
- `REQ-SESSION-001`: automatic summary/instruction materialization is request-scoped and does not mutate explicit session activation.
- `NFR-RELIABILITY-001`: malformed, invented, duplicate, cancelled, or unavailable selections fall back safely and cannot suppress the four core tools.

## Non-goals

- Exposing a model-visible Skill search/load tool.
- Allowing the auxiliary model to write, rewrite, or summarize Skill metadata.
- Changing Skill import, trust, enablement, persistence, supporting-resource loading, MCP resource reads, or extension context semantics.

## Impact

- Go application orchestration and tests: extend the private auxiliary resolver shape, exact validation, and Skill context rendering.
- Providers: the private resolver prompt changes; the primary provider/tool wire contract does not.
- Renderer, Electron, persistence/schema/data, public API/RPC/IPC, credentials, executable lifecycle, Web UI, platform scope, dependencies, packaging, and release formats: none.

## Implementation constraints

- Canonical Skill name, description, scope, and source come only from the current Host-owned imported/enabled record after exact selection validation.
- An instruction request must be a subset of selected exact Skill keys. Invalid keys and non-Skill resources are ignored.
- Summary-only materialization must not read or expose the Skill body or supporting resources.
- Full instructions retain existing size bounds, cancellation, deterministic ordering, and failure isolation. A failed Skill read must not hide healthy context or core tools.
- Existing explicitly active Skills retain their current session semantics and are not duplicated by automatic materialization.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `SKILL-PROTOCOL-001` | `REQ-EXTENSION-001` | Auxiliary response selects exact resources and an exact Skill-instruction subset | `AT-EXTENSION-001` | Complete |
| `SKILL-SUMMARY-001` | `REQ-EXTENSION-001` | Selected Skill canonical summary is injected before optional full instructions | `AT-EXTENSION-001` | Complete |
| `SKILL-SCOPE-001` | `REQ-SESSION-001` | Automatic summary/instructions remain request-scoped | `AT-SESSION-001` | Complete |
| `SKILL-FAIL-001` | `NFR-RELIABILITY-001` | Invalid modes, missing bodies, cancellation, bounds, and fallback preserve a usable request | `CT-RELIABILITY-001` | Complete |

## Acceptance and evidence

- A summary-only auxiliary selection produces a canonical `<skill_summary>` and does not expose a distinctive private instruction body.
- An execution selection produces the same canonical summary plus `<skill_content>` and its bounded supporting-resource listing.
- Invented instruction keys, context keys presented as Skill instructions, and instruction keys absent from selected resources are ignored.
- Inventory, repeated requests, cross-conversation isolation, local resolver fallback, extension context, MCP/plugin tools, four-core Tool Snapshot, failure, cancellation, and bounds remain covered.
- Applicable gates: focused Go tests, `pnpm test:core`, `pnpm docs:check`, `pnpm scripts:test`, `pnpm lint`, `pnpm build`, and `git diff --check`.
- UI screenshots, package/signing, target-OS installer acceptance, migration, and rollback execution are N/A because no renderer, package, platform, persistence, or irreversible format changes.
- Provider-boundary regression `TestAuxiliarySummaryOnlySkillSelectionInjectsCanonicalSummary` proves the auxiliary request receives only the canonical description and the primary request receives `<skill_summary>` without the distinctive private Skill body when `skillInstructions` is absent. `TestHostUsesOneAuxiliaryResolutionForToolSkillMCPAndExtensionContextCandidates` proves an execution selection injects the same summary plus full instructions while retaining selected Manifest context and MCP/extension tool schemas.
- Exact-validation regressions prove duplicate, invented, non-Skill, and non-selected instruction keys cannot trigger full Skill loading. Existing inventory, request-scope, cross-conversation, local fallback, cancellation, failed-source, bounds, and four-core-tool tests remain passing.
- Verification on 2026-08-03, Darwin 23.6.0 x86_64: focused progressive Skill and prompt tests passed; `pnpm test:core` passed all Core packages; `pnpm docs:check` and `pnpm scripts:test` passed; `pnpm lint` passed with only pre-existing Fast Refresh warnings; `pnpm build` passed with only existing large barrel/chunk advisories; `git diff --check` passed.

## Security and data lifecycle

Skill summaries and bodies remain private local data sent only to the configured provider after exact imported/enabled selection. The auxiliary model sees existing bounded sanitized catalog metadata and returns identifiers/disclosure intent only; its prose cannot become injected summary content. No new persistence, secrets, logs, clipboard data, crash data, backups, or temporary artifacts are introduced.

## Compatibility and migration

No public API/RPC/IPC, schema, settings, stored activation, or extension manifest migration. The private auxiliary JSON parser will accept the new instruction subset while retaining safe summary-only behavior when the optional field is absent. Rollback restores eager full-instruction materialization and does not mutate user data.
