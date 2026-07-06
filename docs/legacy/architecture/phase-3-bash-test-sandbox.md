# Phase 3 Bash, Test Runner, Sandbox, And Terminal Implementation Plan

## Objective

Phase 3 adds controlled command execution to the agent runtime. The implementation should let coding sessions run tests and diagnostic commands while keeping every external side effect behind deterministic permission checks, workspace enforcement, output limits, timeout handling, and audit records.

Phase 3 also delivers an OpenCode-style user terminal panel in the conversation surface: a production-grade bottom shell that users can open under the chat composer, backed by real PTY sessions, durable workspace-scoped terminal state, WebSocket streaming, multi-tab management, resize, reconnect, cleanup, and clear separation from model-visible `bash` tool execution. This is not an MVP feature; partial terminal behavior may be developed behind internal flags, but it should not be exposed as complete until the full acceptance criteria in this plan are met.

This phase builds on the existing Phase 2 path:

- `ToolRuntime.ExecuteWithContext` already normalizes tool calls, validates arguments, evaluates permissions, applies timeouts, truncates output, and records tool results through `Service`.
- `PermissionEngine` already supports allow/ask/deny, pending approval requests, remembered rules, session permission modes, and UI notifications.
- Coding sessions already expose read tools, write tools, `apply_patch`, `git_status`, and `git_diff` through `NewCodingToolRegistry`.

## Reference Model

This plan borrows from three mature agent designs:

- Codex: production-grade shell runtime, cached approval keys, explicit exec policy, platform sandbox transforms, network approval, and streamed command output.
- Hermes: pluggable terminal environments, local/Docker/SSH/cloud backend contracts, process-group cleanup, foreground timeout caps, hardline command blocks, orphan container reaping, and secret-aware environment filtering.
- OpenCode: lightweight model-facing shell tool, shell parsing for path and command-pattern extraction, permission prompts for external directories and command patterns, output truncation with retained full-output artifacts, timeline-oriented metadata, and a workspace-scoped bottom PTY terminal panel with tabs, resize, replay, reconnect, and WebSocket transport.

Aivo should combine these strengths rather than copy one implementation:

- Use Codex-style deterministic permission and sandbox enforcement as the security spine.
- Use Hermes-style backend abstraction and lifecycle hardening for process execution.
- Use OpenCode-style shell parsing and output retention to make approvals and timelines understandable.

## Scope

Phase 3 should deliver:

- command detector and exec policy evaluator for normalized shell command classification
- `bash` tool for guarded ad-hoc command execution, including controlled foreground, managed background, and optional PTY modes
- `run_tests` tool for known test/lint/build commands
- environment allowlist and secret-deny behavior
- explicit env override permission for safe non-secret variables
- external-directory permission for cwd and path access outside the workspace
- command timeout, stdout/stderr limits, cancellation, and process cleanup
- managed process registry for background commands
- local sandbox runner interface and first local backend
- Docker sandbox backend interface, with a stub or unavailable implementation until Docker execution is enabled
- retained full-output artifacts for large stdout/stderr
- exact approval-resource keys for remembered shell/test approvals
- hardline command blocklist that cannot be bypassed by permission mode
- tool-call persistence and UI rendering for command, stdout, stderr, exit code, duration, timeout, and approval state
- user-facing conversation bottom shell panel, separate from model tools
- production PTY service with create/list/get/update/remove/attach operations
- PTY WebSocket transport with stdin, output streaming, replay cursor, terminal resize, abnormal disconnect handling, and origin/auth checks
- workspace-scoped terminal state with multi-tab, active tab, title, size, retained buffer cursor, and panel height/open state
- terminal UI integration with the existing conversation composer, permission dock, and timeline spacing

Phase 3 should not deliver:

- broad network sandboxing guarantees on the local backend
- SSH or cloud execution
- plugin/MCP command tools
- autonomous background jobs
- untracked OS-daemon background processes
- secret injection into command environments
- model-supplied sudo passwords or arbitrary stdin to privileged prompts
- a user-visible terminal that only wraps one-shot `bash` commands without PTY semantics
- a terminal panel that cannot reconnect, resize, close sessions, or clean up child processes
- automatic injection of terminal output or terminal input into model context

## Design Principles

1. Shell access is a critical-risk capability. It must never be enabled by prompt text alone.
2. `run_tests` and `bash` should share one sandbox runner. Command execution policy belongs below tools, not duplicated inside each tool.
3. The model-visible tool arguments should be simple, but the runtime must store structured command metadata for approvals, audit, and UI.
4. Known test commands can be lower-friction than arbitrary shell commands, but they still run through the same permission and sandbox pipeline.
5. Local execution is a backend, not the abstraction. Docker, SSH, and cloud backends should fit the same request/result shape later.
6. Approval prompts are not a sandbox. A command is executable only when both the permission engine and the sandbox/backend policy allow it.
7. Saved approvals must be narrow, explainable, revocable, and tied to execution context.
8. Production behavior must be observable: every command has durable metadata, bounded output, duration, exit state, and cancellation/timeout reason.

## Production Non-negotiables

Phase 3 is complete only when these conditions hold:

