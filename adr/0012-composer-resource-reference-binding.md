# ADR-0012: Validate composer resource references in Core

- Status: Accepted
- Date: 2026-08-07
- Related Work: `CHG-2026-026-composer-resource-mentions`
- Closes OPEN: none

## Context

Visible `@name` text cannot safely identify a project, Skill, built-in tool, extension, or MCP source because names can collide, catalog state can change after rendering, and the renderer does not own project association or capability eligibility. The conversation submit boundary must preserve explicit user selection without letting renderer-authored identifiers bypass immutable project ownership, source enablement, global tool policy, or Tool Snapshot assembly.

## Decision

- `SubmitSessionMessage` MUST accept an optional bounded list of typed resource references containing a kind and stable ID; project references additionally carry the exact displayed root path for ID/path consistency validation.
- Core MUST treat the renderer as untrusted, deduplicate at most 32 references, and validate all references against current authoritative project, Skill, extension, MCP, and globally eligible tool state before appending the user event or starting a turn.
- Plain prompt text MUST NOT create a resource binding. Unknown, blank, stale, disabled, mismatched, conflicting, or over-limit references MUST fail closed.
- An exact project reference MUST use the existing immutable current-session association contract. A new conversation MAY be created at the selected root; an unscoped conversation MAY bind once; a different bound project MUST NOT be switched or detached.
- Skill and tool references MUST merge with existing activation for only the target conversation. Extension and MCP references MUST remain source-level at the contract/UI boundary and MAY expand only to that source's currently registered, globally eligible canonical tools in Core. No reference may install, import, trust, globally enable, bind credentials, authorize, or execute a resource.
- The normal Host-owned conversation context and Tool Snapshot paths MUST attach validated Skill instructions, project context, and tools to Provider requests. Core MAY persist canonical non-secret reference summaries on the user event, but MUST NOT persist renderer display text, credentials, schemas, MCP prompt/resource bodies, or local paths in that payload.
- Resource references MUST be accepted only for immediate submission; a queued/steered input carrying them MUST be rejected rather than silently dropping executable intent.
- The `@` chooser MAY expose one combined local file-or-directory action outside the typed resource-reference contract. Activating it MUST directly open one Electron-main-owned native chooser without a renderer-authored path or intermediate type confirmation. Main MAY return the selected directory path or a bounded payload for the exact selected regular file, but MUST reject other filesystem object types and files above the existing attachment limit. Renderer MUST route the result through the existing attachment or project-context flow.

## Rationale

- Stable IDs preserve the user's exact chooser selection while server-side revalidation closes stale-catalog and renderer-forgery gaps.
- Reusing immutable project binding, session activation, global eligibility, and Tool Snapshot assembly keeps one owner for every authorization and execution decision.
- Source-level extension/MCP references match the global configuration model without duplicating their contributed tools in the chooser.
- An optional additive request field preserves existing clients and requires no persistence migration.

## Consequences

- Submit orchestration now has a validation-and-apply phase before user-event creation, and partial dependency failures are reported as actionable submit errors.
- Project, Skill, and tool bindings remain conversation-scoped and therefore influence later turns in that conversation until changed through their existing controls.
- Retrying or resubmitting the same reference is idempotent; event payloads provide reviewable canonical evidence without becoming an authorization source.
- Existing generated renderer bindings remain generated from Go and must not be hand-edited.
- The local chooser adds a narrow preload capability but does not add a Core RPC, persistence shape, implicit path authorization, or resource-reference kind.
- Electron's native dialog supports combined file/directory selection on macOS. Windows and Linux expose directory-only behavior when both native properties are requested, so those platforms require separate acceptance and a follow-up design before claiming equivalent combined selection.

## Rejected alternatives

- Parse `@name` from prompt text: names are ambiguous and text must not grant capability.
- Trust the renderer's enabled state: catalogs can become stale and renderer state is outside the authorization boundary.
- Expand extension and MCP tools in the chooser: this contradicts source-level configuration and exposes unstable implementation detail.
- Inject raw catalog/source data directly into the prompt: it risks secrets and private data, bypasses canonical context assembly, and cannot enforce execution eligibility.
- Store a separate mention-binding table: existing session project and activation state plus canonical event evidence already own the required lifecycle.

## Verification

`AT-PROJECT-003`, `AT-SESSION-001`, `AT-WORKSPACE-001`, `AT-TOOL-001`, `AT-EXTENSION-001`, `CT-SECURITY-001`, and `AT-UI-001` cover exact selection transport, immutable binding, merge/idempotence, source expansion, global-disable refusal, forged/stale input refusal, event redaction, and composer keyboard/layout behavior.
