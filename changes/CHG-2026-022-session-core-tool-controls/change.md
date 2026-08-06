# Allow session-scoped control of Aivo core tools

## Problem or goal

The tool activation dialog currently hides Aivo's four core coding tools and shows a separate MCP tab. Users need to turn eligible tools, including the core tools, on or off without using this dialog to manage MCP sources.

## Expected behavior

`REQ-TOOL-001` retains `read`, `bash`, `edit`, and `write` as the four default coding tools. They are shown in the tool tab and default to on; a user can disable any subset for the current conversation, or one pending new conversation, so the disabled tool is omitted from that Agent request's model-visible tool surface. The Extensions & MCP settings tool tab lists Aivo-owned built-in tools, including built-in extension tools such as project management. User-installed extension tools, MCP adapters, and MCP-management tools remain in their owning surfaces and are not duplicated there. Existing MCP session activations remain unchanged when the visible selection is saved. `REQ-SESSION-001` session isolation and one-shot pending-draft behavior remain unchanged.

## Non-goals

No MCP installation, enablement, trust, connection, or execution change; no global default; no new RPC endpoint, schema migration, permission policy, or extension lifecycle change.

## Impact

The React dialog filters MCP entries and includes built-in entries. Core session preferences use existing execution-state metadata and the Agent assembly omits explicitly disabled core specs. Electron main/preload, database schema, HTTP/RPC shape, providers, credentials, extension/MCP trust, dependencies, packaging, and platform scope are unchanged.

## Implementation constraints

Missing preference metadata means all four core tools remain enabled for compatibility. A save must preserve hidden MCP names, reject bridge names, and make disabled core tools unavailable before the Provider request and Tool Snapshot are assembled. Repeated saves are idempotent; session cleanup retains existing ownership. Failure, cancellation, timeout, persistence rollback, and authorization semantics are unchanged or N/A because the existing session metadata path is reused.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `CORE-TOOLS-DOC-001` | `REQ-TOOL-001`, `REQ-SESSION-001` | Requirement, scope, traceability, and Work agree | `AT-TOOL-001`, `AT-SESSION-001` | Completed |
| `CORE-TOOLS-STATE-001` | `REQ-TOOL-001`, `REQ-SESSION-001` | Session preference preserves defaults and omits selected core tools | `AT-TOOL-001`, `AT-SESSION-001` | Completed |
| `CORE-TOOLS-UI-001` | `NFR-UI-001` | Tool tab shows built-in/extension controls and no MCP controls | `AT-UI-001` | Completed |
| `CORE-TOOLS-QA-001` | `AT-TOOL-001`, `AT-SESSION-001`, `AT-UI-001` | Focused tests, core tests, docs, lint, and build evidence | `AT-TOOL-001`, `AT-SESSION-001`, `AT-UI-001` | Pending |

## Acceptance and evidence

- A new or legacy session with no core preference exposes all four core tools in stable order.
- Disabling a core tool removes it from the next Provider request and Tool Snapshot for that session; re-enabling restores it.
- The Extensions & MCP settings tool tab renders only Aivo built-in entries with working switches, and its count is derived from that same unfiltered built-in list. Saving a visible selection preserves hidden extension and MCP activation state.
- Empty, loading, error, repeat-save, long labels, keyboard switch activation, narrow layout, cancellation, timeout, persistence rollback, security, and platform-package effects are covered by existing behavior or are N/A as appropriate. Command evidence will be recorded before verification.

Implementation evidence recorded on 2026-08-06: the session active-tools response now returns the separately selected core names while retaining the existing manual/deferred tool list. Missing metadata reports all four core tools; a save stores only the disabled core subset in existing session execution metadata. Agent assembly applies that subset before the Provider request and immutable Tool Snapshot. The renderer merges core names into the active selection, lists enabled built-in and extension tools, and excludes MCP and bridge entries; the MCP tab is not mounted. Automated evidence passed: focused Go activation/assembly and MCP-isolation tests, `node --test --experimental-strip-types apps/desktop/tests/project-tool-activation-model.test.ts`, `pnpm test:core`, `pnpm lint`, `pnpm build`, `pnpm docs:check`, and `git diff --check`. Lint retained only pre-existing Fast Refresh warnings; build retained existing large-barrel/chunk-size advisories. Wide/narrow interactive visual acceptance remains pending, so this Work stays `Implementing`.

## Security and data lifecycle

Only selected core tool names are represented in session execution metadata. No prompt, tool payload, result, credential, MCP configuration, or private filesystem content is added to renderer state, logs, or persistence.

## Compatibility and migration

This is additive session metadata. Existing sessions lack that metadata and therefore retain the default four core tools. Rollback ignores the preference and safely restores the existing default surface.

## Bug root cause (type=bug only)

N/A.
