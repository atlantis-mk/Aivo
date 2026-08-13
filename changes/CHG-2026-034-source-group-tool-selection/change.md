# Select and inject MCP and extension tools by source group

## Problem or goal

Automatic selection currently exposes every concrete MCP and extension tool to the auxiliary model and expects concrete tool names back. That makes a source with many related tools expensive and lets selection operate at a different level from the source-level capability users name. A new conversation that asks to use one MCP or extension should let the auxiliary model choose that source once, while the Host expands the selected source to its complete current eligible concrete tool set.

## Expected behavior

For `REQ-SESSION-001`, `REQ-TOOL-001`, and `REQ-EXTENSION-001`, every enabled ready MCP or extension with executable tools contributes one stable Host-owned group name and one bounded optional capability description to auxiliary selection. The auxiliary request renders only the user intent and one `name：description` line per group, leaving the value blank when no description exists, and accepts only a strict JSON array of exact group names. The Host validates selected groups, expands each to all currently registered mode-eligible and globally visible concrete tools, persists the exact expanded automatic set, and freezes those identities in the Tool Snapshot. MCP creation stores an omitted description as empty. Extensions use their available validated Manifest description without synthesized fallback text.

## Non-goals

This Work does not let a model install, trust, enable, authenticate, authorize, or execute a source; does not change manual source mention expansion; does not remove per-tool global visibility or call-time permission checks; and does not make an existing conversation absorb tools added to a source after its automatic set was frozen.

## Impact

Core auxiliary selection, Registry catalog metadata, MCP validation, extension registration metadata, Provider declarations, session automatic state, tests, and the desktop MCP form change. The existing MCP description column and RPC shape are reused, so persistence schema and migration are unchanged. Electron privilege boundaries, secrets, child processes, release packaging, dependencies, and renderer/core IPC versions do not change.

## Implementation constraints

The Host owns group identity, membership, bounds, eligibility, and expansion. Descriptions are bounded single-line untrusted data and cannot inject selector instructions. Unknown, duplicate, malformed, unavailable, empty, or oversized selections fail closed. Initial selection may fall back to bounded local group matching; a failed Agent-requested replacement preserves the prior automatic set. Provider-specific namespace grouping may contain only the expanded selected members. Concrete registration and schema identities remain the execution boundary.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `TASK-034-01` | REQ-EXTENSION-001 | Optional bounded MCP functional description and extension group metadata | AT-EXTENSION-001 | Completed |
| `TASK-034-02` | REQ-TOOL-001 | Minimal auxiliary group prompt and strict JSON name-array parser | AT-TOOL-001, CT-SECURITY-001 | Completed |
| `TASK-034-03` | REQ-SESSION-001 | Selected groups expand to an exact stable session automatic set and snapshot | AT-SESSION-001, AT-EXTENSION-001 | Completed |
| `TASK-034-04` | NFR-SECURITY-001 | Invalid descriptions/responses and ineligible members fail closed | CT-SECURITY-001 | Completed |

## Acceptance and evidence

- Pre-fix reproduction: multiple concrete tools from one MCP/extension appear separately in the auxiliary catalog and the response contract includes tools/resources/reason fields rather than a strict group-name array.
- Required: focused Go and desktop tests, `pnpm docs:check`, `pnpm test:core`, `pnpm lint`, and `pnpm build`.
- UI acceptance is limited to the existing responsive MCP add/edit form gaining a functional-description field; no new layout surface is introduced.
- Failure, repetition, source loss, unknown output, hidden members, Provider namespace serialization, and snapshot stability require automated coverage. Cancellation and timeout continue through the existing auxiliary/provider context path. Migration/rollback is N/A because no schema changes.

Implementation evidence on 2026-08-11: focused group-selection, optional/blank MCP description, Host pre-call, and replaceable-session tests pass; `pnpm test:core`, `pnpm docs:check`, `pnpm lint`, and `pnpm build` pass. The strict parser rejects object wrappers, Markdown, duplicates, unknown names, and trailing content; a blank description is retained as `name：`; eligibility tests exclude hidden members; complete-group coverage verifies expansion beyond the legacy eight-tool limit up to the Host bound. The user retained final visual acceptance of the MCP form, so this Work remains `Implementing` and unsealed until that acceptance is reported.

## Security and data lifecycle

Capability descriptions, when present, are non-secret persisted configuration/Manifest metadata. They are trimmed, single-line bounded before model context, and explicitly treated as untrusted candidate data; absent descriptions remain blank. Raw tool schemas, credentials, endpoints, and source configuration do not enter the auxiliary prompt. Group selection grants no authority; the Host revalidates every expanded registration and permissions remain call-scoped.

## Compatibility and migration

No database transition is required. Existing and new MCP sources with blank descriptions stay stored, remain manageable, and enter automatic group selection with an empty description value. Existing conversation concrete automatic sets remain readable and stable. The auxiliary response format changes incompatibly during development from an object to a bare JSON string array.
