# Design QA: sidebarless chat workspace

- Source: `/Users/atlan/.codex/generated_images/019fb8c5-9c57-7641-848c-a39c0a77fd84/exec-8740e147-df79-492e-b9db-d6ddc2de20b9.png`
- Implementation: `/Users/atlan/Documents/Aivo/artifacts/design-qa/workspace-option-2-1536x1024.png`
- Viewport: 1536 × 1024 CSS pixels
- Capture: 1536 × 1024 pixels at 1× density
- State: empty project chat workspace, light appearance
- Full-view comparison: captured; implementation preserves the selected option's sidebarless shell, compact toolbar, central task prompt, four shortcuts, and bottom-centered composer.
- Focused comparison: not completed because the user chose to perform final verification.
- Fonts: semantic Aivo/HIG typography tokens applied; final visual sign-off pending.
- Spacing: full-width canvas and 720px composer implemented; final visual sign-off pending.
- Colors: semantic system tokens applied; final visual sign-off pending.
- Images and icons: existing Hugeicons and project icon assets only; no new raster content required.
- Copy: title, subtitle, four shortcut labels, and existing composer controls retained.
- Interactions: implementation preserves new conversation, history, extensions, settings, project, model, permission, agent mode, attachment, microphone, and submit entry points; manual interaction verification pending.
- Console: not fully reviewed after the user stopped automated QA.
- Findings history: initial wide capture showed the intended hierarchy without a persistent sidebar. Narrow-view, focused-state, overflow, and interaction checks were not completed.
- History popover implementation: conversation-only outlined shadcn `Item` rows; ordinary and project conversations share one recency-sorted list. Associated project names are conditional, and the active row uses `aria-current="page"` with a muted selected background.
- History popover capture: `/Users/atlan/Documents/Aivo/artifacts/design-qa/history-item-list-1536x1024.png` at a 1536 × 1024 viewport. The popover measured 340 × 560 CSS pixels with no horizontal overflow and no console errors or warnings.
- History popover limitation: the browser fixture had zero conversations, so populated two-line rows, conditional project labels, and the selected-row appearance could not be visually signed off there.

Final result: blocked — populated history rows and the remaining user-owned visual and interaction verification are pending. Do not mark `CHG-2026-003-sidebarless-chat-workspace` as Verified or archive it until that evidence is supplied.

---

# Composer reference-layout QA

## Visual truth and capture conditions

- Reference: `/var/folders/6r/04ktqfxx18v8qt0g2tdmx33m0000gn/T/codex-clipboard-57c0691c-5e0f-4b85-b8a4-827997e582ca.png` (1770×386, supplied layout reference).
- Implementation: `apps/desktop/src/features/projects/project-prompt-composer.tsx` and related composer components.
- Wide capture: `artifacts/design-qa/composer-reference-layout-wide-2026-08-07.jpg` at 1280×720, 1× density, new-conversation empty state, light semantic theme.
- Narrow capture: `artifacts/design-qa/composer-reference-layout-narrow-2026-08-07.jpg` at 390×720, 1× density, new-conversation empty state, light semantic theme.
- Focused comparison: `artifacts/design-qa/composer-layout-comparison-2026-08-07.jpg` combines the supplied reference and the wide implementation crop in one image.

## Fidelity review

- Typography: existing Aivo type scale retained; hierarchy matches the reference with quiet context labels, a prominent prompt area, and compact action labels.
- Spacing: context strip, overlapping input card, expanded prompt space, and single bottom row match the reference structure. Wide and narrow states have no visible overlap or clipping.
- Color: Aivo semantic tokens intentionally replace the reference's fixed dark palette, preserving automatic light/dark theme behavior and permission emphasis.
- Radius and borders: the context strip and input card use large nested radii with a subtle semantic ring and shadow, matching the reference silhouette.
- Composition: project/local/available Git context, tool selection, and Agent mode sit above the input; add and permission align left in the bottom row; model, reasoning, voice, and submit align right. Existing Aivo controls beyond the reference remain present by product requirement.

## Interaction and responsive checks

