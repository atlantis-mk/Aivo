# Conversation timeline visual QA

## Comparison target

- Source visual truth: `/var/folders/6r/04ktqfxx18v8qt0g2tdmx33m0000gn/T/codex-clipboard-2fdecc28-cc07-4aac-82d3-485cf13df5a3.png`, `/var/folders/6r/04ktqfxx18v8qt0g2tdmx33m0000gn/T/codex-clipboard-00e31859-1aa2-4120-b9eb-95216538b6f3.png`, `/var/folders/6r/04ktqfxx18v8qt0g2tdmx33m0000gn/T/codex-clipboard-5f8ddbcb-e339-4b2c-a5ad-306739907ce9.png`, `/var/folders/6r/04ktqfxx18v8qt0g2tdmx33m0000gn/T/codex-clipboard-9d72121d-c885-4f8b-add1-95e933534538.png`, and `/var/folders/6r/04ktqfxx18v8qt0g2tdmx33m0000gn/T/codex-clipboard-73cd1c04-2076-4de8-b79e-745b95026fb4.png`.
- Source pixels: `3358 × 2110`, `3840 × 2110`, `1538 × 504`, and `1538 × 114`; the target is the compact grouped tool-turn treatment, the light-grey Shell result block, and a left-aligned status with one divider below it, excluding every floating composer visible in the captures.
- Implementation screenshot: unavailable. The available capture surface exposes browsers only; it does not expose the running Electron window.
- Viewport, CSS size, and density normalization: unavailable without an Electron capture.
- Intended state: a completed conversation initially shows only a clickable left-aligned `用时` status, one divider, and the final response. Its tool-call rows remain hidden until the status is expanded; expanded tools retain the compact operation list and selected-command `Shell` block.

## Evidence

- `pnpm --filter @aivo/desktop lint` passed.
- `pnpm --filter @aivo/desktop build` passed.
- The previous desktop development runtime started with the existing debug host; its host-only build completed in `0.75s`.
- Full-view and focused-region visual comparisons are unavailable because an Electron screenshot cannot be captured in this environment.

**Findings**

- [P1] Visual comparison is blocked.
  Location: desktop conversation timeline.
  Evidence: the target screenshot is available, but the launched Electron window is not exposed to the available browser or native-app capture surface.
  Impact: exact typography, wrapping, rhythm, and token fidelity cannot be confirmed from a rendered implementation.
  Fix: capture the desktop timeline at the source state and compare its collapsed tool summary plus expanded multi-tool list side-by-side with the target.

**Open Questions**

- None. The supplied screenshot is an unambiguous target; only implementation capture is missing.

**Implementation Checklist**

1. Open a completed conversation containing several consecutive tools in the desktop app.
2. Capture its collapsed and expanded tool group without the composer overlay.
3. Compare both states against the source at the same scale, then correct any remaining P0–P2 drift.

**Follow-up Polish**

- None until the rendered comparison is available.

## Final result

blocked
