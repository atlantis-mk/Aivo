# Four primitive Agent tools

## Core contract

The default Agent receives exactly four tools:

| Tool | Question answered | Mutates |
| --- | --- | --- |
| `read` | What does this file currently contain? | No |
| `bash` | What should the development environment do? | Command-dependent |
| `edit` | Which exact parts of this file should change? | Yes |
| `write` | What should this complete file contain? | Yes |

The tools are bound to one `ExecutionEnvironment`. Local, container, SSH, and remote implementations must provide a coherent filesystem, working directory, process runner, temporary-artifact store, mutation queue, and cancellation boundary.

## `read`

```ts
read({
  path: string,
  offset?: number,
  limit?: number
})
```

- Reads exactly one file; directories fail with `not_file`.
- Relative paths resolve from the active environment workspace; absolute paths use that environment's path namespace.
- For text, `offset` is a 1-based starting line and `limit` is a maximum line count.
- System byte and line limits always apply; callers may request less but not more.
- A partial response reports the next offset and whether more content remains without requiring an unbounded full-file scan.
- Supported images return image model content and bounded metadata. Offset and limit are invalid for images. Image bytes, decoded pixels, and rendered dimensions are capped; large images are resized for model input.
- Unsupported binary files fail explicitly.
- The tool does not list directories, search, filter, modify, or infer a path.

## `bash`

```ts
bash({
  command: string,
  timeout?: number
})
```

- Runs one foreground, non-interactive Bash command from the active environment workspace.
- `timeout` is measured in seconds and is capped by typed runtime configuration.
- Each call has independent shell state. Callers use ordinary Bash composition such as `cd subdir && command`.
- Returns separate stdout, stderr, exit status, timeout/cancellation state, duration, and bounded output metadata.
- When output exceeds the model limit, the result preserves a deterministic useful tail and writes complete stdout/stderr to private session temporary files in the same environment. The paths can be read with `read`.
- Temporary artifacts have private permissions where supported, belong to the session, are removed on session cleanup, and are reclaimed after stale-process recovery.
- Timeout or cancellation terminates the owned process tree and closes streams.
- PTY, background jobs, follow-up stdin, persistent shell state, and background process management are extension responsibilities.
- `bash` means Bash-compatible syntax. It must not silently become PowerShell or `cmd.exe`.

## `edit`

```ts
edit({
  path: string,
  edits: Array<{
    oldText: string,
    newText: string
  }>
})
```

- Reads one original text snapshot and validates every edit against that snapshot.
- Each `oldText` is non-empty, byte-for-byte exact, and occurs exactly once.
- Matching is not fuzzy and does not normalize whitespace or line endings.
- All matched byte ranges are computed before modification; duplicate or overlapping ranges fail the complete call.
- Later edits cannot depend on content produced by earlier edits in the same call.
- The complete mutation participates in the environment's per-file queue, rechecks external change where supported, applies replacements from the end of the file, and commits atomically.
- Any validation or write failure leaves the file unchanged.
- The result supplies bounded model text and a structured, bounded unified diff for UI presentation.

Exact failure is preferred over an uncertain successful modification.

## `write`

```ts
write({
  path: string,
  content: string
})
```

- Creates a missing text file or completely overwrites an existing text file.
- Creates missing parent directories.
- Participates in the same per-file mutation queue as `edit`.
- Uses a sibling temporary file and atomic replacement where the environment supports it, with cleanup on failure or cancellation.
- Does not append, patch, merge, accept binary/base64 data, or echo the complete content in its result.
- Returns create/overwrite state, byte count, content digest where available, and bounded UI diff/details.

## Results

Tool results serve the model and UI separately:

```ts
type ToolResult = {
  ok: boolean
  modelContent: Array<TextContent | ImageContent>
  error?: {
    code: string
    message: string
    details?: unknown
  }
  details: {
    summary?: unknown
    artifacts?: ArtifactRef[]
    view?: ExtensionViewRef
  }
  meta: {
    operationId: string
    durationMs: number
    truncated: boolean
  }
}
```

