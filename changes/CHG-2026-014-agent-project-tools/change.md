# Add Agent project catalog and immutable session binding tools

## Problem or goal

Agents cannot currently discover the user's registered Aivo projects or bind an unscoped coding conversation to the correct project after the user identifies it. A retired `search_projects` implementation is no longer registered after the four-primitive boundary was adopted and allowed project switching without the required immutable ownership and permission behavior. Add a bounded, namespaced first-party extension that can query projects, register an existing directory, and bind the current unscoped coding session exactly once.

## Expected behavior

`REQ-PROJECT-003` provides `aivo.projects.query`, `aivo.projects.add`, and `aivo.projects.associate` through a trusted built-in Manifest v1 extension. Querying is read-only. Adding registers only an existing absolute directory. Association targets only the current coding session, is atomic with its coding context, and becomes immutable: the same association is idempotent while switching or detaching is forbidden. Project writes follow the active permission mode.

## Non-goals

No directory creation, repository cloning, project-to-project relationships, arbitrary-session mutation, project rename/description editing, hiding, deletion, detachment, rebinding, schema migration, new HTTP/RPC contract, or additional unqualified core tool.

## Impact

Go domain/application/persistence gains project query DTOs, stable cursor behavior, lookup, registration status, and an atomic conditional association operation. The extension supervisor loads one compiled-in built-in extension and the Agent loop refreshes workspace-dependent state after a successful association. The renderer adds project-specific permission copy and otherwise consumes the existing `session.updated` event. Electron privilege boundaries, Provider credentials, network access, MCP/LSP behavior, dependencies, SQLite schema, and release packaging formats do not change.

## Implementation constraints

The built-in extension uses Manifest v1, namespaced registrations, mode-default activation, exact schema identities, and frozen Tool Snapshots under `ADR-0002`; the core registry remains exactly `read`, `bash`, `edit`, and `write`. Query results are bounded and keyset-paginated. Add and associate are idempotent where their requested end state already exists. Association validates project ID/path consistency and rejects non-coding, already bound, specialized-workspace, or live-terminal sessions before mutation. A conditional transaction updates `sessions.project_id` and `coding_contexts` together so concurrent attempts have one winner and failures roll back. Private paths may appear in the approved tool exchange and permission UI but never in logs or diagnostics.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `PROJECT-AGENT-DOC-001` | `REQ-PROJECT-003` | Work, Requirement, architecture, security, test, and traceability contracts | `AT-PROJECT-003` | Completed |
| `PROJECT-AGENT-STORE-001` | `REQ-PROJECT-003`, `NFR-RELIABILITY-001` | Bounded query and atomic immutable association persistence | `AT-PROJECT-003`, `CT-RELIABILITY-001` | Completed |
| `PROJECT-AGENT-EXT-001` | `REQ-PROJECT-003`, `REQ-EXTENSION-001` | Compiled-in Manifest v1 project extension with three frozen tool registrations | `AT-PROJECT-003`, `AT-EXTENSION-001` | Completed |
| `PROJECT-AGENT-PERM-001` | `NFR-SECURITY-001` | Read/write permission separation and exact project-target approvals | `CT-SECURITY-001` | Completed |
| `PROJECT-AGENT-UI-001` | `REQ-PROJECT-003` | Project permission presentation and live session project refresh | `AT-PROJECT-003` | Completed |

## Acceptance and evidence

- Query covers exact ID, local text search, recent ordering, hidden filtering, current-project inclusion, stable cursors, invalid cursors, and limits.
- Add covers absolute existing directories, missing/non-directory refusal, idempotence, hidden restoration, and no directory creation.
- Associate covers first binding, same-project retry, different-project conflict, ID/path mismatch, non-coding refusal, initial-workspace requirement, live-terminal/specialized-workspace refusal, concurrent attempts, cancellation, and transaction rollback.
- Tool acceptance proves the default primitive registry remains exactly four tools while the built-in extension contributes three mode-default namespaced registrations with frozen identities; the legacy executor is absent and history remains displayable.
- Permission acceptance covers read-only query, request-approval allow/deny/exact remember, auto-approve, full-access, and read-only Agent modes.
- Runtime acceptance proves the next model request after association uses the new workspace-dependent Provider, Skill, extension, and tool state, and the renderer receives the complete updated Session.
- Run `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, and `pnpm build`; capture focused permission UI acceptance before verification and sealing.

Automated and UI evidence recorded on 2026-08-04:

- `AT-PROJECT-003` and `CT-RELIABILITY-001`: `pnpm test:core` passed. Focused persistence coverage proves local case-insensitive search, deterministic keyset pagination, hidden filtering/restoration, idempotent registration, concurrent single-winner binding, and transaction rollback after an injected coding-context failure. Application coverage proves path and reference validation, coding/current-session ownership, immutable/idempotent binding, hidden current-project lookup, specialized-workspace and live-terminal rejection, full `session.updated`, and stable error codes.
- Runtime acceptance uses a provider test server to request `aivo.projects.associate`, then request `read` on the following model boundary. The read succeeds only from the newly associated project and its marker is present in the next Provider request, proving coding context, Provider registry, extension registry, Skills/context candidates, and frozen Tool Snapshot are rebuilt against the new workspace.
- `AT-EXTENSION-001`: registry tests keep the unqualified core registry at exactly `read`, `bash`, `edit`, and `write`; the Host-installed `aivo.projects` Manifest v1 contributes exactly three trusted `activation: default` extension registrations with non-empty implementation, schema, and registration identities. Repository search confirms no executable `search_projects` alias remains.
- `CT-SECURITY-001`: permission tests cover query-without-confirmation, default approval, allow/deny, exact remembered operation/target keys, auto-approve, full access, Plan/read-only denial, idempotent same-project bypass, and pre-confirmation `project_already_bound`. Permission paths are opaque while the renderer receives the specific operation and display root.
- Desktop model tests run through `pnpm scripts:test`. Focused browser QA used the real permission dock and generic tool-activity inspector. The 1440 × 900 and 390 × 844 permission captures show the long path, add/associate copy, permanent-binding warning, rejected/approved states, and exact-project remember label with no horizontal overflow. The 1100 × 720 conflict capture shows `project_already_bound` and the no-switch/no-detach message. Browser console review returned no warnings or errors.
- UI evidence: `artifacts/design-qa/project-permission-decisions-1440x900.jpg`, `artifacts/design-qa/project-permission-decisions-390x844.jpg`, and `artifacts/design-qa/project-association-conflict-1100x720.jpg`.
- Repository gates passed: `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, and `pnpm build`. Lint reported only existing Fast Refresh warnings in shared UI/route primitives; the build reported only existing large-barrel and chunk-size advisory warnings.

## Security and data lifecycle

Project roots and repository contents remain private local data. The model receives bounded project metadata only when the project tools are present and called. Permission requests persist only the exact operation/target needed for current approval behavior. Logs and diagnostics record safe operation IDs, tool names, status, and error codes without raw arguments or root paths. The tools create no files, processes, credentials, network calls, analytics, or new persisted entity types.

## Compatibility and migration

No SQLite schema or HTTP/RPC migration. Existing project/session records remain valid. The retired unqualified `search_projects` executor is not restored; historical calls retain generic rendering. Rollback removes the built-in extension and new internal service behavior without rewriting project or session data created through existing tables.

## Bug root cause (type=bug only)

N/A.
