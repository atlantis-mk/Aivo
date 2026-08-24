# Adopt an HIG-aligned desktop theme scale

## Problem or goal

Aivo currently mixes marketing-scale headings, ordinary web defaults, and compact desktop controls without a stable global scale. Establish a coherent desktop theme grounded in Apple HIG logical dimensions while preserving Aivo's cross-platform Electron scope and monochrome identity.

## Expected behavior

`NFR-UI-002` defines a shared desktop visual scale. The renderer provides semantic typography for the macOS reference styles (26/32 large title, 22/26 title 1, 17/22 title 2, 15/20 title 3, 13/16 body and headline, 11/14 subheadline, and 10/13 footnote/caption), a 28 px regular control target with 24 px small and 32 px large variants, and a restrained 4/8/12/16/24/32 spacing rhythm. These logical CSS dimensions are a cross-platform density baseline; they do not claim native AppKit rendering or platform certification.

## Non-goals

No Liquid Glass recreation, macOS-only behavior, brand-color redesign, native AppKit migration, Electron/core contract change, or modification of generated files or shared primitive source under `apps/desktop/src/components/ui`.

## Impact

Renderer theme tokens, base typography, and feature compositions are affected. Electron main/preload, Go domain/application/persistence/transport, public API/RPC/IPC, schema/data, providers, skills/plugins/MCP, LSP, terminals/processes, worktrees, security, dependencies, and release packaging behavior are unaffected. UI acceptance, lint, and build gates apply.

## Implementation constraints

`apps/desktop/src/index.css` owns the semantic theme tokens. Feature code consumes the scale without changing shared primitive source. Existing semantic colors and dark mode remain intact. Wide and narrow layouts must preserve keyboard focus, overflow, and readable minimum sizes. The global scale must remain usable on macOS, Windows, and Linux even though Apple HIG supplies the logical reference values.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `THEME-DOC-001` | `NFR-UI-002` | Primary Requirement and traceability row | `AT-UI-002` | Completed |
| `THEME-CSS-001` | `NFR-UI-002` | Global typography, spacing, radius, and control tokens | `AT-UI-002` | Completed |
| `THEME-SETUP-001` | `NFR-UI-002` | Welcome/setup composition uses the semantic scale | `AT-UI-002` | Completed |
| `THEME-QA-001` | `NFR-UI-002` | Wide/narrow screenshots, interaction check, lint/build evidence | `AT-UI-002` | Pending |

## Acceptance and evidence

- Theme values match the recorded logical scale and retain semantic light/dark colors.
- The setup welcome page and representative project/settings surfaces remain readable without horizontal overflow at wide and narrow desktop sizes.
- Primary controls retain visible focus states and at least the declared size for their variant.
- `pnpm docs:check`, `pnpm lint`, and `pnpm build` pass without new warnings.
- Failure, cancellation, repetition, timeout, cleanup, persistence, migration, rollback, and provider behavior are N/A because this is renderer-only presentation.

Implementation evidence: `pnpm docs:check`, `pnpm lint`, and `pnpm build` passed on 2026-07-31. Lint retained only the repository's existing Fast Refresh warnings, and build retained its existing large-chunk advisory. The setup route loaded successfully after the theme change. Per user direction, final wide/narrow visual and interaction acceptance remains pending for user verification; this Work therefore remains `Implementing` and must not be archived yet. After that evidence is accepted and the Work moves to `Verified`, run `pnpm work:archive -- CHG-2026-002-desktop-hig-theme`.

## Security and data lifecycle

No secret, private data, DTO, persistence, logging, diagnostic, clipboard, crash, or backup behavior changes.

## Compatibility and migration

No schema, data, API, RPC, IPC, setting, dependency, or irreversible migration. Rollback is the CSS/token and feature-composition revert.

## Bug root cause (type=bug only)

N/A.
