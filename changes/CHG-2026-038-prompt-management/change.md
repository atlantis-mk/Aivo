# Manage validated file-backed prompts

## Problem or goal

Core prompts are spread across Go and renderer literals, while Agent prompts have a separate SQLite editor. Users need one global, inspectable Markdown catalog whose edits become model-visible only after Core validation.

## Expected behavior

`REQ-PROMPT-001` provides categorized built-in Markdown defaults, Core-owned global overrides, last-known-good fallback, typed management operations, and a responsive `/prompts` page reached from the project top bar. Prompt text never grants capability or bypasses runtime validation.

## Non-goals

Project-level prompt management, arbitrary unbound prompt documents, live filesystem watching, and editable tool schemas are excluded. Existing `AGENTS.md`, Skills, extension context, MCP Prompts, and project runtime files remain independent inputs.

## Impact

This Work changes Core prompt ownership, schema-v9 Agent-mode payloads, local RPCs, renderer routing, and prompt consumers. It adds no remote service, dependency, credential flow, telemetry, or new executable authority.

## Implementation constraints

ADR-0021 owns file/persistence precedence. Core parses bounded symlink-free Markdown, retains an immutable validated snapshot for an execution, atomically publishes later valid revisions, and emits diagnostics without raw bodies. Invalid working files remain inactive. Renderer access is typed and cannot supply arbitrary filesystem paths.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| PROMPT-01 | REQ-PROMPT-001 | Built-in Markdown registry, validation, override storage, snapshot rendering | AT-PROMPT-001, CT-SECURITY-001 | Verified |
| PROMPT-02 | REQ-PROMPT-001, REQ-AGENT-001 | Consumer migration and Agent prompt references | AT-PROMPT-001, AT-AGENT-001 | Verified |
| PROMPT-03 | REQ-PROMPT-001, REQ-WORKSPACE-001 | Typed RPCs and responsive management page | AT-PROMPT-001, AT-WORKSPACE-001, AT-UI-001 | Verified |
| PROMPT-04 | NFR-RELIABILITY-001 | Schema-v9 backup, migration, fallback and restart recovery | CT-RELIABILITY-001 | Verified |

## Acceptance and evidence

- Required/optional/custom lifecycle, invalid fallback, external reload, repeated save, restart, reference refusal, migration recovery, and immutable snapshot behavior are covered by `core/app/prompt_registry_test.go`, Agent-mode tests, schema migration tests, and typed RPC compilation.
- `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, and `pnpm build` passed on 2026-08-23. Existing lint output contains only the repository's pre-existing Fast Refresh warnings.
- Responsive UI evidence: `evidence/prompts-wide.png` (1440×960) and `evidence/prompts-narrow.png` (760×960), rendered from the real `/prompts` route with an isolated injected catalog and without opening the user's Aivo database.

## Security and data lifecycle

Prompt bodies stay on the local machine in private Core-owned storage, are never logged or exported as diagnostics, and are not authority. Content-addressed validated copies support recovery; reset/delete removes only managed overrides or unreferenced custom documents.

## Compatibility and migration

Legacy Agent-mode `prompt` writes are adapted into prompt documents. Schema v9 requires a verified v8 backup; downgrade restores that backup. Existing project prompt overlays remain higher-precedence project inputs.

## Bug root cause (type=bug only)

N/A.
