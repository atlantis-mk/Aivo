# Prevent cancelled turns from leaking into later conversation turns

## Problem or goal

On macOS development build `0.0.0-development`, start a conversation in request-approval mode, ask Aivo to run `sleep 20`, stop the turn, then send an unrelated exact-edit request. The cancelled Bash intent is presented again and delays the new request until the stale command is denied. Expected behavior is immediate isolation: stopping a turn terminates its active execution, invalidates its pending interactions, and makes the next user message the only active task. Actual behavior preserves the cancelled user event in model history without a model-visible cancellation marker and can leave its approval pending.

## Expected behavior

- `REQ-SESSION-001`: cancelling a turn leaves its visible UI history intact but excludes that turn's user and assistant events from later model chat history, so its instructions cannot be resumed implicitly.
- `REQ-TOOL-001`: active and pending tool work owned by the cancelled turn reaches an actionable terminal state and is never replayed automatically.
- `NFR-RELIABILITY-001`: pending permission and question interactions owned by the cancelled turn are denied/rejected, their waiters are notified, and later turns contain no stale interactive state.

## Non-goals

- Do not hide the cancelled prompt from the user's conversation timeline.
- Do not add a new permission status, persistence schema, RPC, or renderer interaction.
- Do not change explicit retry, user-approved replay, PTY persistence, or permission policy.

## Impact

- Core application: cancellation cleanup, model-context filtering, and terminal tool-call state.
- Persistence: existing rows are updated through current statuses; no schema or migration.
- Renderer/Electron/transport/providers/extensions: no contract change. Existing refresh events surface the cleaned state.
- Security: removes the risk that a stale command is accidentally approved in a later turn.
- Dependencies, packaging, release rollback, and platform scope: none.

## Implementation constraints

- Core remains the lifecycle owner; renderer filtering alone is insufficient.
- Cancellation must happen before interaction rejection so a notified waiter cannot continue the cancelled agent loop.
- Cleanup is scoped by exact session and turn IDs and must not affect another turn.
- Context filtering uses persisted Turn ownership, including `Turn.UserEventID`, because user events are created before their Turn ID exists.
- Existing status values remain authoritative; no migration or ADR revision is required.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `BUG-CANCEL-001` | `REQ-SESSION-001` | Exclude cancelled-turn messages from later model-visible history while retaining normal UI history | `AT-SESSION-001` | Complete |
| `BUG-CANCEL-002` | `REQ-TOOL-001`, `NFR-RELIABILITY-001` | Invalidate turn-owned pending permissions/questions and interrupt unfinished tool calls | `AT-TOOL-001`, `CT-RELIABILITY-001` | Complete |
| `BUG-CANCEL-003` | `NFR-RELIABILITY-001` | Add pre-fix failing/post-fix passing regression coverage for cancellation followed by a new turn | `AT-SESSION-001`, `CT-RELIABILITY-001` | Complete |

## Acceptance and evidence

- A focused regression must fail before the fix because a cancelled prompt remains model-visible and pending interactions remain active, then pass after the fix.
- Cancelling a turn rejects only that turn's pending permissions and questions, not interactions owned by another turn.
- Pending/running tool calls for the cancelled turn reach `interrupted` or a cancellation-specific failed result and cannot be replayed automatically.
- The cancelled prompt and stop marker remain visible in the desktop history; the next model request omits the cancelled task.
- Repeated cancellation is idempotent; cancellation without pending interactions succeeds.
- Timeout, migration, rollback, dependency loss, wide/narrow UI, installer, and signing acceptance are N/A because this is a schema-neutral Core lifecycle correction with no UI layout or package change.
- Verification targets macOS x86_64 development plus the repository Core and documentation gates.
- Pre-fix evidence on 2026-08-03: `go test ./app -run '^TestCancelTurnIsolatesLaterModelHistoryAndPendingInteractions$' -count=1` failed with `cancelled permission status = "pending", want denied` before product-code changes.
- Post-fix focused evidence: the cancellation-isolation regression, provider-stream cancellation, execution interruption, Bash approval cancellation, and request-approval override tests passed together. The regression proves the cancelled user event remains in normal UI history while disappearing from later non-system model messages; its permission becomes denied, its question rejected, and its unfinished tool call interrupted. A stale approval attempt remains denied, while an unrelated session's pending approval is untouched.
- `pnpm test:core` passed on 2026-08-03 after updating the previous timeout-based approval test to use an explicit pending request and denial. All Core app, CLI, persistence, and HTTP transport packages passed.
- Repository gates passed on 2026-08-03: `pnpm docs:check`, `pnpm scripts:test`, `pnpm lint`, and `pnpm build`. Lint reported only the existing Fast Refresh warnings in routes/shared UI; build reported only existing large-barrel and chunk-size advisories.
- `git diff --check` passed. Verification platform: macOS 14.8.7, Darwin 23.6.0 x86_64, Go 1.26.0 darwin/amd64.

## Security and data lifecycle

No new private data is persisted or logged. Existing permission arguments and question content retain their current ownership. Cancellation updates their current rows to terminal denied/rejected states and wakes in-memory waiters. Cancelled conversation events remain visible to the user but are excluded from later model input, preventing stale private instructions and commands from being reissued.

## Compatibility and migration

No schema, API/RPC/IPC, settings, or file-format change. Existing cancelled conversations are handled by turn-aware context assembly when next loaded. Rollback restores the previous behavior but can reintroduce stale command replay; there is no irreversible data operation.

## Bug root cause (type=bug only)

Affected version: `0.0.0-development`. `CancelTurn` cancelled the active context and persisted a normal system note, but model history included only user/assistant events and did not filter by cancelled Turn ownership. The system note was therefore invisible to the model while the cancelled instruction remained active. In addition, context cancellation in permission waiting returned `ask`, and cancellation did not terminalize turn-owned permission/question requests. Existing tests covered provider-request cancellation and primitive process teardown separately, but did not submit a new message after cancelling a turn with an interactive tool call. The fix filters model chat by cancelled Turn and `UserEventID`, denies/rejects pending interactions, interrupts unfinished tool records, makes permission/question persistence transitions pending-only, and prevents late approval from reviving a cancelled request. The pre-fix regression failed on the pending permission and passed after these changes. Fix version: `0.0.0-development`.