`modelContent` is always bounded. `details` drives native or extension Web UI without requiring the renderer to parse prose. Persisted fallback details remain safe and bounded.

## Existing-tool disposition

| Existing capability | Disposition |
| --- | --- |
| `read_file` | Replace with `read` |
| `write_file` | Replace with `write` |
| `edit_file` | Replace with `edit` |
| `apply_patch` | Remove directly; no alias or shipped replacement |
| `list_files`, `glob`, `search_files` | Remove from default; use `bash` with `ls`, `find`, or `rg` |
| `git_status`, `git_diff` | Remove; use Git through `bash` |
| `run_tests`, `read_diagnostics`, `format_code` | Remove; use project CLI through `bash` |
| `lsp_*` | Optional LSP extension |
| `web_*` | Optional Web extension |
| `exec_command`, `write_stdin` | Optional Terminal extension |
| `agent_*` | Optional sub-Agent extension |
| `automation_*` | Optional Automation extension |
| MCP and plugin tools | Language-neutral dynamic extensions |
| `update_plan`, `ask_user` | Host/UI protocols, not core execution tools |
| `skill` | Context resource mechanism, not a core tool |
| discovery bridge tools | Host pre-call resolver; optional namespaced discovery extension only |

Historical tool calls remain displayable without keeping an executor registered.

## Activation

The Host constructs tools before each primary model request:

```text
core + Agent Mode defaults + session-pinned + warm + auxiliary-selected
```

- `pinned` tools remain until the user removes them.
- `warm` tools are previously used tools retained under a bounded turn lease and LRU cap.
- `currentTurn` tools are selected for the current user turn by the auxiliary resolver.
- Exact lease lengths, caps, and automatic selection maximums are Draft defaults to freeze before Accepted.

The auxiliary resolver receives sanitized eligible catalog summaries, not full extension code, credentials, or unbounded schemas. It returns exact names only. Host validation checks registration, activation policy, platform/config readiness, Agent Mode, and count. Resolver failure falls back to the available core and pinned set.

The default primary model does not see a discovery tool. An explicitly enabled extension may expose `tools.resolve` for specialized autonomous discovery and create a new recorded Tool Snapshot revision.

## Tool Snapshot

Every primary model request freezes:

```ts
type ToolSnapshot = {
  revision: string
  tools: Array<{
    name: string
    registrationId: string
    schemaHash: string
    sourceId: string
    sourceVersion: string
    activationSource: "core" | "mode" | "pinned" | "warm" | "auxiliary"
  }>
}
```

A call executes only when its expected registration matches the frozen snapshot. Extension update, catalog refresh, or environment switch affects a later snapshot and cannot silently change an already generated call.

## Frozen limits and compatibility policy

- Text reads return at most 16,000 UTF-8 characters, default to 500 lines when paged, and accept at most 2,000 requested lines. Supported image input is capped at 20 MiB and 40 megapixels; model images are resized within 2,048 by 2,048 pixels and capped at 8 MiB.
- Bash defaults to 30 seconds and caps requests at 300 seconds. Stdout and stderr are bounded independently to 12,000 characters of useful tail content; complete truncated streams are retained as private artifacts.
- Windows does not silently substitute PowerShell or `cmd.exe` and this Work does not bundle a Bash runtime. The typed runtime configuration may name a Bash-compatible executable; otherwise `bash` is resolved from `PATH` and absence is an actionable `bash_unavailable` failure.
- Warm tools retain a three-turn lease with an eight-tool LRU cap. Automatic auxiliary selection activates at most four tools for one turn.
- Result paths use canonical paths in the active environment namespace with forward slashes for model and persisted metadata. Input spelling is not treated as an identity.
- Session artifacts survive an app restart for recovery, are reclaimed after 24 hours, and are removed immediately on explicit session deletion or normal session cleanup.
- No development-only legacy adapter is shipped. Historical tool names remain display data only and cannot resolve to executors.
