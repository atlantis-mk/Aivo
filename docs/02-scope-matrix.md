# Aivo scope matrix

The matrix owns current product scope. Optional or Future behavior must not be implemented as Required without an accepted Work.

| Capability | Status | Current boundary |
| --- | --- | --- |
| Electron desktop shell | Required | macOS, Windows, and Linux packaging targets |
| Local Go agent runtime | Required | Owns application behavior and privileged runtime coordination |
| Provider configuration and model use | Required | Supported providers already represented by the registry and runtime |
| Local projects and workspace context | Required | User-selected local repositories and folders |
| Conversations, agent execution, tools, permissions, questions | Required | Exact v2 object hierarchy remains `OPEN-002` |
| Files, diffs, diagnostics, terminal, LSP | Required | Local development workflow |
| Worktrees and parallel agent work | Partial | Existing capabilities retained; v2 defaults and lifecycle need approval |
| Skills, plugins, and MCP | Partial | Existing capabilities retained; v2 information architecture remains open |
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
