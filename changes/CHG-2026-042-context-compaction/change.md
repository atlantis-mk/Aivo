# Add provider-aware session context compaction

## Problem or goal

Aivo already owns summary persistence and a manual `CompactSessionContext` RPC, but automatic pressure uses one fixed character ceiling unrelated to the selected model and the desktop exposes no direct compaction action. Make compaction a complete conversation capability: automatically compact at 80% of the effective model context by default and let the user invoke the same bounded operation through `@compact`.

## Expected behavior

Under `REQ-SESSION-001`, Core resolves the selected model's advertised context length when available, converts the configured threshold percentage into a safe token trigger, and defaults to 80%. It compacts only settled history, retains a recent tail, persists one summary plus a bounded `context_compacted` note, and uses that summary with only post-boundary messages in later Provider requests. Missing model capacity falls back to the existing conservative local budget. Repeated automatic checks below the threshold are no-ops. Under `REQ-WORKSPACE-001`, the composer `@` picker exposes “压缩上下文”; selecting it consumes the active query and invokes the existing RPC without creating prompt text or a typed resource reference.

## Non-goals

No remote telemetry, billing estimate, tokenizer download, database migration, deletion of visible history, Provider-side prompt-cache control, configurable summarizer UI, slash command, or automatic retry after a real Provider context-overflow error is included.

## Impact

Go Core owns capacity resolution, pressure estimation, range selection, summary persistence, execution-state transitions, and model-visible projection. The renderer adds one local `@` action and uses the existing typed desktop service. Electron, credentials, tools, MCP, extensions, terminals, processes, worktrees, dependency graph, database schema, and packaging are unchanged. The existing RPC and summary tables remain authoritative; the system-note payload gains only safe numeric/source metadata.

## Implementation constraints

The default threshold is 80 and project runtime configuration may override or disable automatic compaction through its existing fields. Pressure uses Provider/model catalog capacity when positive and a conservative fallback otherwise; malformed capacity or configuration cannot produce a zero/negative trigger. The newest user turn must never disappear into an automatic summary before it is answered. Compaction runs only while the session is idle, is cancellation-aware, restores a terminal execution state on every outcome, and refuses concurrent manual execution. The summarization input is bounded to the selected visible event range and may carry the previous durable summary for consolidation; logs and notes exclude prompts, responses, tool payloads, and credentials. Historical rows without a boundary remain compatible.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `TASK-042-01` | `REQ-SESSION-001` | Provider-aware 80% pressure policy and safe fallback | `AT-SESSION-001` | Complete |
| `TASK-042-02` | `REQ-SESSION-001` | Bounded summary boundary with recent-tail preservation and repeat safety | `AT-SESSION-001` | Complete |
| `TASK-042-03` | `REQ-WORKSPACE-001`, `NFR-UI-001` | Keyboard/click accessible `@compact` local action with success/failure feedback | `AT-WORKSPACE-001`, `AT-UI-001` | Complete |
| `TASK-042-04` | `REQ-SESSION-001`, `REQ-WORKSPACE-001` | Focused and full Core/desktop/build/document verification | `AT-SESSION-001`, `AT-WORKSPACE-001`, `AT-UI-001` | In progress; automated gates passed, user-owned visual acceptance remains |

## Acceptance and evidence

Automated evidence: focused Core compaction tests, desktop mention/action tests, full Core tests, lint, build, and documentation validation. Per user direction, final visual acceptance is performed manually by the user, so this Work remains `Implementing` rather than being sealed.

- Happy path: a known-capacity model automatically compacts settled context at the configured 80% boundary; `@compact` performs the same operation while idle and leaves the composer ready for the next prompt.
- Boundary/repetition: 79% is a no-op, 80% qualifies, disabled automatic compaction remains off, missing capacity uses the conservative fallback, and a second check with no new compactable head does not create another summary.
- Retention: the latest user/assistant turn remains verbatim, prior summary facts are consolidated, compacted messages remain in user-visible history, and only the summary plus post-boundary messages reach later Provider requests.
- Failure/cancellation/concurrency: busy sessions refuse manual compaction, cancelled/failed summarization records no successful compaction note, and execution state does not remain `compacting`.
- UI: the local action is fuzzy-searchable by `compact` and Chinese copy, consumes the `@` query, creates no resource tag, provides success/error feedback, and remains keyboard reachable at wide and narrow widths.
- Compatibility/migration/rollback: existing summaries, clients, project configuration, and RPC callers remain valid; disabling automatic compaction and omitting the renderer action rolls back behavior without data conversion.

## Security and data lifecycle

The existing local session database retains summaries and original history. Only the bounded selected conversation range and previous summary may be sent to the configured summarization Provider. The renderer supplies only the active session ID through the existing RPC. No credential, authorization header, raw Provider payload, private filesystem content outside already model-visible context, or new analytics/log payload is added.

## Compatibility and migration

No schema transition applies. Runtime settings remain optional and existing clients may continue calling `CompactSessionContext` with only `sessionId` and `characterBudget`. Additive note metadata is ignored by older renderers. Historical summaries without the new selection metadata continue to use their stored event boundary.
