# Aivo security and privacy

## Trust boundaries

- React renderer: unprivileged presentation and interaction layer.
- Electron main/preload: desktop capability and OS boundary.
- Local Go core: application authorization, agent runtime, and persistence boundary.
- Providers, MCP servers, plugins, skills, web services, LSP servers, and child processes: external or separately trusted dependencies.
- User-selected project and worktree files: private user data, not telemetry or fixtures.

## Secret ownership

Provider credentials, OAuth tokens, authorization headers, signing material, and secure-store values remain in privileged local services. UI and diagnostic DTOs expose only status, provider identity, expiry, or other non-secret summaries. Secrets are cleared from transient memory when their owning session or process ends where supported by the runtime.

## Execution safety

- Renderer input is validated at the privileged boundary, including paths, origins, command policy inputs, and identifiers.
- File and command tools enforce workspace and permission policy before execution.
- Child processes and external clients inherit explicit cancellation and cannot outlive their owner silently.
- Plugin, skill, and MCP capabilities are manifest/configuration driven and do not gain ambient credentials or unrestricted local access.
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
