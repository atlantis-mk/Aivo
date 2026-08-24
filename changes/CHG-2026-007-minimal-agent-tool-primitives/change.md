# Reduce Agent tools to four primitives with extensible activation

## Problem or goal

The current Agent runtime exposes overlapping file, search, shell, terminal, diagnostics, formatter, Git, web, planning, automation, Agent, skill, MCP, and discovery tools. Several tools represent product features or implementation mechanisms rather than distinct Agent intentions. This increases prompt size, model choice ambiguity, compatibility surface, and lifecycle complexity.

The goal is a minimal sufficient default: `read`, `bash`, `edit`, and `write`. These cover observation, general execution, exact local modification, and complete file creation or replacement. Long-tail capability comes from language-neutral extensions whose tools are selected before a model call by Host policy, Agent Mode, user pinning, or an auxiliary-model resolver. Complex extension UI is preferably served as an isolated Web Service surface.

## Expected behavior

- `REQ-TOOL-001`: a default coding Agent sees exactly the unqualified tools `read`, `bash`, `edit`, and `write`.
- `read` reads one text or supported image file. Text uses 1-based line offset and bounded line count; the tool does not list directories, search, or mutate.
- `bash` runs one foreground, non-interactive Bash command in the current Execution Environment and returns stdout, stderr, and exit status. Bounded model output retains useful tail content while complete truncated output is stored in private session temporary files.
- `edit` applies one or more exact, unique, non-overlapping replacements against one original file snapshot and commits the complete batch atomically. It never uses fuzzy matching.
- `write` creates or completely overwrites one text file, creates missing parent directories, and commits atomically.
- `apply_patch` is removed directly. Aivo does not ship a compatibility alias or replacement patch extension.
- Git, testing, builds, repository search, formatting, and diagnostics use `bash` and existing CLI ecosystems rather than default specialized tools.
- LSP, web, MCP, sub-Agents, automation, interactive terminals, policy controls, alternate execution environments, and other long-tail capabilities are extensions.
- Planning, questions, skills, and model/tool activation are Host protocols or context mechanisms, not default Agent execution tools.
- `REQ-EXTENSION-001`: extensions may be Go built-ins, language-agnostic child processes, local services, external services, or static resources. They use one versioned manifest and protocol boundary.
- Extension tool activation is distinct from installation, trust, loading, eligibility, and execution. Only the frozen Tool Snapshot is exposed to a model request.
- The existing auxiliary-model tool selector becomes a Host pre-call resolver. Its selection is validated and bounded; it cannot install, trust, authorize, or execute extensions.
- Extension Web Service UI is rendered only through isolated Host surfaces and does not receive privileged Electron APIs or ambient credentials.
- `NFR-SECURITY-001`: primitives and extension processes use the starting user's OS authority. Real containment is supplied by a container, VM, OS sandbox, micro-VM, SSH host, or remote sandbox. Aivo must not describe in-process path checks or confirmation prompts as containment.
- `NFR-RELIABILITY-001`: tools, extension processes, services, streams, calls, temporary artifacts, and Web views have owners, bounded outputs, cancellation, draining, deterministic teardown, and explicit failure states.

The accepted design record is preserved in `design.md` and `extension-protocol.md`; the primary specifications now own the stable behavior.

## Non-goals

- Selecting or bundling a specific container, VM, micro-VM, SSH, or remote sandbox implementation.
- Treating the extension protocol, extension trust prompt, workspace path, or permission UI as a process sandbox.
- Adding a built-in general browser or allowing extension UI to navigate arbitrary privileged Electron content.
- Reintroducing `apply_patch`, fuzzy edits, default background Bash, default MCP, default sub-Agents, default plan tools, or default permission tools.
- Letting a model install or trust executable extension code.
- Closing current product hierarchy, worktree-default, or browser-scope open decisions.
- Changing persistence schema in this Work.

## Impact

