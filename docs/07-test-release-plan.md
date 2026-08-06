# Aivo test and release plan

## Test layers

| Layer | Purpose | Primary command or evidence |
| --- | --- | --- |
| Documentation | Routing, IDs, traceability, commands, and immutable Work archives | `pnpm docs:check` |
| Governance scripts | Work archive behavior and tamper detection | `pnpm scripts:test` |
| Go unit/integration | Domain, services, transport, persistence, concurrency, providers, tools, cancelled-turn isolation | `pnpm test:core` |
| Tool/extension contracts | Four-tool registry, primitive schema guidance/path semantics, Host pre-call Skill/Manifest/MCP catalog preparation, canonical Skill summary plus validated instruction materialization, request-scoped context injection, session-scoped/one-shot activation, one globally Provider-safe canonical tool name with no wire codec, snapshots, manifests/protocol, explicit legacy plugin format/RPC/process/catalog refusal, fixed/dynamic loopback service startup, trust, lifecycle, Host-derived tool view references, mount-scoped context reuse, and standalone/embedded Web isolation | `AT-TOOL-001`, `AT-PROVIDER-001`, `AT-SESSION-001`, `AT-EXTENSION-001`, `CT-SECURITY-001`, `CT-RELIABILITY-001` evidence in the owning Work |
| Conversational MCP registration | Non-mutating turn-owned proposals, exact native authorization, safe credential references, no pre-confirmation process/network activity, disabled-first persistence, bounded discovery, collision/rollback/replay behavior, restart availability, later-conversation eligibility, and session activation isolation | `AT-EXTENSION-002`, `AT-EXTENSION-001`, `AT-SESSION-001`, `CT-SECURITY-001`, `CT-RELIABILITY-001` evidence in the owning Work |
| Extension runtime messaging | Strict Manifest/API v2 validation, v1 and mixed-version refusal, explicit permission, isolated sender ownership, fixed Host-brokered endpoint routing, one-shot bounds/timeouts, ordered Port messages/events, connection/backpressure limits, same-View no-navigation reuse, EOF/failure/disconnect, and deterministic teardown | `AT-EXTENSION-003`, `CT-SECURITY-001`, `CT-RELIABILITY-001` evidence in the owning Work |
| Visual extension installation | Native directory selection, non-executing Manifest v2 preview, authority/permission review, integrity-bound confirmation, bounded staged copy, copied-integrity verification, atomic managed generation publication, platform application-data root selection, former-root migration, source independence, durable list/enable/disable/uninstall, restart restoration, schema v3 linked-row promotion, schema v4 migration backup/rollback, scoped cleanup, and responsive states | `AT-EXTENSION-004`, `CT-SECURITY-001`, `CT-RELIABILITY-001`, `AT-UI-001` evidence in the owning Work |
| Agent project contract | Bounded project query, existing-directory registration, exact permission targets, immutable current-session association, concurrent winner, rollback, workspace refresh | `AT-PROJECT-003`, `AT-EXTENSION-001`, `CT-SECURITY-001`, `CT-RELIABILITY-001` evidence in the owning Work |
| Desktop static/build | Type safety, lint, renderer/main/preload build | `pnpm lint`, `pnpm build` |
| Provider diagnostics | Configured provider auth and backend behavior | provider smoke command in `docs/provider-backend.md` |
| UI acceptance | Wide/narrow layout, keyboard, loading/error/cancel/permission behavior | Screenshots and `docs/opencode-replacement-manual-acceptance.md` as applicable |
| Package smoke | Installer contents, launch, bundled core, health | `pnpm smoke:release` after platform package command |

## Work verification

- Run focused tests while implementing, then all applicable repository gates.
- New or changed Requirements need stable Test IDs and an updated row in `08-traceability.md`.
- A Bug Work records a reproduction test that fails before the fix and passes after it; cancellation regressions must submit later input and prove no prior command or interaction is replayed.
- Failure, cancellation, repeated execution, timeout, dependency loss, teardown, compatibility, migration, rollback, and security paths are tested or explicitly marked N/A in the Work.
- Dynamic local-service tests bind port zero before announcement and cover handshake timeout/EOF/overflow, protocol and origin refusal, current-generation routing, fixed-service compatibility, idle restart, and deterministic child cleanup.
- Embedded View tests cover same-identity no-navigation reuse, monotonic latest-wins bounded context delivery, stale mount refusal, different-identity teardown, close/reopen, and bridge compatibility for pages that do not subscribe.
- Runtime messaging tests cover v1 denial/v2 permission, sender and current-generation ownership, one-shot timeout/overflow/malformed response, Port connection limits, ordered bounded NDJSON, post/disconnect repetition, EOF/failure, and View/service teardown.
- Local extension installation tests prove preview/copy executes no code, confirmation re-hashes against TOCTOU changes, copied packages match the confirmed integrity before atomic publication, executable trust remains native-user-only, schema v3 receives a verified backup before v4, exact linked rows promote safely, managed installs survive source mutation/removal and restart, partial staging stays invisible, and uninstall/cleanup cannot escape the Host-owned root or delete source.
- Legacy plugin retirement tests prove plugin RPC methods are unsupported, no legacy manifest or stored row starts a process or contributes a tool/hook/provider, no plugin management UI remains, and preserved rows/source folders are untouched.
- UI Work includes wide and narrow screenshots when behavior or layout changes.
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

macOS signing/notarization and platform packaging details remain in `docs/release-quality.md`. Cross-compilation cannot satisfy target-OS package acceptance.

## CI archive baseline

CI must set `AIVO_ARCHIVE_BASE_REF` to a trusted target-branch commit or push-before commit. This prevents a single commit from rewriting a sealed Work and recalculating its digest. A local check defaults to `HEAD`.