- No model-visible shell or test tool can execute without passing permission evaluation.
- No command can run with inherited provider API keys, OAuth tokens, refresh tokens, gateway tokens, or app secrets.
- No command can run outside the active workspace unless an explicit external-directory permission covers the target cwd/path.
- Timeout, cancellation, turn cancellation, app shutdown, and session cancellation all trigger process cleanup.
- Local backend cleanup kills the process tree or process group on Unix-like platforms.
- Docker backend, once enabled, uses resource limits, labels, lifecycle tracking, and orphan cleanup.
- Hardline-blocked commands fail before approval creation and before sandbox execution.
- Failed commands remain visible as failed tool calls with stdout/stderr snippets where available.
- Large outputs are retained as artifacts while model-visible and UI-visible summaries remain bounded.
- Approval reuse never applies across a different workspace root, cwd, command, backend, sandbox profile, or network policy.
- The user terminal panel runs through a dedicated PTY service and never reuses the model-facing `bash` permission decision as a blanket interactive shell grant.
- User terminal sessions are workspace-scoped, observable, removable, and cleaned up on app shutdown or explicit close.
- PTY attach uses an authenticated WebSocket or equivalent local-only channel with origin protection, replay cursor, resize propagation, and bounded retained buffer.
- Terminal stdin is user-driven by default. Model-driven stdin must go through `shell.stdin` policy and must not be able to send secrets invisibly.
- Terminal output is not model-visible unless the user explicitly attaches or sends selected output to the conversation.
- The bottom shell is not considered shipped until multi-tab, reconnect, resize, close, process exit, buffer replay, keyboard paste/copy, and layout tests pass.

## Proposed Backend Interfaces

Add sandbox types in `core/app`, keeping process details out of `domain` unless they must cross the API boundary.

```go
type SandboxRunner interface {
    Run(ctx context.Context, request SandboxRequest) (SandboxResult, error)
}

type SandboxRequest struct {
    WorkspaceRoot  string
    CWD            string
    Command        string
    Argv           []string
    Mode           string // foreground, background, pty
    Timeout        time.Duration
    Stdin          string
    Env            map[string]string
    EnvOverrides   map[string]string
    EnvAllowlist   []string
    NetworkPolicy  string
    Backend        string
    Shell          string
    OutputPolicy   domain.OutputPolicy
    SessionID      string
    TurnID         string
    ToolCallID     string
    ToolName       string
    ApprovalKey    string
}

type SandboxResult struct {
    Command       string
    Argv          []string
    Mode          string
    CWD           string
    ExitCode      int
    Stdout        string
    Stderr        string
    TimedOut      bool
    Cancelled     bool
    Truncated     bool
    OriginalSize  int
    StdoutRef     string
    StderrRef     string
    Duration      time.Duration
    Backend       string
    NetworkPolicy string
    ProcessID     int
    ProcessRef    string
}
```

The first implementation should provide:

- `LocalSandboxRunner`: uses `exec.CommandContext`, enforces workspace-relative cwd, sanitized environment, timeout, output caps, and process-group cleanup.
- `DockerSandboxRunner`: satisfies the interface but returns a stable unavailable error until container execution policy is designed.

On Unix-like platforms, local runner cleanup should start the child in a process group and kill the process group on cancellation or timeout. On unsupported platforms, provide best-effort child process cleanup and document the weaker guarantee.

The runner interface should return structured errors instead of string-only failures:

- `sandbox_unavailable`: backend is configured but cannot run.
- `sandbox_policy_denied`: backend policy refused the request.
- `command_timeout`: timeout fired and cleanup completed or was attempted.
- `command_cancelled`: context or session cancellation fired.
- `process_cleanup_failed`: command ended or was cancelled, but cleanup could not be verified.
- `output_retention_failed`: command output exceeded model limits and the full artifact could not be stored.

## Backend Policy

### Local backend

The local backend is for trusted workspace-local work. It must still enforce:

- cwd must resolve under workspace root.
- command starts from a sanitized environment.
- stdin is closed or ignored by default.
- foreground commands have a default timeout and hard maximum.
- process group cleanup is used on Unix-like hosts.
- child processes are killed on timeout, cancellation, app shutdown, and session cancellation.
- stdout and stderr are captured separately, merged only for UI convenience if needed.
- background commands are registered in a process registry with session ownership, poll/wait/kill actions, retained output, and cleanup on expiry or session close.
- PTY commands are opt-in, tracked as interactive process refs, and cannot be silently converted into ordinary foreground commands.

Local backend should not promise strong network isolation. If network policy is `deny`, Phase 3 local execution should either reject network-classified commands before execution or require a future platform sandbox that can enforce it.

### Docker backend

Docker is a production backend, not a thin `docker run` wrapper. Enabling it requires:

- image allowlist or explicit user configuration.
- workspace mount policy with read/write scope.
- CPU, memory, process, and disk limits.
- `no-new-privileges`, capability drop, and non-root default where practical.
- labels including app name, workspace id/hash, session id, and creation time.
- orphan container reaper for exited Aivo containers.
- explicit env forwarding allowlist.
- deterministic cwd mapping such as workspace root to `/workspace`.
- clear unavailable state when Docker is missing, stopped, or unhealthy.

Phase 3 may compile the Docker backend as unavailable first, but the interface must already include the fields needed for this production behavior.

## Tool Specs

### bash

