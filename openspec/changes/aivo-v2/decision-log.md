# Aivo v2 decision log

Implementation should not begin beyond reversible scaffolding until the P0
items below are decided. Record the decision, date, rationale, and affected
slices in this file.

| Priority | Decision | Status | Default working assumption |
| --- | --- | --- | --- |
| P0 | Primary user and top three jobs-to-be-done | Open | Local software builder using coding agents |
| P0 | Primary object shown on launch: project, conversation, or task | Open | Project with recent conversations |
| P0 | Relationship among conversation, task, agent run, and terminal | Open | Conversation owns tasks; tasks own agent runs |
| P0 | v1 data compatibility promise | Open | Preserve projects, sessions, providers, and history |
| P0 | First releasable v2 slice | Open | Setup + provider health + project entry |
| P0 | Whether v1 and v2 UI coexist behind a flag | Open | Coexist during migration |
| P1 | Built-in browser replacement or permanent removal | Open | Keep web tools, defer browser UI |
| P1 | Worktree defaults and lifecycle ownership | Open | Opt-in per task, visible cleanup |
| P1 | Plugin/skill/MCP information architecture | Open | Unified extensions surface |
| P1 | Settings scope and navigation | Open | Dedicated full-page settings |
| P1 | Supported providers at v2 launch | Open | Preserve currently implemented providers |
| P2 | Cloud sync, collaboration, accounts, or telemetry | Open | Out of scope |

## Decision record template

```markdown
### YYYY-MM-DD — Decision title

- Status: accepted / superseded
- Decision:
- Rationale:
- Alternatives considered:
- Affected slices and contracts:
- Follow-up tasks:
```
