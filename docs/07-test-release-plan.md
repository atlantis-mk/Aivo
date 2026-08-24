# Aivo test and release plan

## Test layers

| Layer | Purpose | Primary command or evidence |
| --- | --- | --- |
| Documentation | Routing, IDs, traceability, commands, and immutable Work archives | `pnpm docs:check` |
| Governance scripts | Work archive behavior and tamper detection | `pnpm scripts:test` |
| Go unit/integration | Domain, services, transport, persistence, concurrency, providers, normalized runtime/token/cache metrics, tools, cancelled-turn isolation | `pnpm test:core` |
| Tool/extension contracts | Four-tool execution registry, always-visible Host selection control, strict inspect/use source-group auxiliary selection, one-request complete eligible inspection beyond the former 64-tool cap, conversation-persistent use selection and replacement, complete uncapped concrete MCP/extension expansion, complete safe MCP tool-catalog description generation with draft-only non-mutation, replaceable concrete session automatic set, independent manual set, future-selection global visibility, primitive schema guidance/path semantics, Host Skill/Manifest/MCP catalog preparation, canonical Skill summary plus validated instruction materialization, request-scoped context injection, one globally Provider-safe canonical tool name with no wire codec, snapshots, manifests/protocol, explicit legacy plugin format/RPC/process/catalog refusal, fixed/dynamic loopback service startup, trust, lifecycle, Host-derived tool view references, mount-scoped context reuse, and standalone/embedded Web isolation | `AT-TOOL-001`, `AT-PROVIDER-001`, `AT-SESSION-001`, `AT-EXTENSION-001`, `CT-SECURITY-001`, `CT-RELIABILITY-001` evidence in the owning Work |
| Conversational MCP registration | Non-mutating turn-owned proposals, exact native authorization, safe credential references, no pre-confirmation process/network activity, disabled-first persistence, bounded discovery, collision/rollback/replay behavior, restart availability, later-conversation eligibility, and session activation isolation | `AT-EXTENSION-002`, `AT-EXTENSION-001`, `AT-SESSION-001`, `CT-SECURITY-001`, `CT-RELIABILITY-001` evidence in the owning Work |
| Extension runtime messaging | Strict Manifest/API v2 validation, v1 and mixed-version refusal, explicit permission, isolated sender ownership, fixed Host-brokered endpoint routing, one-shot bounds/timeouts, ordered Port messages/events, connection/backpressure limits, same-View no-navigation reuse, EOF/failure/disconnect, and deterministic teardown | `AT-EXTENSION-003`, `CT-SECURITY-001`, `CT-RELIABILITY-001` evidence in the owning Work |
| Visual extension installation | Native directory selection, non-executing Manifest v2 preview, authority/permission review, integrity-bound confirmation, bounded staged copy, copied-integrity verification, atomic managed generation publication, platform application-data root selection, former-root migration, source independence, durable list/enable/disable/uninstall, restart restoration, schema v3 linked-row promotion, schema v4 migration backup/rollback, scoped cleanup, and responsive states | `AT-EXTENSION-004`, `CT-SECURITY-001`, `CT-RELIABILITY-001`, `AT-UI-001` evidence in the owning Work |
| Agent project contract | Bounded project query, existing-directory registration, exact permission targets, immutable current-session association, concurrent winner, rollback, workspace refresh | `AT-PROJECT-003`, `AT-EXTENSION-001`, `CT-SECURITY-001`, `CT-RELIABILITY-001` evidence in the owning Work |
| Dynamic Agent modes | Exactly one visible built-in Assistant default, retired built-in ID fallback, required hidden workers, Assistant origin/reset, bounded CRUD, protected internal IDs, session/association referenced deletion and role-change refusal, management toolset omission, built-in/safe runtime capability restoration, bounded subagent allowlists, multi-select creation/editing, generated narrowed Provider delegation schema/context, execution revalidation, global/project precedence, restart persistence, schema-v6 creation, schema-v7 payload cleanup, schema-v8 association contract, picker integration, and responsive management states | `AT-AGENT-001`, `AT-SESSION-001`, `CT-RELIABILITY-001`, `AT-UI-001` evidence in the owning Work |
| Prompt management | Embedded Markdown integrity, bounded parsing, variable/reference validation, atomic working/active revisions, invalid fallback, immutable execution snapshots, Agent prompt-reference migration, retirement of the managed subagent protocol in favor of Core-generated association context, typed CRUD/reload, quick-prompt refresh, and responsive editor states | `AT-PROMPT-001`, `AT-AGENT-001`, `AT-WORKSPACE-001`, `CT-SECURITY-001`, `CT-RELIABILITY-001`, `AT-UI-001` evidence in the owning Work |
| Desktop static/build | Type safety, lint, renderer/main/preload build | `pnpm lint`, `pnpm build` |
| Provider management and diagnostics | Settings catalog/ecosystem-refresh/configured-provider-refresh/add/connect/confirmed-delete states, public Provider/model cache replacement, immediate add-flow model availability, protocol-compatible ecosystem Provider refresh, unsupported/failing refresh feedback, persisted-config remote model refresh, per-Provider cache/default-list replacement, configured provider auth, active-preference preservation, and backend behavior | `AT-PROVIDER-001`, `AT-UI-001`, provider smoke command in `docs/provider-backend.md` |
| UI acceptance | Wide/narrow layout, keyboard, loading/error/cancel/permission behavior, and composer runtime-statistics truncation/clearance | Screenshots and `docs/opencode-replacement-manual-acceptance.md` as applicable |
| Package smoke | Installer contents, launch, bundled core, health | `pnpm smoke:release` after platform package command |
| Source license | Canonical PolyForm Noncommercial text, source-available/noncommercial wording, commercial authorization boundary, required notice, and workspace package-manifest consistency | `CT-LICENSE-001` in `scripts/source-license-metadata.test.mjs` |
| Release publication | Tag/version validation, normalized four-platform asset contract, immutable R2 publication plan, stable channel manifest, and GitHub Release asset set | `CT-RELEASE-001` in `scripts/release-publication.test.mjs` plus a successful tagged Actions run |
| Desktop automatic update | Stable SemVer/platform selection, fixed R2/GitHub identity and schema validation, cross-source name/size/SHA-256 agreement, bounded streaming download/progress, exact byte verification, typed IPC isolation, startup/manual flows, cancellation/cleanup, user-confirmed package handoff, and responsive Settings states | `AT-UPDATE-001`, `CT-SECURITY-001`, `CT-RELIABILITY-001` evidence in the owning Work |
| Public repository | External backup, secret scan, new root history, new Public repository identity, metadata, and repository security settings | `CT-REPOSITORY-001` evidence in the owning Work |