- Go domain/application: replace the broad built-in registry, split catalog/registration/activation/execution state, move auxiliary resolution before the primary model request, and add a language-neutral Extension Supervisor.
- Filesystem/processes: consolidate observation and mutations under four primitives, add a per-file mutation queue, bind all primitives to one Execution Environment, and manage bounded Bash artifacts.
- Providers: send four core schemas plus a small frozen set of activated extension schemas in deterministic order.
- Extensions: introduce Manifest v1 and Protocol v1 for Go built-ins, processes written in any language, local Web Services, external services, MCP adapters, and static context resources.
- Renderer/Electron: add isolated Web surfaces and an internal extension-resource proxy without Node integration or privileged preload access. This is a new renderer/extension trust boundary and is governed by `ADR-0002`.
- Persistence: no schema change is planned. Existing execution-state metadata may be reinterpreted or replaced within current serialized metadata; any required schema mutation must create a separate migration Work.
- Historical UI: generic tool-call history remains renderable when an implementation or extension is removed. Extension-specific Web views degrade to a bounded stored summary.
- Platforms: `bash` always means Bash-compatible syntax. Supported Windows behavior must be decided and verified before Accepted.
- Dependencies: a bundled Bash, extension runtime manager, Web proxy dependency, or sandbox product requires explicit review before adoption.

## Implementation constraints

- The four core names and schemas are stable and reserved. Ordinary extensions cannot override them.
- An Environment Extension replaces the operations behind all four tools as one coherent environment; it does not re-register their names or silently route only `bash` elsewhere.
- `read`, `edit`, `write`, and truncated Bash artifact paths must refer to the same active environment.
- Tool selection changes only at a model-call boundary. A Tool Snapshot records exact registration IDs and schema hashes and remains valid for the calls produced by that request.
- The default primary model never receives `tool_resolve`, `tool_search`, `tool_list`, `tool_detail`, or `tool_call`. An optional explicitly enabled discovery extension may provide a namespaced fallback without changing the core default.
- Auxiliary selection receives only eligible, sanitized catalog summaries; selected names are validated against exact registrations and activation policy before schemas are loaded.
- Extension manifests are read without executing extension code. Project-local or changed executable extensions require explicit trust before loading.
- Extension results separate bounded model content from structured UI details and artifact references.
- Web UI uses an internal `aivo-extension` resource boundary, isolated browser context, restrictive CSP, and a versioned message bridge. It cannot call privileged services directly.
- Secret values are never stored in manifests or injected ambiently. Extension credentials are explicitly bound and supplied through a Host credential broker.
- No product code changes until this Work is Accepted. Before Implementing, merge the accepted behavior into Scope, `REQ-TOOL-001`, `REQ-EXTENSION-001`, Security, Architecture, Test Plan, and Traceability, and accept `ADR-0002`.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `TOOL-PRIM-001` | `REQ-TOOL-001` | Freeze four primitive schemas, bounded results, errors, and Execution Environment operations | `AT-TOOL-001` | Complete |
| `TOOL-PRIM-002` | `REQ-TOOL-001` | Implement the four-tool default registry and direct `apply_patch` removal | `AT-TOOL-001` | Complete |
| `TOOL-PRIM-003` | `REQ-TOOL-001` | Implement text/image read, bounded Bash artifacts, exact batch edit, atomic write, and mutation queue | `AT-TOOL-001`, `CT-RELIABILITY-001` | Complete |
| `TOOL-ACT-001` | `REQ-EXTENSION-001` | Move auxiliary tool selection to Host pre-call activation with pinned, warm, and current-turn state | `AT-EXTENSION-001` | Complete |
| `TOOL-SNAP-001` | `REQ-EXTENSION-001` | Add immutable registration identities and per-request Tool Snapshots | `AT-EXTENSION-001`, `CT-RELIABILITY-001` | Complete |
| `EXT-PROTO-001` | `REQ-EXTENSION-001` | Implement Manifest v1 and Protocol v1 for built-in, process, service, external, and static runtimes | `AT-EXTENSION-001` | Complete |
| `EXT-LIFE-001` | `NFR-RELIABILITY-001` | Implement trust, enable, start, ready, activate, drain, stop, update, crash, and cancellation lifecycle | `AT-EXTENSION-001`, `CT-RELIABILITY-001` | Complete |
| `EXT-UI-001` | `REQ-EXTENSION-001` | Implement isolated Web Service views, Host proxy, bridge, fallback rendering, and teardown | `AT-EXTENSION-001`, `CT-SECURITY-001` | Complete |
| `EXT-SEC-001` | `NFR-SECURITY-001` | Implement credential binding, safe metadata, explicit isolation state, and project-extension trust | `CT-SECURITY-001` | Complete |
| `TOOL-MIG-001` | `REQ-TOOL-001` | Remove or migrate legacy tools and preserve historical rendering without execution aliases | `AT-TOOL-001` | Complete |

## Acceptance and evidence

