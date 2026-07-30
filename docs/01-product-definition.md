# Aivo product definition

## Product promise

Aivo is a local-first desktop environment for running AI-assisted software work against user-selected projects. It coordinates model providers, conversations, tools, permissions, terminals, worktrees, skills, plugins, MCP, and code intelligence while keeping privileged execution in a local runtime.

## Current users

The working user is a software builder who wants an agent to inspect and change a local codebase, run development tools, ask for approval when required, and return evidence-backed results. The exact v2 primary persona and top jobs remain `OPEN-001`.

## Required outcomes

- A user can configure a supported model provider without exposing credentials to the renderer or logs.
- A user can open a local project, start or continue a conversation, submit a task, observe progress, answer questions or permissions, cancel work, and review results.
- Tool, terminal, MCP, LSP, skill, plugin, and worktree activity has clear ownership, lifecycle, and recoverable failure behavior.
- Existing user data is not silently lost or corrupted during v2 development or migration.
- The desktop remains usable for narrow windows, long content, loading, empty, failure, cancellation, and permission states.

## Product principles

- Local privileged operations stay local and capability-bound.
- The user can understand what the agent is doing and when intervention is needed.
- Every long-running activity can be cancelled and cleaned up by its owner.
- Public contracts, persistence transitions, and release compatibility are explicit and testable.
- New v2 work is delivered as small vertical slices, not a simultaneous rewrite.

## Non-goals

Unless promoted by an accepted scope Work, Aivo does not require cloud sync, multi-user collaboration, Aivo accounts, mobile clients, telemetry, or a built-in browser UI. Final v2 visual styling is not a substitute for approved workflows and contracts.

## Success measures

- A new user can connect a provider, open a project, run a task, understand tool activity, and review the result without undocumented setup.
- Core tests, desktop lint, desktop build, documentation checks, and applicable packaging gates pass for each release.
- Failure and cancellation leave sessions, processes, streams, and data in recoverable states.
- Upgrade and rollback procedures protect v1 data before any persistence transition.