## Work verification

- Run focused tests while implementing, then all applicable repository gates.
- New or changed Requirements need stable Test IDs and an updated row in `08-traceability.md`.
- A Bug Work records a reproduction test that fails before the fix and passes after it; cancellation regressions must submit later input and prove no prior command or interaction is replayed.
- Failure, cancellation, repeated execution, timeout, dependency loss, teardown, compatibility, migration, rollback, and security paths are tested or explicitly marked N/A in the Work.
- Dynamic local-service tests bind port zero before announcement and cover handshake timeout/EOF/overflow, protocol and origin refusal, current-generation routing, fixed-service compatibility, idle restart, and deterministic child cleanup.
- Embedded View tests cover same-identity no-navigation reuse, monotonic latest-wins bounded context delivery, stale mount refusal, different-identity teardown, close/reopen, and bridge compatibility for pages that do not subscribe.
- Runtime messaging tests cover v1 denial/v2 permission, sender and current-generation ownership, one-shot timeout/overflow/malformed response, Port connection limits, ordered bounded NDJSON, post/disconnect repetition, EOF/failure, and View/service teardown.
- Local extension installation tests prove preview/copy executes no code, confirmation re-hashes against TOCTOU changes, copied packages match the confirmed integrity before atomic publication, executable trust remains native-user-only, schema v3 receives a verified backup before v4, exact linked rows promote safely, managed installs survive source mutation/removal and restart, partial staging stays invisible, and uninstall/cleanup cannot escape the Host-owned root or delete source.
- Desktop update tests prove the renderer cannot choose a URL/path/command, the R2 and same-tag GitHub metadata agree before bytes are accepted, redirects/oversize/ambiguity/downgrade/mismatch fail closed, partial files are removed on failure/cancellation/shutdown, repeated actions retain one owner, and each target OS exposes only the documented explicit package handoff.
- Legacy plugin retirement tests prove plugin RPC methods are unsupported, no legacy manifest or stored row starts a process or contributes a tool/hook/provider, no plugin management UI remains, and preserved rows/source folders are untouched.
- UI Work includes wide and narrow screenshots when behavior or layout changes.
- Agent-mode tests prove code defaults require no rows, built-in overrides reset exactly, management data omits toolsets while runtime restores code/safe capabilities, association/self/role/target bounds are enforced, custom deletion and role changes refuse durable references, empty associations omit model delegation, non-empty associations narrow Provider exposure and reject forged targets before child creation, project file overlays remain higher precedence, schema v5 receives a verified backup before v6, schema v6 receives a verified backup before v7 payload cleanup, and schema v7 receives a verified backup before v8.
- `Verified` requires command/platform evidence in `change.md`, followed immediately by Work archive sealing.

## Release gates

| Gate | Requirement |
| --- | --- |
| GATE-1 | `pnpm docs:check` and `pnpm scripts:test` pass |
| GATE-2 | `pnpm test:core` passes |
| GATE-3 | `pnpm lint` and `pnpm build` pass without new warnings |
| GATE-4 | Applicable provider and end-to-end/manual acceptance passes |
| GATE-5 | Target-OS package and `pnpm smoke:release` pass |
| GATE-6 | Migration backup, forward migration, failure recovery, and rollback pass when data changes |
| GATE-7 | Release record references only sealed Work and the same-name Git tag is created |
| GATE-8 | R2 immutable assets and GitHub Release assets have the same normalized names, sizes, and SHA-256 digests; the stable channel manifest is published last and verified by readback |
| GATE-9 | Packaged update check/download/handoff passes on each target OS and architecture using the published stable-channel contract; unsigned artifacts remain explicit and non-silent |

macOS signing/notarization and platform packaging details remain in `docs/release-quality.md`. Cross-compilation cannot satisfy target-OS package acceptance.

## CI archive baseline

CI must set `AIVO_ARCHIVE_BASE_REF` to a trusted target-branch commit or push-before commit. This prevents a single commit from rewriting a sealed Work and recalculating its digest. A local check defaults to `HEAD`.
