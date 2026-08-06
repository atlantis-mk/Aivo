# Aivo scope matrix

The matrix owns current product scope. Optional or Future behavior must not be implemented as Required without an accepted Work.

| Capability | Status | Current boundary |
| --- | --- | --- |
| Electron desktop shell | Required | macOS, Windows, and Linux packaging targets |
| Local Go agent runtime | Required | Owns application behavior and privileged runtime coordination |
| Provider configuration and model use | Required | Supported providers already represented by the registry and runtime |
| Local projects and workspace context | Required | User-selected local repositories/folders, one configured initial directory for unscoped conversations, and Agent-assisted project query/registration with immutable one-time conversation binding |
| Conversations, agent execution, tools, permissions, questions | Required | Coding Agents receive four core primitives by default, with per-conversation controls for the built-in tool surface; exact v2 object hierarchy remains `OPEN-002` |
| Files, diffs, diagnostics, terminal, LSP | Partial | `read`, `bash`, `edit`, and `write` are core; diagnostics, interactive terminals, and LSP are optional extensions, while chat may open a contextual tool inspector that hosts an associated isolated extension tool page |
| Worktrees and parallel agent work | Partial | Existing capabilities retained; v2 defaults and lifecycle need approval |
| Skills, extensions, and MCP | Partial | Manifest/API v2 language-neutral extensions replace the retired legacy plugin runtime and provide optional tools, context, MCP adapters, fixed or dynamically announced loopback services, isolated Web views, Host-brokered messaging, and native user-confirmed integrity-bound installation into Aivo-managed local package storage; Host-confirmed conversational MCP registration is supported, while legacy plugin compatibility/conversion, Chrome Web Store compatibility, remote extension registries, content scripts, generic model-driven installation, and v2 information architecture remain out of this slice |
| Settings, recovery, diagnostics, data management | Partial | Existing settings surface is incomplete |
| Versioned v2 HTTP resource contracts | Partial | New contracts use `/api/v2`; v1 method RPC remains a compatibility adapter |
| Explicit persistence migrations | Partial | v1 baseline exists; v2 transition framework is required before schema change |
| Built-in browser UI | Future | Previously removed; decision is `OPEN-005` |
| Cloud sync, collaboration, Aivo accounts | Out | No approved server or multi-user scope |
| Mobile clients | Out | Desktop only |
| Telemetry or remote analytics | Out | No approved collection contract |

## Platform scope

- Development and automated core checks may run on any supported development OS.
- Installer, signing, notarization, and OS integration acceptance must run on the target OS.
- A platform is not release-ready until its package and smoke gates in `07-test-release-plan.md` pass.

## Scope changes

A scope change requires a Work with `type: feature`, `migration`, `security`, or `governance` as applicable. The Work must update this matrix, the affected Requirement, Traceability, compatibility behavior, and any applicable ADR before implementation.
