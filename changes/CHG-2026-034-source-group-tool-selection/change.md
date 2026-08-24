# Select and inject MCP and extension tools by source group

## Problem or goal

Automatic selection currently exposes every concrete MCP and extension tool to the auxiliary model and expects concrete tool names back. That makes a source with many related tools expensive and lets selection operate at a different level from the source-level capability users name. A new conversation that asks to use one MCP or extension should let the auxiliary model choose that source once, while the Host expands the selected source to its complete current eligible concrete tool set.

## Expected behavior

For `REQ-SESSION-001`, `REQ-TOOL-001`, and `REQ-EXTENSION-001`, every enabled ready MCP or non-built-in extension with executable tools contributes one typed exact source ID and one bounded capability description to auxiliary selection. The auxiliary request renders only the user intent and one `kind:id：description` line per source and accepts only a strict classified JSON object containing exact MCP/extension IDs. It never accepts a concrete tool name. The Host validates selected sources, expands each to all currently registered mode-eligible and globally visible concrete tools, persists the exact expanded automatic set, and freezes those identities in the Tool Snapshot. MCP creation still stores an omitted description as empty; when selecting, the Host assembles every current eligible tool name and description as its fallback. Extensions use their validated Manifest description without synthesized fallback text.

## Non-goals

This Work does not let a model install, trust, enable, authenticate, authorize, or execute a source; does not change manual source mention expansion; does not remove per-tool global visibility or call-time permission checks; and does not make an existing conversation absorb tools added to a source after its automatic set was frozen.

## Impact

Core auxiliary selection, Registry catalog metadata, MCP validation, extension registration metadata, Provider declarations, session automatic state, tests, the desktop four-category resource manager, and the visible pre-call disclosure change. The existing MCP description column and RPC shape are reused, so persistence schema and migration are unchanged. Electron privilege boundaries, secrets, child processes, release packaging, dependencies, and renderer/core IPC versions do not change.

## Implementation constraints

The Host owns group identity, membership, bounds, eligibility, and expansion. Descriptions are bounded single-line untrusted data and cannot inject selector instructions. Unknown, duplicate, malformed, unavailable, empty, or oversized selections fail closed. Initial selection may fall back to bounded local group matching; a failed Agent-requested replacement preserves the prior automatic set. Provider-specific namespace grouping may contain only the expanded selected members. Concrete registration and schema identities remain the execution boundary.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `TASK-034-01` | REQ-EXTENSION-001 | Exact typed source metadata plus complete MCP tool-description fallback | AT-EXTENSION-001 | Completed |
| `TASK-034-02` | REQ-TOOL-001 | Minimal typed source-ID prompt and strict classified source parser | AT-TOOL-001, CT-SECURITY-001 | Completed |
| `TASK-034-03` | REQ-SESSION-001 | Selected groups expand to an exact stable session automatic set and snapshot | AT-SESSION-001, AT-EXTENSION-001 | Completed |
| `TASK-034-04` | NFR-SECURITY-001 | Invalid descriptions/responses and ineligible members fail closed | CT-SECURITY-001 | Completed |

## Acceptance and evidence

- Pre-fix reproduction: multiple concrete tools from one MCP/extension appear separately in the auxiliary catalog or visible pre-call disclosure, and generated namespace names become a second resource identity instead of exact source IDs.
- Required: focused Go and desktop tests, `pnpm docs:check`, `pnpm test:core`, `pnpm lint`, and `pnpm build`.
- UI acceptance is limited to the existing responsive MCP add/edit form gaining a functional-description field; no new layout surface is introduced.
- Failure, repetition, source loss, unknown output, hidden members, Provider namespace serialization, and snapshot stability require automated coverage. Cancellation and timeout continue through the existing auxiliary/provider context path. Migration/rollback is N/A because no schema changes.

Implementation evidence on 2026-08-11: focused group-selection, optional/blank MCP description, Host pre-call, and replaceable-session tests pass; `pnpm test:core`, `pnpm docs:check`, `pnpm lint`, and `pnpm build` pass. On 2026-08-21 the source contract was unified: the strict parser accepts only typed MCP/extension IDs, rejects concrete names, duplicates, unknown identities, Markdown, extra fields, and trailing content; MCP adapters now retain the exact server ID as registration source identity; a blank MCP source description is assembled from every current eligible member description; complete-group coverage verifies unsplit expansion beyond the former member limit. Standalone built-in tools remain available without entering source selection and are disclosed under the `tool` category. The final `pnpm scripts:test`, `pnpm test:core`, `pnpm docs:check`, `pnpm lint`, `pnpm build`, and `git diff --check` gates passed; lint/build retained only existing Fast Refresh, large-barrel, and chunk advisories. The user retained final visual acceptance, so this Work remains `Implementing` and unsealed until that acceptance is reported.

## Security and data lifecycle

Capability descriptions are non-secret persisted configuration/Manifest metadata or, for a blank MCP description, a bounded deterministic assembly of every current eligible tool name and description. They are single-line and explicitly treated as untrusted candidate data. Raw schemas, credentials, endpoints, and source configuration do not enter the auxiliary prompt. Source selection grants no authority; the Host revalidates every expanded registration and permissions remain call-scoped.

## Compatibility and migration

No database transition is required. Existing and new MCP sources with blank descriptions stay stored and editable; selection derives a non-persisted fallback from current tool metadata. Existing conversation concrete automatic sets remain readable and stable. The auxiliary response format changes incompatibly during development to the strict classified typed-source object.