`bash` should use JSON arguments rather than freeform input at first. Freeform shell input is convenient, but JSON keeps cwd, timeout, mode, stdin, environment, and future network policy explicit.

Arguments:

- `command`: required string. Executed by the local shell as a non-interactive command.
- `cwd`: optional workspace-relative directory. Defaults to coding context cwd or workspace root.
- `timeoutSeconds`: optional integer. Capped by runtime maximum.
- `network`: optional enum, initially `deny` or `inherit`. Defaults to `deny` for unknown commands and `inherit` only when policy explicitly allows it.
- `mode`: optional enum: `foreground`, `background`, or `pty`. Defaults to `foreground`.
- `stdin`: optional string. Allowed only when command policy and permission explicitly allow stdin.
- `env`: optional map of env overrides. Allowed only for safe keys and explicit env-override permission.
- `justification`: optional short user-visible reason shown in approval prompts.

Spec:

- capability: `shell.exec`
- risk: `critical`
- category: `shell`
- toolsets: `shell`, `coding`
- requires workspace: true
- default permission: ask unless a saved command rule applies

`bash` should support mature terminal behavior, but every advanced behavior is a separately visible permission dimension:

- background mode requires managed process tracking and may not create untracked daemon processes.
- PTY mode requires an interactive process ref and explicit user-facing state.
- stdin requires explicit approval and is never auto-populated from model guesses.
- env overrides require key-level allow/deny evaluation and cannot include secrets.
- external cwd requires explicit external-directory permission.
- sudo requires a high-risk sudo permission and user-entered credentials through a secure UI path; model-supplied sudo passwords and `sudo -S` password piping are denied.

### run_tests

`run_tests` should be a structured tool that chooses from known commands instead of accepting arbitrary shell text.

Arguments:

- `target`: `core`, `desktop`, or `all`
- `kind`: `test`, `lint`, `build`, or `auto`
- `filter`: optional package/test filter where supported
- `timeoutSeconds`: optional integer. Capped by runtime maximum.

Initial command mapping:

- `core` + `test`: `pnpm test:core`
- `desktop` + `lint`: `pnpm lint`
- `desktop` + `build`: `pnpm build`
- `all` + `test`: `pnpm test:core`
- `all` + `build`: `pnpm build`
- `auto`: infer from changed files and `CodingContext.LanguageStack`

Spec:

- capability: `shell.test`
- risk: `medium`
- category: `shell`
- toolsets: `coding`
- requires workspace: true
- default permission: allow for known read/test commands in full-access mode and ask in request-approval mode

`run_tests` must never pass `filter` through raw string concatenation. Build argv from validated fields or pass through the same command detector before execution.

`run_tests` may run more than one command only when the sequence is declared in code. It should return a structured array of command results if a target expands to multiple commands.

## Command Detector

Add a command detector that runs before permission evaluation and before sandbox execution. This detector is a policy input, not the only security boundary.

Inputs:

- raw command
- normalized cwd
- tool name
- workspace root

Outputs:

- normalized command string
- command category: `read`, `test`, `build`, `write`, `network`, `dangerous`, `unknown`
- risk level: `low`, `medium`, `high`, `critical`
- paths touched or likely touched, when statically knowable
- network hint
- deny reason, when deterministically blocked
- permission resource key for saved approvals
- exact argv/prefix token list when statically parseable
- extracted path patterns and external-directory candidates

Initial rules:

- allow low-risk read commands to be represented as read resources: `git status`, `git diff`, `git log`, `pwd`, `ls`, `find` within bounded arguments.
- classify known test/build commands as `test` or `build`: `go test`, `pnpm test:core`, `pnpm lint`, `pnpm build`.
- classify package install and dependency mutation commands as high-risk write: `pnpm install`, `pnpm add`, `go get`, `go mod tidy`.
- deny obviously destructive commands unless the user explicitly requests future support: `rm -rf /`, `rm -rf .`, `chmod -R 777`, writes under `.git`, fork bombs, direct disk formatting, and command strings that target absolute paths outside the workspace. Classify `sudo` and privilege-escalation wrappers as `shell.sudo` high-risk capability unless the specific target is hardline-denied.
- classify network commands as network-risk: `curl`, `wget`, `ssh`, `scp`, `rsync`, `git push`, `npm publish`, `gh release`.
- classify shell metacharacter usage conservatively. Pipes, redirects, command substitution, and backgrounding should raise risk unless the full command is a known safe pattern.

The first implementation should use a conservative tokenizer plus prefix rules. The parser should be replaceable by a tree-sitter based Bash/PowerShell parser later, following the OpenCode pattern for extracting command nodes, path arguments, cwd changes, redirects, and external directories.

The detector should be deterministic and unit-tested. It is not a full shell parser; it should return `unknown` or `deny` when it cannot classify safely.

## Exec Policy

Add a small Codex-style exec policy layer above the detector.

Policy shape:

```text
prefix_rule(pattern=["git", "status"], decision="allow")
prefix_rule(pattern=["npm", "run", "test:core"], decision="allow")
prefix_rule(pattern=["npm", "install"], decision="ask")
prefix_rule(pattern=["git", "push"], decision="ask")
prefix_rule(pattern=["rm", "-rf", "/"], decision="deny")
```

Initial implementation can be Go structs or JSON/YAML, not a DSL:

