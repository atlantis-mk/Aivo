# Allow session-scoped control of Aivo core tools

> Current direction: `CHG-2026-056-always-on-hidden-core-tools` replaces this unfinished behavior. The four required primitives are always active and omitted from management and selection surfaces; do not advance this Work's earlier core-tool toggle acceptance criteria.

## Problem or goal

The tool activation dialog currently hides Aivo's four core coding tools and shows a separate MCP tab. Users need to turn eligible tools, including the core tools, on or off without using this dialog to manage MCP sources.

## Expected behavior

`REQ-TOOL-001` retains `read`, `bash`, `edit`, and `write` as the four default coding tools. They are shown in the tool tab and default to on; a user can disable any subset for the current conversation, or one pending new conversation, so the disabled tool is omitted from that Agent request's model-visible tool surface. The activation dialog mirrors the global Extensions & MCP settings drawer's `扩展`, `MCP`, `技能`, and `工具` classifications and shows only entries made eligible by their global enablement state. Switches in the activation dialog change only the current conversation, or the one pending new conversation; they never enable, disable, install, trust, or configure the global source. Hidden globally disabled entries retain any remembered conversation selection without becoming eligible. `REQ-SESSION-001` session isolation and one-shot pending-draft behavior remain unchanged.

## Non-goals

No MCP installation, enablement, trust, connection, or execution change; no global default; no new RPC endpoint, schema migration, permission policy, or extension lifecycle change.

## Impact

The React dialog filters the existing global catalogs by enablement, presents the same four source classifications, and applies its switches through existing session-scoped tool and Skill activation APIs. Core session preferences use existing execution-state metadata and the Agent assembly omits explicitly disabled core specs. Electron main/preload, database schema, HTTP/RPC shape, providers, credentials, extension/MCP trust, dependencies, packaging, and platform scope are unchanged.

## Implementation constraints

Missing preference metadata means all four core tools remain enabled for compatibility. Global source switches only control catalog eligibility and dialog visibility; dialog switches only control the selected conversation. A save must preserve hidden globally disabled names, reject bridge names, and make disabled core tools unavailable before the Provider request and Tool Snapshot are assembled. Repeated saves are idempotent; session cleanup retains existing ownership. Failure, cancellation, timeout, persistence rollback, and authorization semantics are unchanged or N/A because the existing session metadata path is reused.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `CORE-TOOLS-DOC-001` | `REQ-TOOL-001`, `REQ-SESSION-001` | Requirement, scope, traceability, and Work agree | `AT-TOOL-001`, `AT-SESSION-001` | Completed |
| `CORE-TOOLS-STATE-001` | `REQ-TOOL-001`, `REQ-SESSION-001` | Session preference preserves defaults and omits selected core tools | `AT-TOOL-001`, `AT-SESSION-001` | Completed |
| `CORE-TOOLS-UI-001` | `NFR-UI-001` | Dialog mirrors all global catalog classifications and eligibility while keeping activation session-scoped | `AT-UI-001` | Implementing |
| `CORE-TOOLS-QA-001` | `AT-TOOL-001`, `AT-SESSION-001`, `AT-UI-001` | Focused tests, core tests, docs, lint, and build evidence | `AT-TOOL-001`, `AT-SESSION-001`, `AT-UI-001` | Pending |

## Acceptance and evidence

- A new or legacy session with no core preference exposes all four core tools in stable order.
- Disabling a core tool removes it from the next Provider request and Tool Snapshot for that session; re-enabling restores it.
- The pending/new-conversation composer exposes a `工具` button that opens the existing tool-selection dialog. Project, local runtime, the available Git branch, tool selection, and Agent mode occupy a context strip above the large rounded input panel until the first turn starts. Once a conversation is active, that strip is not mounted; the prompt card remains with add/permission on the left and model/voice/submit on the right of its bottom row.
- The activation dialog mirrors the global `扩展`, `MCP`, `技能`, and `工具` classifications, shows only globally enabled entries, and keeps every dialog switch scoped to the selected conversation or one-shot pending conversation. Saving preserves hidden globally disabled activation state.
- Empty, loading, error, repeat-save, long labels, keyboard switch activation, narrow layout, cancellation, timeout, persistence rollback, security, and platform-package effects are covered by existing behavior or are N/A as appropriate. Command evidence will be recorded before verification.

Implementation evidence recorded on 2026-08-06: the session active-tools response now returns the separately selected core names while retaining the existing manual/deferred tool list. Missing metadata reports all four core tools; a save stores only the disabled core subset in existing session execution metadata. Agent assembly applies that subset before the Provider request and immutable Tool Snapshot. The renderer merges core names into the active selection, lists enabled built-in and extension tools, and excludes MCP and bridge entries; the MCP tab is not mounted. Automated evidence passed: focused Go activation/assembly and MCP-isolation tests, `node --test --experimental-strip-types apps/desktop/tests/project-tool-activation-model.test.ts`, `pnpm test:core`, `pnpm lint`, `pnpm build`, `pnpm docs:check`, and `git diff --check`. Lint retained only pre-existing Fast Refresh warnings; build retained existing large-barrel/chunk-size advisories. Wide/narrow interactive visual acceptance remains pending, so this Work stays `Implementing`.

