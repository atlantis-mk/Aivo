# Select and inject explicitly grouped or individual tools

## Problem or goal

Automatic selection must avoid exposing Host-owned group membership to the auxiliary model, but grouping every MCP/extension implicitly by source also prevents registrants from declaring independent tools and makes desktop surfaces infer different groups from namespaces. Registration must explicitly choose between one individual selection resource and membership in one declared group.

## Expected behavior

For `REQ-SESSION-001`, `REQ-TOOL-001`, and `REQ-EXTENSION-001`, every eligible registration contributes either one individual tool candidate or one declared group candidate. The auxiliary request renders only the user intent and one bounded `kind:id：display-name｜description` line per candidate, and accepts only a strict classified object containing exact typed resource IDs. Grouped candidates never expose or accept model-authored member names. The Host validates the selected resources, expands a group to all currently registered mode-eligible and globally visible members and an individual to itself, persists the concrete automatic set, and freezes those identities in the Tool Snapshot. MCP adapters explicitly declare one group per server; Manifest tools are individual unless `toolGroups` declares membership.

## Non-goals

This Work does not let a model or renderer author group membership, install, trust, enable, authenticate, authorize, or execute a source; does not remove per-tool global visibility or call-time permission checks; and does not make an existing conversation absorb tools added to a group after its concrete automatic set was frozen.

## Impact

Core auxiliary selection, Registry and Manifest catalog metadata, MCP registration, Provider declarations, session automatic state, tests, the desktop four-category resource manager, composer chooser, conversation activation, and visible pre-call disclosure change. Persistence schema is unchanged. Electron privilege boundaries, secrets, child processes, release packaging, and dependencies do not change.

## Implementation constraints

The registration contract and Host own group identity, membership, bounds, eligibility, and expansion. Each tool belongs to at most one group; missing group metadata means individual registration. Descriptions and display names are bounded single-line untrusted data and cannot inject selector instructions. Unknown, duplicate, malformed, unavailable, empty, or oversized selections fail closed. Initial selection may fall back to bounded local resource matching; a failed Agent-requested replacement preserves the prior automatic set. Concrete registration and schema identities remain the execution boundary.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `TASK-034-01` | REQ-EXTENSION-001 | Exact typed source metadata plus complete MCP tool-description fallback | AT-EXTENSION-001 | Completed |
| `TASK-034-02` | REQ-TOOL-001 | Minimal typed source-ID prompt and strict classified source parser | AT-TOOL-001, CT-SECURITY-001 | Completed |
| `TASK-034-03` | REQ-SESSION-001 | Selected groups expand to an exact stable session automatic set and snapshot | AT-SESSION-001, AT-EXTENSION-001 | Completed |
| `TASK-034-04` | NFR-SECURITY-001 | Invalid descriptions/responses and ineligible members fail closed | CT-SECURITY-001 | Completed |
| `TASK-034-05` | REQ-TOOL-001, REQ-EXTENSION-001 | Explicit Registry/Manifest group metadata and grouped-or-individual auxiliary candidates | AT-TOOL-001, AT-EXTENSION-001 | Completed |
| `TASK-034-06` | REQ-SESSION-001 | Shared desktop grouping, member disclosure, group-only switch, and group composer activation | AT-SESSION-001 | Completed |

## Acceptance and evidence

- Pre-fix reproduction: multiple concrete tools from one MCP/extension appear separately in the auxiliary catalog or visible pre-call disclosure, and generated namespace names become a second resource identity instead of exact source IDs.
- Required: focused Go and desktop tests, `pnpm docs:check`, `pnpm test:core`, `pnpm lint`, and `pnpm build`.
- UI acceptance covers wide and narrow tool-management and conversation-activation lists: grouped rows disclose every member and expose one switch, while individual rows remain separate with their own switch.
- Failure, repetition, source loss, unknown output, hidden members, Provider namespace serialization, and snapshot stability require automated coverage. Cancellation and timeout continue through the existing auxiliary/provider context path. Migration/rollback is N/A because no schema changes.

Implementation evidence on 2026-08-11: focused group-selection, optional/blank MCP description, Host pre-call, and replaceable-session tests pass; `pnpm test:core`, `pnpm docs:check`, `pnpm lint`, and `pnpm build` pass. On 2026-08-21 the source contract was unified: the strict parser accepts only typed MCP/extension IDs, rejects concrete names, duplicates, unknown identities, Markdown, extra fields, and trailing content; MCP adapters now retain the exact server ID as registration source identity; a blank MCP source description is assembled from every current eligible member description; complete-group coverage verifies unsplit expansion beyond the former member limit. Standalone built-in tools remain available without entering source selection and are disclosed under the `tool` category. The final `pnpm scripts:test`, `pnpm test:core`, `pnpm docs:check`, `pnpm lint`, `pnpm build`, and `git diff --check` gates passed; lint/build retained only existing Fast Refresh, large-barrel, and chunk advisories. The user retained final visual acceptance, so this Work remains `Implementing` and unsealed until that acceptance is reported.

Implementation evidence on 2026-08-26: Registry and Tool Catalog gained optional explicit selection-group metadata; Manifest v2 gained validated optional `toolGroups`; MCP adapters explicitly register one server group; `aivo.projects` is grouped and `aivo.tools` remains individual. The auxiliary contract now receives one display-name line per group or individual and accepts strict typed `resources`, with grouped members withheld from the prompt and expanded only by the Host. Shared desktop grouping renders member disclosure and one group switch without inferring groups from source or namespace. Focused Core and desktop tests, scripts tests, full Core tests, docs checks, lint, build, and diff checks pass. The PTY replay test had one unrelated empty-replay timing failure during the first final Core run, passed three focused repetitions, and the complete Core gate then passed; visual acceptance remains before sealing.

## Security and data lifecycle

Capability descriptions are non-secret persisted configuration/Manifest metadata or, for a blank MCP group description, a bounded deterministic assembly of every current eligible member name and description. They are single-line and explicitly treated as untrusted candidate data. Grouped member names, raw schemas, credentials, endpoints, and source configuration do not enter the auxiliary prompt. Group or individual selection grants no authority; the Host revalidates every expanded registration and permissions remain call-scoped.

## Compatibility and migration

No database transition is required. Existing and new MCP sources with blank descriptions stay stored and editable; selection derives a non-persisted fallback from current tool metadata. Existing conversation concrete automatic sets remain readable and stable. Missing selection-group metadata means individual registration, while the MCP adapter explicitly preserves server-level grouping. The auxiliary response format changes incompatibly during development from the strict typed-source object to the strict classified typed-resource object.