```go
type CommandPolicyRule struct {
    Pattern       []string
    Decision      string // allow, ask, deny
    RiskLevel     string
    Category      string
    NetworkPolicy string
    Justification string
}
```

Evaluation rules:

- exact executable/prefix match first.
- deny is strictest, then ask, then allow.
- unknown commands default to ask for `bash` and deny for `run_tests`.
- hardline deny rules are not overrideable by full-access mode or remembered approval.
- rules may produce a proposed approval key but may not execute a command themselves.

The exec policy should eventually support host executable metadata for absolute command paths, but Phase 3 can reject absolute executable paths unless they resolve to known system shells or known toolchain executables.

## Mature Bash Capability Model

Phase 3 should implement a real bash capability model instead of a single all-powerful `bash` switch.

Capabilities:

- `shell.exec.foreground`: bounded non-interactive command, default path for tests and diagnostics.
- `shell.exec.background`: tracked background process with process ref, poll/wait/kill, retained output, and completion state.
- `shell.exec.pty`: interactive terminal process with PTY output framing and explicit user-visible attachment state.
- `shell.stdin`: write stdin to a running process or foreground command.
- `shell.env.override`: pass approved env overrides.
- `shell.cwd.external`: run in an external directory.
- `shell.sudo`: attempt privileged command execution.
- `shell.network`: allow network-classified command execution.

Each capability can independently evaluate as allow, ask, or deny. A command that needs multiple capabilities must satisfy all of them; the strictest decision wins.

### Background process requirements

Background mode is production-ready only if it provides:

- durable process record with session id, turn id, command, cwd, backend, pid/process id, start time, and status.
- `poll`, `wait`, `kill`, and `read_output` service operations.
- output retention independent of the originating model turn.
- cleanup on timeout, app shutdown, session close, and explicit user kill.
- clear ownership when multiple sessions use the same workspace.
- no shell tricks such as `cmd &` as a substitute for the registry.

### PTY and stdin requirements

PTY support is useful for real tools, but it must be explicit:

- PTY commands are displayed as attached interactive processes, not ordinary completed tool calls.
- PTY output is framed and bounded to protect UI rendering.
- stdin writes are separate auditable actions.
- model-supplied stdin is allowed only after policy approval; user-entered secrets must never be echoed into model-visible history.
- commands that wait for input without PTY/stdin permission should time out with a clear diagnostic.

### Sudo requirements

Sudo is a separate privileged capability, not a normal shell command.

- `sudo`, `su`, privilege-escalation wrappers, and direct writes to system paths are high-risk or hardline-denied depending on target.
- `sudo -S`, password piping, and password guessing are denied.
- If sudo prompt support is enabled, only the user can enter the password through secure UI; the value is never persisted in conversation, tool args, logs, or model-visible output.
- Password caching is off by default. If added later, cache scope must be session-local, short-lived, encrypted where practical, and manually clearable.
- Sudo approval does not imply arbitrary root shell approval; the approval key includes the exact command, cwd, backend, and target paths.

### Env override requirements

Env overrides are allowed only when all keys pass policy:

- safe build toggles may be allowed, for example `CI`, `NODE_ENV`, `GOFLAGS`, or cache directories under workspace/home.
- dangerous behavior-changing variables require ask or deny, for example `PATH`, `SHELL`, `NODE_OPTIONS`, `PYTHONPATH`, `LD_PRELOAD`, `DYLD_*`, `GIT_*`, `NPM_CONFIG_*`.
- secret-like keys are denied unless a future dedicated credential-passthrough permission exists.
- env overrides appear in approval metadata with values redacted when sensitive.

### External cwd requirements

External cwd support must be explicit:

- cwd outside workspace triggers `shell.cwd.external`.
- approval displays the resolved absolute path and whether it will be read-only or read/write.
- saved approvals bind to the exact external directory by default.
- external cwd does not expand file tool permissions; file tools still enforce their own path policy.

## User Terminal Panel

The conversation bottom shell is a user-facing terminal surface, not a model-facing tool call. It should feel like OpenCode's terminal panel while fitting Aivo's Electron, React, and Go runtime architecture.

### Product contract

The bottom shell must support:

- open, close, and toggle from the conversation composer area.
- keyboard shortcut support, with a default such as `Ctrl+\`` where it does not conflict with platform conventions.
- vertical resize with persisted height and a sensible minimum, maximum, and collapse threshold.
- multiple terminal tabs per workspace.
- create, close, switch, rename, and reorder terminal tabs.
- automatic first terminal creation when the panel is opened and no terminal exists.
- visible process status: connecting, connected, exited, reconnecting, failed, and closed.
- copy, paste, text selection, link opening with modifier key, and normal shell keyboard behavior.
- retained visible buffer when switching sessions in the same workspace.
- graceful behavior when the active conversation changes, the workspace changes, or the core reconnects.

Terminal state should be workspace-scoped rather than conversation-scoped. If the user switches between conversations for the same project, the terminal panel should preserve its tabs. If the user switches to another project, Aivo should load that workspace's terminal state.

### Backend service

Add a dedicated PTY service in `core/app` or a nearby runtime package. It should not be a thin wrapper around `SandboxRunner.Run`; PTY sessions have a different lifecycle than one-shot commands.

Service API:

