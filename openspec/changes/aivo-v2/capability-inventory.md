# Current capability inventory

This inventory is derived from the archived source tree. Product behavior still
needs hands-on acceptance review before final disposition.

| Area | Current evidence | Initial v2 disposition |
| --- | --- | --- |
| Desktop shell | Electron main/preload with Vite/React renderer | Retain boundary; redesign shell and navigation |
| First-run setup | `/setup`, provider selection and connection flows | Redesign as slice 1 |
| Project workspace | `/projects`, `/projects/chat`, `/projects/plugins` | Redesign information architecture |
| Settings | `/settings` shell with return navigation | Complete/rebuild |
| Providers | Registry, auth, models, validation, health and fallback | Retain core; simplify product flow |
| Projects | Project persistence, description and instructions | Retain and clarify |
| Sessions | Turns, events, execution state, pending input and context | Retain domain intent; review model and API |
| Agent runtime | Agent loop, parallel runs, PTY registry and compaction | Retain with explicit lifecycle semantics |
| Commands/terminal | Terminal service, interactive terminal and command runtime | Retain; unify visibility and cancellation |
| Worktrees | Domain types, persistence, service, RPC and desktop dialog | Validate, then promote to first-class workflow |
| Permissions | Requests, rules and notifications | Retain; redesign confirmation UX |
| File/code tools | File, shell, symbols, diagnostics and LSP tools | Retain with auditable result presentation |
| Web capability | Web tools remain; prior built-in browser UI/runtime removed | Product decision required |
| Skills | Scan, resolve, import candidates and activation UI | Retain; simplify discovery and trust signals |
| Plugins/MCP | Plugin process/catalog plus MCP servers/tools/resources/prompts | Retain behind clearer activation and diagnostics |
| Local persistence | SQLite at `~/.aivo/aivo.db`, GORM schema version 1 | Preserve; replace implicit migration assumptions |
| Local transport | Method-oriented RPC dispatch over HTTP plus event streams | Add versioned resource contracts/adapters |
| Release quality | Go tests, lint, build, release smoke documentation | Expand with v2 acceptance matrix |

## Required hands-on inventory pass

Before changing a capability's disposition, capture the current UI and verify:

- entry point and happy path;
- persisted data created or changed;
- external processes and network dependencies;
- cancellation and restart behavior;
- loading, empty, error, permission, long-content, and narrow-window states;
- whether existing automated tests cover the behavior.

The result for each area must be marked `retain`, `refactor`, `redesign`, or
`remove`, with an owner and target slice in `tasks.md`.
