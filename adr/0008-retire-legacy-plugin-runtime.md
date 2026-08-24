# ADR-0008: Retire the legacy plugin runtime in favor of Manifest v2 extensions

- Status: Accepted
- Date: 2026-08-06
- Related Work: `CHG-2026-021-retire-legacy-plugins`
- Closes OPEN: none

## Context

Aivo has a legacy plugin contract based on `aivo.plugin.json`, a supervised JSONL subprocess, plugin-specific RPCs, and plugin-owned tools, hooks, MCP declarations, and provider contributions. ADR-0002 introduced a broader language-neutral extension boundary, and Manifest/API v2 now supports built-in, process, service, external, and static runtimes plus explicit trust, integrity-bound installation, tools, policies, contexts, Views, messaging, and lifecycle recovery. Keeping both systems duplicates authority and makes it unclear which contract owns optional capability.

## Decision

- Manifest/API v2 extensions MUST be the only Aivo package and extension runtime contract.
- Aivo MUST NOT discover, install, list, enable, reload, start, or invoke legacy plugin packages.
- Legacy plugin tools, hooks, provider contributions, and embedded MCP declarations MUST NOT enter current catalogs, Tool Snapshots, Provider registries, or model context.
- Legacy plugin RPC methods MUST be removed rather than retained as aliases to extension operations.
- Aivo MUST NOT automatically convert or trust a legacy plugin as a Manifest v2 extension.
- Existing legacy plugin persistence rows MUST remain inert and untouched until a separately approved schema migration defines removal; user-owned plugin source directories MUST NOT be deleted.
- Build-system, Markdown-renderer, router, lint, and other third-party concepts named "plugin" are outside this product decision.

## Rationale

- One runtime contract gives users and maintainers one trust, lifecycle, naming, UI, and recovery model.
- Manifest v2 covers the valid use cases of the narrower JSONL plugin implementation without coupling packages to one process protocol.
- Removing aliases makes unsupported packages fail clearly instead of receiving partial or misleading compatibility.
- Preserving rows and source folders avoids an irreversible migration solely to remove an executable path.

## Consequences

- Legacy plugin packages and RPC clients are incompatible with the new development build and require explicit author migration to Manifest v2.
- The plugin management tabs, add flow, runtime manager, process protocol, provider bridge, and hook path are deleted.
- Historical plugin data may remain in SQLite but has no current behavior or UI.
- Extension, MCP, Skill, and tool management remain supported and must retain independent failure handling.

## Rejected alternatives

- Keep both runtimes: preserves duplicate trust and lifecycle boundaries indefinitely.
- Translate legacy manifests automatically: cannot safely infer Manifest v2 permissions, integrity, services, Views, or explicit native trust.
- Drop plugin tables immediately: requires a destructive schema transition without improving the runtime retirement outcome.
- Keep deprecated RPC aliases: risks callers assuming semantic compatibility that does not exist.

## Verification

`AT-EXTENSION-001` proves only core, Manifest v2 extension, Skill, and MCP sources enter current catalogs and snapshots. `AT-WORKSPACE-001` and `AT-UI-001` prove the replacement management and navigation surfaces. `CT-SECURITY-001` and `CT-RELIABILITY-001` prove legacy rows are inert, no legacy process starts, and supported extension/MCP lifecycle behavior remains bounded and deterministic.
