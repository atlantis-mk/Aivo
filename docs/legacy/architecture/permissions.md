# Permission Architecture

## Principle

Permissions are deterministic runtime enforcement, not prompt instructions. Every external side effect must pass through the Permission Engine before execution. Bash is a critical-risk super-tool and must never be enabled by prompt text alone.

## Decision Model

Permissions resolve to:

- `allow`: execute without interrupting the user.
- `ask`: create an approval request and wait for a one-shot or saved decision.
- `deny`: block execution and return a structured tool error.

Rules apply from broad to narrow: product defaults, agent mode, workspace, tool, command pattern, path, network domain, secret class, and session-level saved approvals. The most specific deny wins; ask wins over allow unless a saved approval applies.

## Configuration Sketch

```yaml
agents:
  planner:
    toolsets: [safe]
    permissions:
      read_file: allow
      search_files: allow
      write_file: deny
      apply_patch: deny
      bash: ask

  builder:
    toolsets: [coding]
    permissions:
      read_file: allow
      search_files: allow
      apply_patch: ask
      bash:
        "go test ./...": allow
        "git diff": allow
        "git push *": ask
        "rm -rf *": deny
        "*": ask
```

## Runtime Flow

Before a tool executes, ToolRuntime should build a permission request containing session id, turn id, agent mode, tool name, normalized arguments, resources such as paths or domains, risk level, and source tool call id. Permission Engine returns allow/ask/deny. Ask creates a pending approval record and UI event. Deny returns a `permission_denied` tool result.

Saved approvals must be scoped: one-shot, session, workspace, or explicit global. They should expire or be user-manageable.

## Audit

Every decision logs session id, turn id, message id, tool name, normalized resources, decision, matched rule, approval id, user reply, start/end time, success/failure, error, workspace, and truncation status.

## First Implementation Boundary

Phase 1 only records tool calls and uses read-only low-risk tools. Phase 2 introduces the Permission Engine before adding write or patch tools.
