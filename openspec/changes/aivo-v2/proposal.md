# Proposal: Aivo v2 redesign

## Why

Aivo has accumulated a broad desktop and local-agent surface: provider setup,
projects, sessions, tools, permissions, plugins, MCP, skills, terminals, and
worktrees. The next version requires substantial product and interaction
changes. Continuing through isolated screen-level edits would make contracts,
data compatibility, and completion criteria difficult to control.

Aivo v2 will be treated as an incremental product migration with a protected v1
archive and independently releasable vertical slices.

## Goals

- Define a coherent primary workflow from first launch to completed agent task.
- Make projects, conversations, agent runs, tools, and worktrees understandable
  as distinct product concepts.
- Keep the Electron renderer, local Go core, and persistence boundaries clear.
- Version backend contracts and expose actionable, consistent error responses.
- Preserve user projects, provider configuration, sessions, and history through
  explicit migrations or documented non-migration decisions.
- Keep every milestone runnable, testable, and recoverable.
- Support narrow windows, long content, loading, empty, failure, cancellation,
  and permission states as first-class UI states.

## Non-goals for preparation

- Choosing final visual styling before primary workflows are approved.
- Rewriting every subsystem at once.
- Removing v1 data or compatibility paths before migration verification.
- Publishing, packaging, or pushing branches without a separate release action.
- Expanding into cloud sync, collaboration, or mobile clients unless later
  approved as explicit v2 scope.

## Proposed product slices

1. Application shell, first-run setup, and provider health.
2. Project creation/opening and project context.
3. Conversation creation, prompt submission, streaming, and cancellation.
4. Agent execution visibility, terminal activity, permissions, and questions.
5. Files, diffs, diagnostics, and task-completion review.
6. Worktree lifecycle and parallel agent work.
7. Skills, plugins, and MCP discovery/activation.
8. Settings, recovery, diagnostics, and data-management surfaces.

The ordering is provisional until the decision log is resolved.

## Success criteria

- A new user can connect a provider, open a project, run a task, understand tool
  activity, and review the result without undocumented setup.
- An existing user can start v2 without silent loss or corruption of v1 data.
- Each slice has documented contracts and passes core tests, frontend lint, and
  frontend build.
- Failure and cancellation paths leave sessions and processes in recoverable,
  observable states.
- No renderer feature directly bypasses the preload/transport boundary to reach
  privileged OS or persistence capabilities.
- A rollback procedure can recover the archived source and a pre-migration data
  copy.

## Release strategy

Develop on `codex/aivo-v2` using small commits grouped by vertical slice. Keep
the v1 archive immutable. Introduce compatibility adapters and feature flags
where old and new behavior must coexist, then remove them only after migration
and acceptance evidence exists.