```go
type TerminalService interface {
    List(ctx context.Context, workspaceRoot string) ([]TerminalInfo, error)
    Create(ctx context.Context, input TerminalCreateInput) (TerminalInfo, error)
    Get(ctx context.Context, workspaceRoot string, terminalID string) (TerminalInfo, error)
    Update(ctx context.Context, input TerminalUpdateInput) (TerminalInfo, error)
    Remove(ctx context.Context, workspaceRoot string, terminalID string) error
    Attach(ctx context.Context, input TerminalAttachInput) (TerminalAttachment, error)
}

type TerminalInfo struct {
    ID            string
    WorkspaceRoot string
    Title         string
    Command       string
    Args          []string
    CWD           string
    Status        string // running, exited, removed
    PID           int
    ExitCode      *int
    TimeCreated   time.Time
    TimeUpdated   time.Time
}

type TerminalCreateInput struct {
    WorkspaceRoot string
    CWD           string
    Title         string
    Shell         string
    Env           map[string]string
    Rows          int
    Cols          int
}

type TerminalUpdateInput struct {
    WorkspaceRoot string
    TerminalID    string
    Title         string
    Rows          int
    Cols          int
}

type TerminalAttachInput struct {
    WorkspaceRoot string
    TerminalID    string
    Cursor        int64 // -1 means tail from current end
    OnData        func([]byte)
    OnExit        func(exitCode int)
}

type TerminalAttachment interface {
    Replay() []byte
    Cursor() int64
    Write([]byte) error
    Activate()
    Detach()
}
```

Required backend behavior:

- use a real PTY implementation on supported platforms, for example `creack/pty` or an equivalent maintained Go PTY package.
- spawn the user's configured shell as a login or interactive shell where appropriate.
- cwd must resolve under workspace root unless future external-terminal-cwd support is explicitly added.
- environment must be sanitized using the same secret-deny policy as command tools, with terminal-specific values such as `TERM=xterm-256color` and `AIVO_TERMINAL=1`.
- keep an in-memory retained output ring buffer per terminal, with a hard byte cap.
- maintain an absolute output cursor so reconnecting clients can replay from a known point.
- preserve exited sessions long enough for UI visibility, but cap retained exited sessions and buffers.
- kill the PTY process group on remove, workspace close, app shutdown, and service shutdown.
- publish lifecycle events: `terminal.created`, `terminal.updated`, `terminal.exited`, and `terminal.removed`.
- ensure terminal ids are opaque and scoped to workspace ownership checks.

### HTTP and WebSocket transport

Expose explicit local API endpoints through the Go HTTP transport:

- `GET /api/terminals?workspaceRoot=...`
- `POST /api/terminals`
- `GET /api/terminals/{id}`
- `PATCH /api/terminals/{id}`
- `DELETE /api/terminals/{id}`
- `POST /api/terminals/{id}/connect-token`
- `GET /api/terminals/{id}/connect?workspaceRoot=...&cursor=...&ticket=...`

The exact routing can use Aivo's RPC style for CRUD operations, but PTY attach should use WebSocket or an equivalent bidirectional streaming transport. A one-shot RPC loop is not acceptable for mature terminal behavior.

WebSocket protocol:

- outbound terminal data frames are raw UTF-8/binary PTY output chunks.
- outbound control frames use a reserved prefix, for example byte `0x00` followed by JSON such as `{"cursor":12345}`.
- inbound frames are terminal input bytes/text.
- resize is sent either through `PATCH /api/terminals/{id}` or a reserved control frame; the implementation should choose one path and make it deterministic.
- replay is chunked to bounded frame sizes.
- abnormal disconnect should not kill the PTY; explicit remove or process exit should.
- attach should fail with 404 for unknown/exited terminals and 403 for invalid origin/ticket.

Security requirements:

- the connect token endpoint should require a non-simple request header or another CSRF-resistant mechanism before issuing a ticket.
- WebSocket attach must validate workspace ownership, terminal id, ticket, and origin.
- tickets are single-use, short-lived, and bound to workspace root plus terminal id.
- no provider credentials or app auth tokens are forwarded into terminal env by default.
- terminal output and input are never appended to conversation history automatically.

### Frontend architecture

Add terminal UI outside `components/ui`:

- `apps/desktop/src/features/projects/terminal/terminal-panel.tsx`
- `apps/desktop/src/features/projects/terminal/terminal-view.tsx`
- `apps/desktop/src/features/projects/terminal/terminal-state.ts`
- `apps/desktop/src/services/terminal.ts`

Recommended renderer dependency:

- use `@xterm/xterm` plus `@xterm/addon-fit` for the first Aivo implementation.
- add `@xterm/addon-web-links` only if link behavior is needed beyond a custom click handler.
- do not use a plain `<textarea>` terminal; it will not provide mature shell behavior.

The panel should integrate with `ProjectSelectionScreen` below the timeline and above the composer, similar to OpenCode's `TerminalPanel`. Composer and timeline bottom spacing must account for panel height so messages, permission docks, and the input box do not overlap.

State model:

```ts
type WorkspaceTerminalState = {
  opened: boolean
  height: number
  activeId?: string
  tabs: Array<{
    id: string
    title: string
    titleNumber: number
    rows?: number
    cols?: number
    cursor?: number
    scrollY?: number
    bufferSnapshot?: string
    status?: "connecting" | "running" | "exited" | "failed"
  }>
}
```

