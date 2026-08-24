# Reduce built-in Agents to one visible default

## Problem or goal

The fresh Agent catalog exposes several overlapping built-in modes even though the composer needs one capable default. Remove the unnecessary built-ins so the default input experience presents only Assistant, while retaining hidden workers required for summaries, titles, and scheduled execution.

## Expected behavior

`REQ-AGENT-001` exposes exactly one visible built-in primary Agent, `assistant`, and uses it as the composer and new-session default. User-created and project-defined modes remain supported. Removed legacy built-in identifiers do not reappear as built-ins; an old session carrying one of those identifiers resolves safely through Assistant.

## Non-goals

This Work does not remove custom Agent management, project runtime Agent definitions, hidden summary/title/scheduler workers, delegation, permissions, tools, providers, or prompt management.

## Impact

Core removes the redundant built-in definitions and their embedded prompt documents, preserves only runtime-required hidden workers, and adds a bounded compatibility fallback for old session values. The renderer fallback catalog contains only Assistant, so a fresh composer shows one default Agent. There is no Electron, transport, schema, secret, process, dependency, or release-boundary change.

## Implementation constraints

Core remains authoritative for catalog visibility and resolution. Compatibility applies only to the known removed built-in IDs and must not hide errors for unknown or malformed IDs. Persisted user definitions and project overlays keep their current contracts. Internal workers remain hidden and unmanageable.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `SINGLE-AGENT-DOC-001` | `REQ-AGENT-001`, `REQ-SESSION-001` | Primary specs and Traceability define one visible built-in default | `AT-AGENT-001`, `AT-SESSION-001` | Completed |
| `SINGLE-AGENT-CORE-001` | `REQ-AGENT-001`, `REQ-SESSION-001` | Core catalog and prompt registry retain Assistant plus required hidden workers and handle legacy session IDs | `AT-AGENT-001`, `AT-SESSION-001` | Completed |
| `SINGLE-AGENT-UI-001` | `REQ-AGENT-001`, `NFR-UI-001` | Fresh composer fallback presents only Assistant | `AT-AGENT-001`, `AT-UI-001` | Completed |
| `SINGLE-AGENT-QA-001` | all | Focused and full applicable gates pass | all | Completed |

## Acceptance and evidence

- A fresh global/project catalog returns exactly one visible built-in mode, Assistant; required internal workers appear only when hidden modes are explicitly requested.
- The composer fallback contains Assistant only and initializes to Assistant.
- Custom global and project-defined modes remain available under their existing rules.
- Known removed built-in IDs from historical sessions resolve through Assistant; arbitrary unknown IDs still fail.
- Removed built-in Agent prompts are absent from the embedded prompt catalog.
- Failure, cancellation, timeout, teardown, migration, rollback, secrets, platform packaging, and narrow/wide layout changes are N/A; this is a synchronous catalog reduction with no new lifecycle or layout.

Verification evidence on 2026-08-23:

- `pnpm docs:check` passed with 84 Markdown files, 40 YAML files, 20 Requirements, 20 Test IDs, 21 ADRs, and 39 Work Packages before this Work was sealed.
- Focused `go test ./app ./infra/persistence` passed, including exact visible-catalog membership, required hidden prompts, custom/project modes, retired-session fallback, and unknown-mode refusal.
- `pnpm scripts:test` passed 59 desktop model tests plus governance and extension example suites; the new composer fallback tests prove Assistant is the only fresh option and every retired built-in normalizes to it.
- `pnpm test:core` passed all Core packages.
- `pnpm lint` passed with only the repository's existing Fast Refresh warnings in shared UI and root files.
- `pnpm build` passed TypeScript compilation and the Vite production build; Vite reported only existing large-barrel and chunk-size advisories.
- `git diff --check` passed.

## Security and data lifecycle

No secret, private data, credential, logging, clipboard, process, network, or persistence ownership changes. Removing capability profiles cannot grant authority; Assistant continues to be constrained by the existing runtime permission engine.

## Compatibility and migration

No schema or destructive user-data migration. Existing session/turn values are not rewritten. Known retired built-in mode IDs use Assistant behavior when resolved; custom and project modes remain distinct. Downgrade restores the older built-in list from code.

## Bug root cause (type=bug only)

N/A.
