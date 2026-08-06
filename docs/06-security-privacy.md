# Aivo security and privacy

## Trust boundaries

- React renderer: unprivileged presentation and interaction layer.
- Electron main/preload: desktop capability and OS boundary.
- Local Go core: application authorization, agent runtime, and persistence boundary.
- Providers, MCP servers, Manifest v2 extensions, skills, web services, LSP servers, and child processes: external or separately trusted dependencies.
- User-selected project and worktree files: private user data, not telemetry or fixtures.

## Secret ownership

Provider credentials, OAuth tokens, authorization headers, signing material, and secure-store values remain in privileged local services. UI and diagnostic DTOs expose only status, provider identity, expiry, or other non-secret summaries. Secrets are cleared from transient memory when their owning session or process ends where supported by the runtime.

## Execution safety

- Renderer input is validated at the privileged boundary, including paths, origins, command policy inputs, and identifiers.
- File and command tools enforce environment and permission policy before execution, but these checks and prompts are not described as process containment.
- The four core primitives and executable extensions run with the authority of their containing local or remote process. Containers, VMs, OS sandboxes, micro-VMs, SSH hosts, or remote sandboxes own any claimed containment state.
- Child processes and external clients inherit explicit cancellation and cannot outlive their owner silently.
- Extension, skill, and MCP capabilities are manifest/configuration driven. Untrusted executable extensions cannot start, connect, receive credentials, or activate; models cannot install or trust them. Legacy plugin packages and stored rows are inert and cannot start or contribute capability.
- Local extension installation starts only from a native user directory selection. Core previews without execution, revalidates the exact canonical source and complete integrity, copies only bounded symlink-free regular content into a private Host-owned staging directory beneath platform application data, reloads the copy, and atomically publishes only the confirmed integrity generation. Runtime never falls back to the source; later source changes cannot become executable updates. Managed package paths and contents never enter model context, extension Web content, logs, or diagnostics. Former-root migration and uninstall deletion are refused unless the exact target is a validated descendant of the applicable Host-owned root. The deeper application-data location reduces accidental discovery but is not a security boundary from the owning OS user. The UI states that executable extensions run with the user's OS authority.
- A model may create only a non-mutating Host-validated MCP registration proposal. Exact native user confirmation is required before the Host persists, connects to, or starts the source; approval is bound to the immutable session/turn proposal, cannot be model-authored or replayed, and carries credential references rather than raw values.
- Host-owned project tools may expose bounded registered-project metadata to the active model. Registering a root or binding the current conversation requires the active write-permission mode, uses an exact target, cannot create content, and cannot switch or detach an existing project association.
- Extension credentials are explicitly bound to Host-owned secure-store entries, leased only to the operation that needs them, and cannot be enumerated. Extension Web views receive neither ambient credentials nor privileged Electron APIs.
- A dynamically announced extension service endpoint is accepted only after the supervised child owns an explicit non-zero loopback HTTP port and emits one bounded versioned readiness record. The endpoint is generation-scoped and ephemeral; timeouts, malformed records, credentials, non-root routes, and non-loopback origins terminate startup before the service becomes ready.
- An extension Web page embedded in the main window still runs in its own ephemeral sandboxed WebContents session. Renderer-supplied bounds and tool details cannot change the Host-resolved logical origin, backend proxy, credential binding, navigation policy, or declared action set.
- Reused embedded Views receive only mount-scoped, bounded, revisioned Host context updates for operation/session/turn/tool identity. A stale mount or renderer update cannot retarget a replacement View or mutate origin, backend, bearer, actions, route, surface, or extension identity.
- Manifest/API v2 runtime messaging is same-extension only and opt-in. Electron main validates the registered isolated sender, View identity, permission, current ready service generation, fixed endpoint, JSON size, timeout, and per-View Port limit before injecting the Host bearer. Message contents are not logged or persisted; Port readers enforce bounded events/backpressure and are aborted on View, renderer, or service teardown.
- Permission denial, cancellation, timeout, dependency failure, and malformed external responses are first-class outcomes.

## Logging and diagnostics

Logs use structured safe metadata and operation IDs. They must not contain API keys, refresh tokens, authorization headers, raw prompt content, raw tool arguments/results with private content, local credential stores, or unsanitized provider responses. User-facing diagnostic exports require the same redaction boundary.

## Data protection

- Persistence migrations back up data and define rollback/compensation before mutation.
- Test and migration fixtures are synthetic or irreversibly sanitized.
- No analytics or remote telemetry is in current scope.
- Security bugs involving credential exposure, unauthorized execution, data loss, or boundary bypass require a `security` or `bug` Work, a regression test, and durable evidence.

## Security-sensitive ADR triggers

Create or revise an ADR for privilege-boundary ownership, command/sandbox authorization, provider credential ownership, plugin/MCP trust, persistence ownership or irreversible migration, remote service/telemetry introduction, or a new platform security boundary.