Persist only UI state and safe buffer snapshots. The source of truth for running PTY processes remains the Go core.

### Context boundary with the agent

The user terminal and model `bash` tool share visual concepts, but not authority:

- terminal input is user authority.
- model `bash` execution is agent authority and must pass command policy and permission checks.
- terminal output is private UI state until the user explicitly attaches selected output or asks the agent to use it.
- an assistant turn may reference that a terminal exists only through explicit user action or structured future context, not by scraping terminal buffers.
- if a future "send terminal output to agent" command is added, it must show exactly what text will enter the conversation and apply size limits.

### Production acceptance criteria

The user terminal panel is complete only when all of these are true:

- panel opens from the conversation bottom and creates a working PTY if none exists.
- multiple tabs can be created, closed, switched, renamed, and reordered without losing the active process.
- resize changes both panel height and PTY rows/cols, with debounced backend updates.
- WebSocket reconnect replays from cursor without duplicating large chunks.
- terminal process exit updates UI state and keeps final output visible.
- explicit close kills or removes the PTY and clears the tab.
- switching conversations in the same workspace preserves terminal tabs.
- switching workspaces loads a different terminal state.
- app shutdown and core shutdown clean up running PTY processes.
- copy, paste, selection, and common shell control keys behave correctly.
- terminal output cannot push layout outside bounds or overlap the composer/permission dock.
- invalid ticket, missing terminal, backend crash, and reconnect exhaustion each show distinct UI states.
- no terminal buffer, input, env secret, or secure prompt value is sent to the model automatically.

## Permission Integration

Extend `PermissionEngine` without bypassing existing saved approvals:

- Add `permissionActionShell = "shell"` and `permissionActionTest = "test"`.
- Update `permissionActionForSpec` so `shell.exec` maps to `shell` and `shell.test` maps to `test`.
- Extend `permissionPathsForTool` into a broader resource extractor, or add a parallel function that returns paths plus command metadata.
- Store command metadata in `PermissionRequest.Arguments`: command, cwd, category, risk level, network hint, timeout, backend, and detector reason.
- Continue storing path data in `PermissionRequest.Paths` when commands statically target workspace paths.
- Saved approvals for shell commands should match an exact approval key by default, not only wildcard paths.

Approval key fields:

- workspace root or stable workspace id/hash
- session id when approval scope is session-only
- normalized cwd
- normalized command text
- canonical argv/prefix tokens when available
- tool name: `bash` or `run_tests`
- requested shell capabilities: foreground, background, pty, stdin, env override, external cwd, sudo, network
- backend: local, docker, or future backend
- sandbox profile and network policy
- command policy decision and risk level
- extracted path patterns and external-directory scope

Permission defaults:

- request-approval mode: ask for `bash`; ask for `run_tests` unless the command is a known read/test command with an exact saved approval.
- full-access mode: allow known test/read commands; ask for arbitrary `bash`; deny detector-blocked commands.
- saved deny always wins.
- hardline deny always wins.

This avoids turning "full access" into unbounded shell access while preserving fast iteration for tests.

Approval prompts should show enough context for a safe decision:

- command and cwd
- requested capabilities
- target tool and backend
- risk category and detector explanation
- path patterns and external directories
- network policy
- timeout
- env override keys with sensitive values redacted
- stdin/PTY/background state when requested
- exact scope of "remember" if selected

Remembered shell approvals should default to exact command plus cwd. Wildcard command approvals are out of scope for Phase 3.

## Environment Policy

The sandbox runner should construct a clean environment from an allowlist. It should not forward the parent environment wholesale.

Initial allowlist:

- `PATH`
- `HOME`
- `USER`
- `LOGNAME`
- `SHELL`
- `TMPDIR`
- `TEMP`
- `TMP`
- `LANG`
- `LC_*`
- `TERM`
- `CI`
- Go cache variables: `GOCACHE`, `GOMODCACHE`, `GOPATH`
- package-manager cache variables only when already set and under the user's home or workspace: `NPM_CONFIG_CACHE`, `PNPM_HOME`, `YARN_CACHE_FOLDER`

Always deny known secret-bearing names, even if they match another pattern:

- names containing `KEY`, `TOKEN`, `SECRET`, `PASSWORD`, `CREDENTIAL`, `COOKIE`, `SESSION`, or `AUTH`
- provider credentials such as `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GITHUB_TOKEN`, `GOOGLE_API_KEY`, and similar names
- Aivo desktop/core session tokens, OAuth refresh tokens, local auth stores, provider-specific API keys, and gateway credentials

The runner may set safe values such as `AIVO_SANDBOX=local` and `CI=1`. Tool arguments must not support arbitrary unapproved env overrides; approved overrides are evaluated key-by-key through `shell.env.override`.

If a build genuinely needs credentials, Phase 3 should fail with a clear message unless a dedicated credential-passthrough permission exists. Credential passthrough must not be hidden inside generic `bash` or generic env override approval.

## Process Lifecycle

The command lifecycle should be explicit:

1. Build command request.
2. Detect and classify command.
3. Evaluate exec policy.
4. Evaluate permission and maybe wait for approval.
5. Build sanitized environment.
6. Start sandbox/backend process.
7. Register process if background or PTY mode is requested.
8. Stream bounded output updates.
9. Enforce timeout/cancellation.
10. Cleanup process tree or leave a tracked background process.
11. Persist final result or process ref.

