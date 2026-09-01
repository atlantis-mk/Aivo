# Aivo scope matrix

The matrix owns current product scope. Optional or Future behavior must not be implemented as Required without an accepted Work.

| Capability | Status | Current boundary |
| --- | --- | --- |
| Electron desktop shell | Required | macOS, Windows, and Linux packaging targets; packaged builds automatically check the trusted stable channel and offer a verified, user-confirmed native update handoff |
| Local Go agent runtime | Required | Owns application behavior and privileged runtime coordination |
| Local assistant identity | Required | Initialization persists one bounded user-selected name, defaulting to `Aivo`, and uses it for the built-in Assistant and desktop home presentation without renaming technical identifiers |
| Provider configuration and model use | Required | Supported providers already represented by the registry and runtime |
| Local projects and workspace context | Required | User-selected local repositories/folders, one configured initial directory for unscoped conversations, and Agent-assisted project query/registration with immutable one-time conversation binding |
| Conversations, agent execution, tools, permissions, questions | Required | Coding Agents receive four always-on core execution primitives plus the always-on `update_plan` and `ask_user` Host controls; all six are omitted from management and selection surfaces, and a separate Host-owned `resource_resolve` control lets the Agent inspect or activate optional capabilities during its work; permission mode is an explicit choice between request approval and full access, with no automatic-approval mode, where full access suppresses approval prompts for trusted registered tools after Core validation succeeds; conversations own independent manual and replaceable automatic tool sets, while optional resources are not automatically searched or injected before the first primary request; settled local model work exposes bounded runtime/token/cache statistics beside the active composer without remote telemetry; model-visible history supports durable local summary compaction at a default 80% context-pressure threshold plus an explicit composer action without deleting visible history; global Agent modes remain Core-managed dynamic definitions with one visible built-in Assistant default, required hidden workers, user-created modes, bounded subagent associations, code-owned capability sets, and per-project file overlays; exact v2 object hierarchy remains `OPEN-002` |
| Files, diffs, diagnostics, terminal, LSP | Partial | `read`, `exec_command`, `write_stdin`, `edit`, and `write` are always-on core execution primitives and are not user-toggleable; `update_plan` and `ask_user` are same-level always-on Host controls rather than execution primitives; diagnostics, interactive terminals, and LSP are optional extensions, while chat may open a contextual tool inspector that hosts an associated isolated extension tool page |
| Worktrees and parallel agent work | Partial | Existing capabilities retained; v2 defaults and lifecycle need approval |
| Skills, extensions, and MCP | Partial | Manifest/API v2 language-neutral extensions replace the retired legacy plugin runtime and provide optional tools, context, MCP adapters, fixed or dynamically announced loopback services, isolated Web views, Host-brokered messaging, and native user-confirmed integrity-bound installation into Aivo-managed local package storage; Host-confirmed conversational MCP registration and single Skill resource registration through the `skills` CLI from an exact skills package/source plus Skill slug, or a complete inline file snapshot, are supported, while legacy plugin compatibility/conversion, Chrome Web Store compatibility, remote extension registries, content scripts, generic model-driven installation, implicit pack/bulk installation, and v2 information architecture remain out of this slice |
| Settings, recovery, diagnostics, data management | Partial | Provider configuration and global validated prompt management are supported; the remaining settings, recovery, diagnostics, and data-management surface is incomplete |
| Versioned v2 HTTP resource contracts | Partial | New contracts use `/api/v2`; v1 method RPC remains a compatibility adapter |
| Explicit persistence migrations | Partial | v1 baseline exists; v2 transition framework is required before schema change |
| Built-in browser UI | Future | Previously removed; decision is `OPEN-005` |
| Cloud sync, collaboration, Aivo accounts | Out | No approved server or multi-user scope |
| Mobile clients | Out | Desktop only |
| Telemetry or remote analytics | Out | No approved collection contract |
| Public source distribution | Required | The recreated GitHub repository is Public and distributes licensor-owned source under `PolyForm-Noncommercial-1.0.0`; this is source-available/noncommercial, not OSI open source. Commercial use requires a separate written agreement. An explicit operator-triggered stable tag publishes the complete target-OS artifact set to immutable R2 version paths and matching GitHub Release assets; release readiness is operator-managed rather than blocked by CI quality gates. |

## Platform scope

- Development and automated core checks may run on any supported development OS.
- Installer, signing, notarization, and OS integration acceptance must run on the target OS.
- A platform is not release-ready until its package and smoke gates in `07-test-release-plan.md` pass.
- Public source and packaged releases must preserve the licensing boundary in `LICENSE` and `LICENSING.md`; package-registry `private: true` metadata prevents accidental registry publication and does not change the source license.

## Scope changes

A scope change requires a Work with `type: feature`, `migration`, `security`, or `governance` as applicable. The Work must update this matrix, the affected Requirement, Traceability, compatibility behavior, and any applicable ADR before implementation.