Additional renderer evidence recorded on 2026-08-07: after removal of the standalone environment-information popover, its tool-dialog trigger was restored as an accessible `选择本次工具` button in the composer. The composer toolbar now uses two rows: add/permission sit at the upper left, model/voice/submit at the upper right, agent/tool at the lower left, and project selection at the lower right. Browser interaction confirmed the tool button invokes the dialog-open path; the browser-only preview then reaches the expected missing-Electron-preload boundary while the production Electron path retains its typed bridge. Visual checks at 1280×720, 760×720, and 390×720 confirmed the requested row order and two-sided alignment, visible compact tool entry, and no horizontal overflow. Captures are stored at `artifacts/design-qa/composer-tools-row-bottom-wide-2026-08-07.jpg` and `artifacts/design-qa/composer-tools-row-bottom-narrow-2026-08-07.jpg`. `pnpm lint` and `pnpm build` passed with only the existing warnings; broader Electron dialog acceptance remains pending, so `CORE-TOOLS-QA-001` and this Work remain `Implementing`.

Catalog-scope correction recorded on 2026-08-07: the activation dialog now mounts `扩展`, `MCP`, `技能`, and `工具`, filters tool entries by the global catalog's enabled registrations, filters Skills by their global enabled state, and uses source metadata only for grouping and display. Dialog switches continue to save through the existing session-scoped tool and Skill APIs, so they cannot mutate global extension, MCP, or Skill enablement. `pnpm lint` and `pnpm build` passed with only the repository's existing warnings; focused model coverage now includes globally disabled exclusion and MCP classification.

The final 2026-08-07 composer layout supersedes that intermediate two-row arrangement. It uses the conversation timeline's 680 px A4-like reading width, places real project/local/Git context plus tool selection and Agent mode in a rounded strip behind the input card, keeps the empty input compact while allowing content-driven growth, and leaves add/permission opposite model/voice/submit in a single bottom row with compact behavior below 640 px. Reference-to-implementation comparison and 1280×720/390×720 captures are stored at `artifacts/design-qa/composer-layout-comparison-2026-08-07.jpg`, `artifacts/design-qa/composer-reference-layout-wide-2026-08-07.jpg`, and `artifacts/design-qa/composer-reference-layout-narrow-2026-08-07.jpg`. Project selection and the Agent mode menu opened correctly in the browser preview, the tool entry remained visible and enabled, and no horizontal overflow or control overlap was observed. `pnpm lint`, `pnpm build`, and `git diff --check` passed with only the repository's existing warnings; broader Electron dialog acceptance remains pending, so `CORE-TOOLS-QA-001` and this Work remain `Implementing`.

Started-conversation visual evidence recorded on 2026-08-07: the context strip now follows the existing new-conversation visibility state and unmounts as soon as an optimistic first turn starts or an existing conversation is open. A browser-only optimistic turn verified the transition without creating backend data; project, local, Git, tool, and Agent controls disappeared while the 680 px prompt card and its two-sided bottom action row remained available. The 1280×720 and 390×720 captures at `artifacts/design-qa/composer-context-hidden-active-wide-2026-08-07.jpg` and `artifacts/design-qa/composer-context-hidden-active-narrow-2026-08-07.jpg` show no horizontal overflow or control overlap. The source-to-implementation comparison is stored at `artifacts/design-qa/composer-active-layout-comparison-2026-08-07.jpg`. `pnpm lint`, `pnpm build`, and `git diff --check` passed with only the repository's existing warnings; broader Electron dialog acceptance remains pending, so `CORE-TOOLS-QA-001` and this Work remain `Implementing`.

A4-width correction recorded on 2026-08-07: the composer frame now shares the conversation timeline's exact 680 px maximum width instead of the earlier 960 px maximum. Browser measurements at 1280×720 showed both timeline and composer at 680 px with the same 300 px left edge; at 390×720 both measured 358 px with a 16 px left edge. Empty, active, and narrow captures are stored at `artifacts/design-qa/composer-a4-empty-wide-2026-08-07.jpg`, `artifacts/design-qa/composer-a4-active-wide-2026-08-07.jpg`, and `artifacts/design-qa/composer-a4-active-narrow-2026-08-07.jpg`. No horizontal overflow or console error was observed. `pnpm lint`, `pnpm build`, `pnpm docs:check`, and `git diff --check` passed with only the repository's existing warnings; broader Electron dialog acceptance remains pending, so `CORE-TOOLS-QA-001` and this Work remain `Implementing`.

Compact-height correction recorded on 2026-08-07: the empty and single-line input card now measures 104 px high instead of the earlier 160 px, matching the supplied compact Codex reference more closely. The textarea starts at one 32 px row and still expands from content; a six-line browser check grew the card to 198 px with a 126 px textarea. Empty, active, and narrow captures are stored at `artifacts/design-qa/composer-compact-height-empty-wide-2026-08-07.jpg`, `artifacts/design-qa/composer-compact-height-active-wide-2026-08-07.jpg`, and `artifacts/design-qa/composer-compact-height-active-narrow-2026-08-07.jpg`. The focused source comparison is stored at `artifacts/design-qa/composer-compact-height-comparison-2026-08-07.jpg`. No horizontal overflow or console error was observed. `pnpm lint`, `pnpm build`, `pnpm docs:check`, and `git diff --check` passed with only the repository's existing warnings; broader Electron dialog acceptance remains pending, so `CORE-TOOLS-QA-001` and this Work remain `Implementing`.

## Security and data lifecycle

Only selected tool names, disabled core tool names, and selected Skill IDs use the existing session execution state. No prompt, tool payload, result, credential, MCP configuration, or private filesystem content is added to renderer state, logs, or persistence.

## Compatibility and migration

This is additive session metadata. Existing sessions lack that metadata and therefore retain the default four core tools. Rollback ignores the preference and safely restores the existing default surface.

## Bug root cause (type=bug only)

N/A.
