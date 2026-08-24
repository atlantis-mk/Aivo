# Show local conversation runtime statistics

## Problem or goal

Aivo parses provider token usage but discards it after each successful model request, and the desktop conversation does not expose request count, latency, throughput, cache reuse, or token totals. Add a compact Harness-inspired statistics strip below the active composer so users can understand the local session's settled model work without opening a diagnostic surface.

## Expected behavior

Under `REQ-SESSION-001` and `NFR-UI-001`, a conversation with persisted runtime metrics shows grouped Chinese text in this order: completed turns and LLM steps, summed LLM duration, average first-token latency and decode throughput when sampled, cache-hit percentage when every contributing input sample reports cache accounting, and input/output token totals. A step is one successful provider response; a turn is counted once when it contains at least one such step. LLM duration spans dispatch to response completion, TTFT spans dispatch to the first non-empty streamed text or tool-call delta, and throughput divides sampled output tokens by sampled post-first-token decode time. Unsupported or unavailable readings are omitted rather than estimated. Historical events without metrics do not contribute.

## Non-goals

No remote telemetry, cost estimate, context-window meter, per-message detail panel, provider benchmarking, failed/cancelled partial-step accounting, database schema change, or retroactive estimation of historical usage is included.

## Impact

Core normalizes OpenAI Chat/Responses, DeepSeek, Anthropic, Gemini, Bedrock Converse, Cohere, Ollama, AI SDK, and common OpenAI-compatible usage fields into inclusive input, output, reasoning, cache-read, and cache-write buckets. It retains whether cache accounting was actually reported, aggregates successful steps inside one turn, writes one bounded `runtimeMetrics` object on the final assistant event, and folds all persisted summaries through a read-only local stats RPC. The renderer renders that durable aggregate as a tertiary single-line strip under the composer and falls back to its loaded event window only for compatibility with an older Core. Electron privilege boundaries, credentials, tools, MCP, LSP, terminals, processes, worktrees, dependencies, and release packaging are unchanged. The additive local event payload and read-only RPC do not require an ADR or schema migration.

## Implementation constraints

Go owns timing and provider-usage semantics; the renderer only validates, folds, formats, and presents typed numeric summaries. Normalized total input includes uncached input plus cache reads and writes. Cache-hit percentage uses provider-reported cache-read tokens divided by that total, is clamped to 0–100, and is omitted when any contributing input sample lacks an explicit cache-usage field; an explicitly reported zero remains a valid `0%`. Negative, non-finite, malformed, or internally inconsistent payload values are ignored or clamped safely. The strip must not log or persist prompt, response, tool, credential, or raw provider data. It must survive conversation reload, update after a settled turn, remain centered beneath the existing composer, truncate with an accessible full-value tooltip, and not alter shared primitives or generated bridge/routes.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `TASK-041-01` | `REQ-SESSION-001` | Normalize cache usage and persist bounded per-turn runtime metrics | `AT-SESSION-001` | Complete |
| `TASK-041-02` | `REQ-SESSION-001` | Validate, fold, and format persisted session statistics in the renderer | `AT-SESSION-001` | Complete |
| `TASK-041-03` | `NFR-UI-001` | Render the responsive composer statistics strip with unavailable groups omitted | `AT-UI-001` | Complete |
| `TASK-041-04` | `REQ-SESSION-001`, `NFR-UI-001` | Run focused tests, core tests, docs, lint/build, and wide/narrow visual acceptance | `AT-SESSION-001`, `AT-UI-001` | Partial |

## Acceptance and evidence

- Happy path: a one-turn/one-step response with timing and usage renders every available group with compact, locale-stable values.
- Multi-step/repetition: tool-follow-up requests add steps while the owning turn contributes once; repeated turns accumulate without double counting.
- Boundaries: explicitly reported zero cache reads display 0%; missing cache accounting omits cache hit without hiding input/output totals; missing usage, TTFT, or decode duration omits only the affected group; long totals compact to K/M; malformed payloads do not render `NaN`, negative values, or percentages over 100.
- Failure/cancellation/timeout: incomplete provider requests do not contribute a successful step; existing cancellation and recovery behavior remains unchanged.
- Compatibility/migration/rollback: historical events are accepted unchanged; removing the renderer strip leaves additive payloads inert; no migration or irreversible action applies.
- UI: wide and narrow dark-theme screenshots verify alignment, truncation/tooltip, scrolling clearance, and no composer overlap.
- Automated evidence on 2026-08-24: `pnpm docs:check` passed; `pnpm scripts:test` passed 4 governance-script tests, 63 desktop tests, and 4 extension UI tests; `pnpm test:core` passed all Core packages; the focused Core metric/parser tests and three renderer metric tests passed; `pnpm lint` passed with only existing Fast Refresh warnings; `pnpm build` passed; `git diff --check` passed.
- The user elected to perform the wide/narrow application verification; the focused QA route is `apps/desktop/qa/session-runtime-stats.html`.
- Work remains `Implementing` until user-run wide/narrow visual acceptance is resolved and recorded.

## Security and data lifecycle

Only integer token/cache counts, step count, and millisecond durations are persisted with the final assistant event already owned by the local session database. No prompt, response body, tool arguments/results, filesystem content, authorization material, or provider payload is copied into metrics. Normal session deletion/retention owns cleanup; logging, clipboard, crash, backup, and remote analytics behavior are unchanged.

## Compatibility and migration

The `runtimeMetrics` payload member and cache-usage fields are additive. Historical rows and providers that omit usage remain valid, no SQLite schema transition is required, and rollback can ignore the extra JSON fields.
