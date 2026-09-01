# ADR-0002: Use four Agent primitives with external isolation and language-neutral extensions

- Status: Superseded in part by `ADR-0016`
- Date: 2026-08-03
- Related Work: `CHG-2026-007-minimal-agent-tool-primitives`
- Closes OPEN: none

## Context

The current runtime combines a broad built-in tool catalog, model-visible discovery, workspace and permission policy, multiple execution styles, MCP/plugins, and Agent/automation workflows. Overlap increases prompt size and ambiguous tool selection. In-process policy checks still run above a local process with the user's OS authority and therefore cannot honestly provide full command containment. The TypeScript-plugin mental model is also too narrow for Go built-ins, Python, compiled executables, local services, remote services, MCP adapters, and rich extension UI.

The replacement must retain high-frequency deterministic file operations, general CLI capability, context bounds, auxiliary-model activation, renderer privilege separation, secret ownership, cancellation, and historical observability while allowing optional capability and Web UI to evolve outside the core tool contract.

## Decision

- The default primary model MUST receive only `read`, `bash`, `edit`, and `write` as unqualified core tools.
- `apply_patch` MUST be removed directly without a shipped execution alias or replacement patch extension.
- Git, repository search, tests, builds, formatting, and diagnostics MUST use Bash/CLI composition by default; LSP, web, terminals, sub-Agents, automation, MCP, and other long-tail capability MUST be optional extensions.
- The four tools MUST bind to one coherent Execution Environment. An Environment Extension MUST replace all primitive operations atomically and MUST NOT silently route only part of the environment elsewhere.
- Primitive and executable extension operations MUST run with the authority of their containing local or remote process. Aivo MUST NOT claim that workspace checks, allowlists, or prompts provide process containment. Real isolation MUST come from an OS sandbox, container, VM, micro-VM, SSH/remote host, or equivalent boundary.
- Core correctness controls MUST retain schema validation, bounded context/results, cancellation, process ownership, per-file mutation serialization, exact edit semantics, atomic writes, safe logs, renderer privilege separation, and Host-owned credentials.
- Extensions MUST use a versioned language-neutral Manifest and Protocol supporting Go built-ins, arbitrary supervised executables, local services, external services, and static resources.
- Extension discovery and trust MUST be separate from enablement, readiness, activation, prompt exposure, authorization, and execution. Models MUST NOT install or trust executable extensions.
- Superseded by `ADR-0016`: auxiliary selection initializes a conversation automatic set, and the default primary model receives only the bounded `resource_resolve` Host control for intentional replacement; other discovery bridge tools remain hidden.
- Each primary model request MUST use a frozen Tool Snapshot containing exact registration and schema identities.
- Every executable tool MUST have one canonical ASCII identifier matching `^[A-Za-z0-9_-]+$` with a 64-byte maximum before registration. Manifest and registry boundaries MUST reject invalid names, generated MCP adapter names MUST follow the same rule while retaining their upstream names separately, and Provider adapters MUST NOT encode, decode, escape, or alias tool identities.
- Complex extension presentation SHOULD use an isolated Web Service view through Host-owned surfaces and proxying. Extension Web content MUST NOT receive Node integration, privileged preload APIs, ambient credentials, or unrestricted privileged navigation.
- Tool results MUST separate bounded model content from structured UI details and artifact/view references.

## Rationale

- Four familiar primitives minimize schema context and tool-choice ambiguity while Bash retains access to mature development CLIs.
- Exact batch edits fail safely instead of guessing, and full writes make overwrite intent explicit.
- Host-side auxiliary selection preserves dynamic long-tail capability without making discovery a fifth default model tool.
- A language-neutral protocol lets built-in Go code, Python, compiled executables, Web Services, remote systems, and MCP share lifecycle and identity rules.
- A coherent Execution Environment prevents local file tools from disagreeing with remote or contained command execution.
- Web Service views allow rich extension UI while a Host proxy and isolated browser context protect Electron privileged boundaries.
- Naming the real external isolation boundary avoids overstating the protection of in-process prompts and path policy.

## Consequences

- Existing tool names, schemas, resolver flow, permission states, runtime registry, and provider prompt assembly change incompatibly during development.
- `apply_patch` and specialized built-ins are removed or moved; historical records require generic fallback rendering.
- A new Extension Supervisor, manifest/protocol compatibility policy, registration identity, Tool Snapshot, Web proxy, and isolated UI bridge are required.
- Executable extension trust is equivalent to local software trust. Separate processes improve reliability but do not create containment.
- Supported Windows builds need a conforming Bash strategy before acceptance.
- A specific sandbox, bundled language runtime, bundled Bash, packaging/signing system, or persistence migration requires separate approval when it crosses its applicable boundary.
- Extension UI expands the renderer trust surface and requires focused security, lifecycle, and responsive acceptance.

## Rejected alternatives

- Keep the broad tool set and improve descriptions: descriptions do not remove overlapping capability, prompt cost, or lifecycle surface.
- Retain `apply_patch` alongside exact edit: it creates a second local-modification intention and preserves ambiguous selection.
- Make model-visible discovery a permanent fifth core tool: it adds a round trip and makes dynamic selection depend on the primary model recognizing a missing capability.
- Make extensions TypeScript-only or load arbitrary code into Electron: this excludes Go/Python/executables and enlarges the privileged renderer/main boundary.
- Allow independent remote override of only Bash: command-created state would disagree with local read/edit/write state.
- Treat permission prompts or workspace roots as the sandbox: they cannot reliably contain arbitrary process and extension execution.
- Render extension output only as native hard-coded UI: it couples every extension presentation change to the desktop release.
- Load arbitrary extension Web URLs with privileged preload access: it exposes the Electron boundary to extension content.

## Verification

`AT-TOOL-001` verifies the four-tool registry, primitive schemas, direct `apply_patch` removal, bounded outputs, exact and atomic mutation, coherent environments, and historical fallback. `AT-EXTENSION-001` verifies Manifest/Protocol compatibility, all runtime types, catalogs, auxiliary activation, registration snapshots, lifecycle, collisions, dynamic MCP behavior, and Web views. `CT-SECURITY-001` verifies trust, same-user authority, accurate containment state, credential binding, redaction, Web isolation, and external sandbox denial. `CT-RELIABILITY-001` verifies cancellation, process trees, per-file queues, artifacts, service health/restart, draining, update/removal, and deterministic teardown.