Cancellation sources:

- context cancellation from the agent turn
- user cancellation from UI
- session cancellation
- app shutdown
- command timeout
- future network denial cancellation

The runner must distinguish `exitCode != 0`, timeout, cancellation, spawn failure, and cleanup failure. These are different user-facing states and different model feedback.

## Output And Audit

Command tools should return model-friendly summaries and structured metadata.

`ToolResult.Content`:

- short command summary
- exit code
- stdout/stderr snippets
- timeout/truncation notice

`ToolResult.Structured`:

- `command`
- `argv`
- `mode`
- `cwd`
- `exitCode`
- `stdout`
- `stderr`
- `timedOut`
- `cancelled`
- `durationMs`
- `backend`
- `networkPolicy`
- `approvalKey`
- `policyDecision`
- `riskLevel`
- `truncated`
- `originalSize`
- `stdoutRef`
- `stderrRef`
- `processRef`

`recordToolResult` already persists structured tool result maps, so Phase 3 can avoid a new database table initially. The existing `EventTypeShellCommand` and `EventTypeShellOutput` constants can be used later if the timeline needs separate shell events.

Large output should be handled like this:

- keep bounded stdout/stderr snippets in `ToolResult.Content` and `ToolResult.Structured`
- write complete stdout/stderr to retained output files under an Aivo-managed session artifact directory
- store artifact refs, size, and retention metadata in the tool result
- never pass complete huge output back to the model automatically
- redact known secret patterns before UI/model summaries where feasible

## Desktop UI Impact

The existing permission dock and tool-call timeline can be extended instead of replaced for model tool execution. The user terminal panel is a new bottom work surface in the same conversation layout.

Required UI behavior:

- Permission cards show command, cwd, category, risk, and touched paths if any.
- Remembered approvals for shell commands make clear they apply to the exact normalized command and cwd.
- Tool timeline renders command rows differently from file tools: command text, status, exit code, duration, stdout, stderr, and timeout.
- Command rows expose retained-output links when output was truncated.
- Command rows distinguish denied, rejected, timeout, cancelled, failed exit, spawn failure, and cleanup warning.
- Background command rows include process ref, poll/wait/kill controls, and completion state.
- PTY command rows include attached/detached state and stdin affordance when allowed.
- Failed commands should be visible as failed tool calls, not hidden assistant errors.
- Pending shell approvals should pause the turn the same way pending patch approvals do today.
- The terminal panel opens below the timeline and above the composer, without covering permission approvals or submitted messages.
- Terminal tabs show title, connection/process state, and close controls.
- The active terminal renders a real PTY surface with copy, paste, selection, scrollback, resize, and reconnect.
- Terminal panel state is restored per workspace after route changes and app reload.
- Terminal failures distinguish startup failure, attach failure, reconnect exhaustion, process exit, and explicit user close.

New HTTP/RPC and WebSocket transport support is required for the user terminal panel. The model tool timeline can continue using the existing RPC/event infrastructure.

## Implementation Sequence

1. Add sandbox request/result types, retained-output refs, and structured sandbox errors.
2. Add sanitized environment builder with secret-deny tests.
3. Add cwd enforcement and workspace-root validation.
4. Add `LocalSandboxRunner` with stdout/stderr capture, timeout caps, process-group cleanup, and cancellation tests.
5. Add managed process registry for background/PTY process refs.
6. Add production PTY service with create/list/get/update/remove/attach, retained buffer cursor, process cleanup, and terminal lifecycle events.
7. Add terminal HTTP/RPC endpoints and WebSocket attach transport with origin/ticket validation.
8. Add terminal frontend service, workspace terminal state, and `@xterm/xterm` renderer integration.
9. Add bottom `TerminalPanel` with resize, multi-tab create/close/switch/rename/reorder, reconnect, and workspace persistence.
10. Add command detector and prefix exec policy with table-driven tests.
11. Extend permission action/resource extraction for shell and test commands.
12. Add exact shell approval keys and remembered approval matching.
13. Add `RunTestsTool`, known command mapping, and registration.
14. Add UI command timeline for `run_tests` and `bash` tool calls.
15. Add `BashTool` foreground mode.
16. Add background and PTY mode behind explicit capability checks.
17. Add env override, stdin, sudo, and external cwd capability checks.
18. Register mature `bash` in `NewCodingToolRegistry` only after the Phase 3 acceptance suite passes.
19. Add Docker backend interface/stub and unavailable-state UI.
20. Add optional Docker implementation behind config/feature flag when production backend requirements are satisfied.

Internal development may land these pieces incrementally, but the user-facing bottom shell should remain hidden until the PTY service, transport, terminal panel, cleanup, reconnect, persistence, and security checks all pass. Phase 3 should not ship a partial terminal as a completed feature.

## Testing Plan

Core unit tests:

- detector classifies known read/test/build/network/dangerous commands
- detector rejects absolute paths outside the workspace
- exec policy applies strictest decision across matching rules
- hardline deny cannot be overridden by full-access mode or saved approvals
- approval key changes when command, cwd, backend, sandbox profile, or network policy changes
- env builder forwards allowlisted values and drops secret-like variables
- cwd normalization rejects traversal and absolute external paths
- local runner returns stdout, stderr, exit code, duration, and truncation metadata
- local runner kills long-running commands on timeout
- local runner kills child process groups, not only the direct shell process
- retained output files are created for oversized stdout/stderr
- background process can be polled, waited, killed, and cleaned up
- PTY process output is bounded and stdin writes are auditable
- env override denies secret keys and asks/denies behavior-changing keys
- external cwd requires exact external-directory approval
- sudo command requires sudo capability and denies password piping
- `bash` asks for permission and executes after approval
- `bash` denies detector-blocked commands before asking
- saved approval only matches exact normalized command and cwd
- `run_tests` maps known targets to expected commands
- `run_tests` rejects unsupported target/kind/filter combinations

Service integration tests:

- pending shell approval pauses the turn and persists the tool call as `pending_approval`
- approved shell command resumes through the existing permission flow
- failed command is persisted as a failed tool call with stdout/stderr metadata
- timeout is persisted as a failed tool call with `timedOut=true`
- cancellation is persisted as a cancelled/failed tool call with `cancelled=true`
- denied hardline command creates a deterministic `permission_denied` or `command_denied` result and no approval request
- full-access mode allows known tests but still asks for arbitrary `bash`
- full-access mode does not imply sudo, external cwd, secret env, or untracked background process approval

Desktop verification:

- permission dock renders command approval details
- timeline renders running, success, failure, pending, and timeout command states
- command stdout/stderr text stays bounded and does not break layout
- retained output link is visible when command output is truncated
- remembered approval copy states exact command/cwd scope
- background and PTY controls are visible only for matching process modes
- terminal panel opens, closes, toggles, and resizes without overlapping the composer or permission dock
- terminal tabs create, close, switch, rename, and reorder correctly
- active terminal sends stdin and receives PTY output through WebSocket
- terminal resize propagates rows/cols to the backend PTY
- reconnect resumes from cursor without duplicate replay
- workspace switching restores the correct terminal state
- terminal error states render distinctly and recover where possible

Manual production checks:

- run a command that spawns a sleeping child and verify timeout removes both parent and child
- run a command that emits more than the output limit and verify artifact retention
- attempt to echo known secret env vars and verify they are absent
- attempt dangerous commands and verify hardline denial happens before approval
- attempt `sudo -S`, secret env override, and external cwd without approval and verify denial/ask behavior
- start a background command, poll it, kill it, and verify no orphan process remains
- quit or cancel during a long command and verify cleanup and persisted state
- open the bottom terminal, run an interactive shell command, resize the panel, reload the app, and verify attach/replay behavior
- create multiple terminal tabs, close one running tab, and verify the process is removed without affecting other tabs
- disconnect the WebSocket or restart the renderer and verify the core-side PTY survives until explicit close or process exit
- verify terminal output is not added to the model conversation unless explicitly attached by the user

## Rollout Strategy

1. Land backend interfaces, env builder, output retention, local runner, PTY service, and terminal transport behind tests.
2. Land detector, exec policy, hardline deny, approval keys, permission integration, and tool timeline UI.
3. Land the complete bottom terminal panel behind an internal feature flag until production acceptance passes.
4. Expose `run_tests` only after command policy, permission UI, timeout cleanup, retained output, and failure rendering are complete.
5. Expose `bash` only after foreground, background, PTY, stdin, env override, sudo policy, external cwd policy, and UI states have dedicated tests and security checks.
6. Expose the user terminal panel only after multi-tab, resize, reconnect, replay, cleanup, persistence, copy/paste, error states, and context-boundary checks pass.
7. Keep Docker backend unavailable but compiled against the interface to protect the abstraction.
8. Enable Docker behind explicit config only after resource limits, labels, env filtering, cwd mapping, and orphan cleanup are implemented.
9. Tighten command rules based on real usage before Phase 4 mode/toolset work.

Rollout can be internally staged, but there should be no public "MVP terminal" milestone. User-facing release means production-ready.

## Open Decisions

- Whether full-access mode should ever allow arbitrary `bash`; the recommended Phase 3 default is no.
- Whether `run_tests all build` should run both `pnpm test:core` and `pnpm build`, or keep build and test explicit.
- Whether command approval rules need a new database shape for command resource keys, or whether `PermissionRequest.Arguments` plus scoped `PermissionRule` metadata is enough for the first implementation.
- Whether local runner should set `HOME` to the real home for toolchain cache compatibility or a workspace-local synthetic home for stronger isolation.
- Whether Phase 3 should include tree-sitter shell parsing immediately or start with conservative prefix tokenization and add tree-sitter in a follow-up.
- Whether retained command output should live in SQLite blobs, session artifact files, or a dedicated artifact table.
- Whether local backend should reject all network-classified commands by default until a platform-enforced network sandbox exists.
- Whether sudo support ships in Phase 3 by default or behind an advanced local-only feature flag.
- Whether model-driven PTY tool calls should ship in Phase 3, but the user terminal panel itself requires HTTP/WebSocket transport.
- Whether terminal state should persist buffer snapshots in local storage, SQLite, or only retain buffer in core memory with UI metadata persisted locally.
- Whether terminal tab rename should be user-only or auto-update from shell-reported title sequences.
