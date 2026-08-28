# ADR-0028: Register pi-style file discovery tools as optional built-ins

- Status: Accepted
- Date: 2026-08-26
- Related Work: `CHG-2026-054-pi-style-file-tool-names`
- Supersedes in part: `ADR-0002`
- Closes OPEN: none

## Context

The four default primitives keep the primary coding surface small, but Aivo already owns bounded implementations for directory listing, path matching, and content search. Leaving those implementations unregistered forces every conversation through Bash even when a user wants explicit globally manageable tools or a conversation needs safe auxiliary activation. Their development names also differ from the established `pi` tool vocabulary requested for Aivo.

## Decision

- Aivo MUST keep `read`, `bash`, `edit`, and `write` as the only default coding primitives.
- Core MUST additionally register the optional built-in canonical tools `grep`, `find`, and `ls` in that order after the default primitives.
- The optional tools MUST be visible in global tool management, MUST support per-conversation manual activation, and MUST participate as standalone candidates in the existing auxiliary selection flow.
- Global visibility, mode eligibility, immutable Tool Snapshots, permissions, cancellation, bounds, and the active Execution Environment MUST apply exactly as they do to other local tools.
- The removed development identities `search_files`, `glob`, and `list_files` MUST NOT remain registered or execute through aliases. Historical records MAY retain display-only rendering.

## Rationale

- Matching the concise `pi` names reduces vocabulary differences without expanding the default Provider tool surface.
- Registration lets existing Host-owned global, manual, and auxiliary selection controls govern the tools consistently.
- Refusing execution aliases preserves one canonical identity across persistence, Provider declarations, snapshots, permissions, and execution.

## Consequences

- Existing development-only selections or global preference rows under the removed names no longer affect the renamed registrations.
- Alternate Execution Environments must implement the three optional operations when selected; Aivo does not fall back to local files after an environment has been chosen.
- Historical UI rendering continues to recognize the removed names without making them executable.

## Rejected alternatives

- Keep only Bash access: this prevents the requested global, manual, and auxiliary controls from addressing these capabilities directly.
- Make all seven tools default: this expands every Provider request and contradicts the four-primitive default boundary.
- Preserve old execution aliases: this creates multiple authoritative identities and weakens snapshot and preference consistency.

## Verification

`AT-TOOL-001` verifies exact names, stable default assembly, execution behavior, no aliases, and environment ownership. `AT-SESSION-001`, `AT-WORKSPACE-001`, and `AT-EXTENSION-001` verify global visibility, conversation activation, auxiliary eligibility, and desktop presentation.
