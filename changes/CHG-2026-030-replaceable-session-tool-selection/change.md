# Replace session automatic tool selections on Agent request

## Problem or goal

Automatic Host selection is currently request-scoped, recently used tools receive a three-turn warm lease, mode-default extension tools bypass selection, and the primary Agent cannot request a new selection because `tool_resolve` is hidden. Global tool preferences also revoke already selected session tools. The required behavior is a stable but replaceable automatic tool set for each conversation, separate from manual activation.

## Expected behavior

The Host initializes one bounded `autoToolSet` from globally visible eligible candidates for the conversation's first primary request. The primary Agent receives the four execution primitives, the Host-owned `tool_resolve` control, every manually pinned conversation tool, and exactly the current automatic set. When those capabilities are insufficient, the Agent calls `tool_resolve` with the missing capability; auxiliary selection atomically replaces the complete automatic set for the next model step instead of accumulating or warming tools. Manual tools remain unchanged. Tools outside the resulting visible set are absent from Provider declarations and the immutable Tool Snapshot.

Global preferences filter future auxiliary candidates and new manual selections only. They do not revoke a tool already held by a conversation. Source trust, enablement, readiness, mode/toolset eligibility, current registration, and the Tool Snapshot remain authoritative execution prerequisites.

## Non-goals

No extension lifecycle, credential, permission, Provider wire-name, renderer layout, schema version, installation, trust, source enablement, or historical rendering change. The selector cannot install, trust, enable, authorize, or execute a contributed tool.

## Impact

Core gains a session-owned replaceable automatic-name set in existing execution metadata, exposes the existing bounded selector as a Host control tool, stops warm leasing and mode-default bypass, and changes global preferences from live revocation to candidate visibility. Provider schemas and Tool Snapshots are rebuilt after a successful replacement. Renderer global and conversation catalogs retain their current switches and RPC ownership. No Electron, preload, dependency, process, credential, database schema, or platform change.

## Implementation constraints

Automatic replacement is atomic and bounded. Resolver failure, cancellation, invalid names, empty required matches, global-preference read failure, stale registrations, and source failure preserve the previous automatic set. A successful replacement may be empty only when the resolver returns a valid optional empty selection. Manual session state is never rewritten by automatic selection. The control tool receives sanitized candidate summaries and never exposes hidden schemas or names to the primary Agent. Runtime execution continues to require the current immutable Tool Snapshot.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `SESSION-TOOLS-DOC-001` | `REQ-SESSION-001`, `REQ-TOOL-001`, `REQ-EXTENSION-001` | Primary specs, ADR, security, tests, and traceability own the replaceable-set contract | `AT-SESSION-001`, `AT-TOOL-001`, `AT-EXTENSION-001` | Completed |
| `SESSION-TOOLS-CORE-001` | `REQ-SESSION-001`, `REQ-EXTENSION-001` | Initial selection persists; `tool_resolve` replaces only the automatic set; manual tools persist | `AT-SESSION-001`, `AT-EXTENSION-001` | Completed |
| `SESSION-TOOLS-GLOBAL-001` | `REQ-TOOL-001`, `NFR-SECURITY-001` | Global preferences filter future candidates/new manual activation without revoking current snapshots | `AT-TOOL-001`, `CT-SECURITY-001` | Completed |
| `SESSION-TOOLS-QA-001` | all | Focused and repository gates pass with recorded evidence | all | Completed |

## Acceptance and evidence

- A new conversation receives core primitives, `tool_resolve`, manual selections, and one bounded initial automatic set; unrelated tools are absent.
- Calling `tool_resolve` with a different capability replaces the previous automatic names for the next primary step, while manual names remain.
- Repeated selection is idempotent; no warm lease or automatic accumulation remains.
- Resolver error, cancellation, invalid/forged selection, required no-match, preference-read failure, and source failure preserve the previous set and do not expose hidden tools.
- Global disablement hides a tool from future auxiliary/manual selection. A conversation already holding it retains it until automatic replacement or manual removal.
- `aivo.projects` tools use automatic selection rather than mode-default exposure.
- Focused Core tests, `pnpm test:core`, `pnpm lint`, `pnpm build`, `pnpm docs:check`, and `git diff --check` provide evidence.

Verification evidence recorded on 2026-08-10: focused Core tests cover initial selection, primary-Agent-triggered replacement on the next Provider step, manual-set independence, resolver failure preservation, global candidate visibility, current-session retention, and Host-control name reservation. `pnpm test:core`, `pnpm lint`, `pnpm build`, `pnpm docs:check`, and `git diff --check` passed. Lint and build retained only the repository's existing Fast Refresh, large-barrel, and chunk-size warnings. Interactive desktop acceptance is not applicable because this Work changes no renderer, preload, Electron, or user-facing control surface.

## Security and data lifecycle

Session metadata stores only bounded canonical tool names and an initialization marker. Auxiliary input contains sanitized catalog summaries without credentials, arguments, results, prompts beyond the concise missing capability, or private filesystem content. Global preferences no longer act as emergency revocation; disabling a source remains the lifecycle mechanism for removing its registrations from active sessions. Snapshot-bound execution rejects unadvertised or stale calls.

## Compatibility and migration

No database schema transition is required. Existing `rememberedDeferredTools` remains the manual set. Existing warm metadata is ignored. Conversations without the new marker initialize on their next primary request. Downgrade may resume request-scoped selection and global live revocation.

## Bug root cause (type=bug only)

N/A.