- Default registry acceptance proves exactly `read`, `bash`, `edit`, and `write` are exposed in stable order.
- `read` covers text lines, pagination, size bounds, supported images, image resizing/bounds, binary refusal, missing files, cancellation, and paths in local and alternate environments.
- `bash` covers stdout/stderr/exit status, timeout units, maximum timeout, tail-preserving truncation, complete private temporary artifacts, cancellation, child-process teardown, Bash unavailable, and environment loss.
- `edit` covers multiple exact unique matches against one original snapshot, empty/no/multiple matches, duplicate or overlapping edits, external file changes, diff bounds, all-or-nothing behavior, and per-file serialization.
- `write` covers create, complete overwrite, parent creation, atomic commit, concurrent mutation ordering, failure cleanup, and bounded result details.
- Migration acceptance proves `apply_patch` is absent, removed specialized tools are not advertised, CLI-equivalent tasks use `bash`, and historical records remain readable.
- Resolver acceptance covers deterministic prefiltering, strict candidate-only auxiliary selection, maximum activation count, failure fallback to core, no trust/permission escalation, pinned/warm/current-turn lifecycle, and catalog revision invalidation.
- Extension acceptance covers every runtime type, invalid or changed manifests, protocol negotiation, static and dynamic catalogs, name collisions, schema drift, crashes, restarts, draining, updates, cancellation, and service loss.
- Web UI acceptance covers page/dialog/tool-detail/settings/notification surfaces, CSP, origin isolation, no privileged APIs, bounded data references, unavailable-extension fallback, responsive layout, keyboard/focus behavior, and teardown.
- Security acceptance proves accurate sandboxed/unsandboxed reporting, same-user process authority, no ambient credential enumeration, safe logs, project trust behavior, and external sandbox denial propagation.
- Applicable gates are `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, and `pnpm build`, plus target-OS package and Bash evidence.
- `AT-TOOL-001` and `CT-RELIABILITY-001`: `pnpm test:core` passed on 2026-08-03. Focused Go coverage proves the exact four-tool order, reserved-name protection, text/image bounds, exact atomic edit/write, approval-time stale-file rejection, per-file serialization, coherent alternate-environment routing, independent stdout/stderr tail artifacts, `0600` artifact permissions, cancellation of Bash child process groups, immutable snapshots, and deterministic extension teardown.
- `AT-EXTENSION-001` and `CT-SECURITY-001`: the same full Core suite passed Manifest/Protocol v1 validation for built-in, process, supervised service, external service, and static runtimes; trust/integrity, catalog/schema identity, policy interception, context bounds, credentials, MCP v1 adaptation, update/drain/restart/cancel behavior, isolated-view descriptors, external proxy credentials, and safe logging. Electron entry/preload/view scripts also passed `node --check`.
- Repository gates passed on 2026-08-03: `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, and `pnpm build`. Lint reported only existing Fast Refresh warnings in shared UI/routes; the build reported only existing large-chunk advisories.
- Platform acceptance passed on macOS 14.8.7 (`x86_64`) with GNU Bash 3.2.57: `pnpm package:mac -- --dir` produced the Darwin x64 application directory and `pnpm smoke:release` started the packaged Core binary and reached `/health`. `git diff --check` also passed.

## Security and data lifecycle

Core primitives, executable extensions, and extension services can act with the authority of their containing process or remote environment. Trusting an extension is equivalent to trusting executable local software; activation merely controls availability and prompt exposure. External containment state must be accurate and visible. Renderer isolation, schema validation, bounded output, cancellation, atomicity, safe logs, explicit secret binding, and least credential disclosure remain required correctness and privacy controls.

Tool arguments, file contents, command output, service payloads, and UI data are private user data. Model content is bounded. Complete truncated Bash output and large extension results use private per-session artifacts in the active environment with deterministic cleanup and startup recovery for stale files. Persisted history stores only the bounded data necessary for conversation continuity and generic fallback rendering. Extension Web UI receives references or sanitized state, not Provider or credential-store secrets.

## Compatibility and migration

The migration is a development-version contract reset. New model requests stop advertising legacy tools. `apply_patch` is deleted directly. Historical calls remain generic display records and do not require a current executor. A temporary compatibility extension may be considered only for exact semantics required by active development data, but no such alias is part of this Work's default design.

No persistence schema migration is planned. If execution metadata cannot support activation state and Tool Snapshots without schema change, implementation must stop and create a separate migration Work with backup and rollback evidence. Rollback restores the previous registry and resolver implementation; ordinary user filesystem mutations are not automatically reversed.

## Bug root cause (type=bug only)

N/A; this is an architecture and security redesign.