- Project selection opens its searchable popover and keeps the context strip anchored.
- The accessible `选择本次工具` action is visible and enabled in the compact layout; production dialog data still requires the Electron preload bridge and is outside browser-only visual acceptance.
- The Agent mode trigger opens its complete mode menu from the context strip at narrow width.
- At 390 px, secondary labels collapse to icons while project/local context remains readable and the submit action remains reachable.
- After the first optimistic turn, the context strip is absent while the prompt card remains anchored at the bottom with its two-sided action row.
- Started-conversation reference: `/var/folders/6r/04ktqfxx18v8qt0g2tdmx33m0000gn/T/codex-clipboard-b9c5532d-4503-4c53-8b3c-5816fef52fbd.png` (1612×274, supplied active-conversation layout reference).
- Started-conversation wide capture: `artifacts/design-qa/composer-context-hidden-active-wide-2026-08-07.jpg` at 1280×720, 1× density, browser-only optimistic turn, light semantic theme.
- Started-conversation narrow capture: `artifacts/design-qa/composer-context-hidden-active-narrow-2026-08-07.jpg` at 390×720, 1× density, browser-only optimistic turn, light semantic theme.
- Started-conversation comparison: `artifacts/design-qa/composer-active-layout-comparison-2026-08-07.jpg` combines the supplied active-conversation reference and the wide implementation crop in one image.
- The browser-only fallback created no backend conversation data. At both widths the document and viewport widths matched exactly, the context strip was not present in the DOM, and the remaining controls stayed reachable.
- Width correction history: the earlier 960 px composer was a P2 alignment mismatch against the 680 px conversation reading column. The composer frame was changed to the same 680 px maximum, producing identical left edges and widths for the conversation timeline and prompt card.
- A4-width empty capture: `artifacts/design-qa/composer-a4-empty-wide-2026-08-07.jpg` at 1280×720, with the context strip and prompt card both constrained to the 680 px reading column.
- A4-width active capture: `artifacts/design-qa/composer-a4-active-wide-2026-08-07.jpg` at 1280×720; timeline and composer each measured 680 px wide at x=300.
- A4-width narrow capture: `artifacts/design-qa/composer-a4-active-narrow-2026-08-07.jpg` at 390×720; timeline and composer each measured 358 px wide at x=16 with no horizontal overflow.
- A4-width comparison: `artifacts/design-qa/composer-a4-width-comparison-2026-08-07.jpg` combines the supplied active-conversation reference with the corrected browser-rendered implementation. The narrower width is an intentional user-directed override so the prompt card aligns with the A4-like conversation column.
- Height correction history: the earlier 160 px empty/single-line card was a P2 density mismatch against the supplied compact Codex reference. Reducing the textarea minimum from 88 px/two rows to 32 px/one row produces a 104 px card while preserving content-driven growth up to the existing 300 px limit.
- Compact-height reference: `/var/folders/6r/04ktqfxx18v8qt0g2tdmx33m0000gn/T/codex-clipboard-bdf90f53-eeae-4e58-9194-06c927b39efd.png` (3218×2118, supplied desktop reference).
- Compact-height empty capture: `artifacts/design-qa/composer-compact-height-empty-wide-2026-08-07.jpg` at 1280×720; card measured 104 px and the textarea 32 px.
- Compact-height active capture: `artifacts/design-qa/composer-compact-height-active-wide-2026-08-07.jpg` at 1280×720; card remained 104 px after the context strip unmounted.
- Compact-height narrow capture: `artifacts/design-qa/composer-compact-height-active-narrow-2026-08-07.jpg` at 390×720; card remained 104 px by 358 px with no horizontal overflow.
- Compact-height comparison: `artifacts/design-qa/composer-compact-height-comparison-2026-08-07.jpg` combines the normalized reference composer crop and the corrected implementation crop. Typography, semantic colors, existing icon assets, and copy remain unchanged; only vertical density was corrected.
- Multiline interaction: six lines expanded the textarea to 126 px and the card to 198 px, confirming the compact empty state does not remove automatic growth.
- No P0, P1, or P2 visual defects remain in the reviewed empty and started-conversation composer states.

final result: passed

---

# Composer file-card QA

- Source visual truth: `/var/folders/6r/04ktqfxx18v8qt0g2tdmx33m0000gn/T/codex-clipboard-ec1aeaaa-9a2a-47de-b7d6-9ecab3cb09c2.png` (1338 × 258 pixels).
- Implementation: `apps/desktop/src/features/projects/project-prompt-attachments.tsx`.
- Intended viewport/state: desktop composer with two text-file attachments, semantic light/dark theme, 1× CSS density.
- Full-view comparison: not completed because the user chose to perform visual verification.
- Focused comparison: not completed; no browser-rendered implementation screenshot was retained.
- Fonts and typography: implementation uses the existing Aivo type system, a single-line filename, and a compact uppercase extension label.
- Spacing and layout: implementation uses a horizontal 224 × 56 CSS-pixel file card with a 40-pixel icon tile and a separate 20-pixel remove control.
- Colors and tokens: semantic Aivo background, muted, border, and foreground tokens replace the reference's fixed dark values.
- Image and icon fidelity: existing Hugeicons `File02Icon` and `Cancel01Icon` are used; ordinary files do not reuse the image-preview treatment.
- Copy and content: filenames truncate to one line; the second line shows `MD`, `TOML`, or another compact file-type label.
- Interaction: remove behavior remains wired to the existing attachment removal action; user-owned verification is pending.
- Comparison history: no P0/P1/P2 comparison iteration was run after the user took ownership of verification.

final result: blocked
