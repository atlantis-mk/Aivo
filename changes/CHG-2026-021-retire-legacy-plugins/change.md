# Retire the legacy plugin runtime

## Problem or goal

Aivo currently has two overlapping local capability systems: the legacy `aivo.plugin.json` JSONL process runtime and the language-neutral Manifest/API v2 extension runtime. Manifest v2 now owns installation, trust, lifecycle, tools, policies, services, Views, and recovery, so keeping the legacy plugin feature creates duplicate UI, RPC, process, catalog, provider, hook, and security paths. The user explicitly approved retiring plugins in favor of extensions on 2026-08-06.

## Expected behavior

`REQ-EXTENSION-001` makes Manifest v2 extensions the only Aivo package/runtime abstraction. The desktop exposes Extensions, MCP, Skills, and Tools without plugin or application-plugin tabs. Core does not discover, install, list, enable, reload, start, call, or inject legacy plugins, and does not register their tools, hooks, MCP declarations, or provider contributions. Legacy plugin RPC methods are unsupported.

## Non-goals

This Work does not remove Manifest v2 extensions, MCP, Skills, package-manager build plugins, legacy historical documentation, or existing SQLite plugin rows. It does not convert legacy plugin packages or silently install them as extensions.

## Impact

Renderer navigation, settings, activation grouping, and desktop service bindings lose plugin-specific behavior. Go domain, application, transport, and lifecycle code lose the legacy plugin manager, manifest loader, JSONL process client, runtime tool adapter, hooks, provider contributions, and RPC cases. Existing plugin persistence rows remain inert so rollback does not require destructive schema work. No provider, secret, MCP, extension, LSP, terminal, worktree, or production dependency is added.

## Implementation constraints

ADR-0008 owns the retirement boundary. Existing plugin rows must never be read or executed by the new runtime. Extension and MCP catalogs must continue to prepare independently when another source fails. Public plugin RPCs fail as unsupported through normal method dispatch. Shared diagnostics storage may retain its historical table name, but current DTOs and UI must not expose a plugin feature. User-owned plugin directories and stored rows are never deleted.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `PLUGIN-RETIRE-DOC-001` | `REQ-EXTENSION-001` | Scope, Requirement, ADR, test plan, and Traceability name extensions as the replacement | `AT-EXTENSION-001` | Completed |
| `PLUGIN-RETIRE-CORE-001` | `REQ-EXTENSION-001`, `NFR-RELIABILITY-001` | No legacy plugin RPC, manager, process, hook, provider, or catalog path remains active | `AT-EXTENSION-001`, `CT-RELIABILITY-001` | Completed |
| `PLUGIN-RETIRE-DESKTOP-001` | `REQ-WORKSPACE-001`, `NFR-UI-001` | Extension management replaces plugin navigation, tabs, add flow, and activation grouping | `AT-WORKSPACE-001`, `AT-UI-001` | Completed |
| `PLUGIN-RETIRE-COMPAT-001` | `NFR-SECURITY-001`, `NFR-RELIABILITY-001` | Existing plugin rows remain inert and user directories remain untouched | `CT-SECURITY-001`, `CT-RELIABILITY-001` | Completed |
| `PLUGIN-RETIRE-QA-001` | `NFR-RELIABILITY-001` | Focused regression tests and repository gates | `AT-EXTENSION-001`, `CT-RELIABILITY-001` | Completed |

## Acceptance and evidence

- Desktop navigation and management surfaces contain no legacy Plugin or application-plugin feature; Extensions, MCP, Skills, and Tools remain reachable.
- `ListPlugins`, `InstallPluginFromPath`, `SetPluginEnabled`, and `ReloadPlugins` are no longer dispatched, and no legacy plugin process can start during service construction, catalog preparation, model requests, tool execution, or shutdown.
- Legacy plugin tools, hooks, MCP declarations, and provider contributions cannot enter a registry, Tool Snapshot, Provider registry, or model request.
- Manifest v2 extension installation, restart restoration, enable/disable/uninstall, tool catalog, hooks, Views, and runtime messaging continue to work.
- Existing `plugin_installs` rows and user-owned source folders are not mutated or deleted; downgrade compatibility remains a data-preservation concern rather than an active feature.
- Applicable gates are `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, `pnpm build`, focused desktop tests, and `git diff --check`.

Verification evidence on 2026-08-06:

- `pnpm docs:check` passed with 53 Markdown files, 22 YAML files, 18 Requirements, 18 Test IDs, 8 ADRs, 21 Work Packages, and 13 previously archived Work Packages.
- `pnpm scripts:test` passed all archive, desktop model, extension installation/runtime, tool-view, permission, activation-scope, and example extension tests.
- `pnpm test:core` passed every Go package, including the legacy-row inertness and unsupported legacy RPC regression tests.
- `pnpm lint` passed with only the repository's existing Fast Refresh export warnings in shared UI and root route files.
- `pnpm build` passed TypeScript project compilation and the Vite production build; the generated route output contains `projects.extensions` and no `projects.plugins` route.
- `git diff --check` passed.
- In-app browser QA at 1280 x 800 and 640 x 800 showed no horizontal overflow. The settings surface exposed only Extensions, MCP, Skills, and Tools, with the sample Manifest v2 extension rendered correctly and no legacy Plugin or application-plugin tab.

## Security and data lifecycle

Removing the legacy subprocess runtime reduces executable trust surface. No secret or private path is migrated. Existing local plugin records remain in the user's database but are not queried or surfaced, and source directories remain user-owned. Extension and MCP credential, logging, cancellation, and teardown boundaries are unchanged.

## Compatibility and migration

The removal is an intentional development-contract break: legacy plugin packages and RPC callers stop working and must move to Manifest/API v2 extensions. There is no automatic conversion. No schema version changes and no rows or files are deleted; an older binary can still read its prior plugin records after downgrade.

## Bug root cause (type=bug only)

N/A.
