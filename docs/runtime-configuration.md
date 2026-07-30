# Project Runtime Configuration

Aivo loads runtime configuration in this order, with later files overriding map entries from earlier files:

1. `~/.config/aivo/config.json` and `config.jsonc`
2. `~/.aivo/config.json` and `config.jsonc`
3. global `agent(s)/*.md`, `mode(s)/*.md`, and `command(s)/*.md`
4. `<project>/aivo.json` and `aivo.jsonc`
5. `<project>/.aivo/config.json` and `config.jsonc`
6. project `.aivo/agent(s)/**/*.md`, `.aivo/mode(s)/**/*.md`, and `.aivo/command(s)/**/*.md`

Unknown fields, invalid values, symlinked files, files larger than 1 MiB, and provider headers containing literal credentials are rejected with source-located diagnostics. Keep secrets in environment variables and reference their names from configuration.

## Example

```json
{
	"defaultAgent": "project-reviewer",
  "instructions": [
    "docs/assistant/*.md",
    "https://example.com/team-instructions.md"
  ],
  "commands": {
    "review": {
      "description": "Review a path with the project reviewer",
      "template": "Review {{path}} for $focus. Extra context: $ARGUMENTS",
      "arguments": [
        { "name": "path", "required": true },
        { "name": "focus", "default": "correctness" }
      ],
      "agent": "project-reviewer",
	  "toolsets": ["safe", "coding"],
	  "subtask": true
    }
  },
  "agents": {
    "project-reviewer": {
      "description": "Read-only project reviewer",
      "prompt": "Review changes against project rules and report actionable findings.",
      "model": { "providerId": "openai", "modelId": "gpt-5.4" },
      "temperature": 0.2,
	  "topP": 0.9,
	  "mode": "all",
	  "variant": "high",
	  "options": { "reasoning_effort": "high" },
      "maxSteps": 12,
      "toolsets": ["safe"],
      "permissionScope": "read-only"
    }
  },
  "maxParallelChildren": 4,
  "compaction": {
    "auto": true,
    "thresholdPercent": 80,
    "reserveTokens": 4096
  },
  "providerExtensions": {
    "team-openai": {
      "protocol": "openai-compatible",
      "displayName": "Team Gateway",
      "baseUrl": "https://llm.example.com/v1",
      "credentialRef": "TEAM_LLM_API_KEY",
      "models": ["team-code", "team-fast"]
    },
    "local-runner": {
      "protocol": "command",
      "displayName": "Local Runner",
      "command": "/absolute/path/to/provider-adapter",
      "args": ["--stdio"],
      "models": ["local-code"]
    }
  },
  "languageServers": {
    "elixir-ls": {
      "languageIds": ["elixir"],
      "extensions": [".ex", ".exs"],
      "rootMarkers": ["mix.exs", ".git"],
      "command": "elixir-ls",
      "args": [],
      "env": { "MIX_ENV": "dev", "TOKEN": "$ELIXIR_LS_TOKEN" },
      "initializationOptions": {},
      "timeoutSeconds": 10
    }
  }
}
```

Agent and command Markdown files use YAML frontmatter and their body as the prompt/template. Nested paths become slash-separated identifiers:

```markdown
---
name: Project Reviewer
mode: subagent
model: openai/gpt-5.4
toolsets: [safe, git]
top_p: 0.9
---
Review the requested change and report findings first.
```

## Behavior

- Project rules are loaded from global and repository `AGENTS.md` or `CLAUDE.md` files. Nested files apply only when a target file is below their directory. Configured instruction paths support local Globs, `~/` paths, absolute paths, and explicit HTTP(S) URLs; reads are bounded and remote requests time out.
- Type `/review README.md security` in the composer to invoke a configured command. The catalog also includes built-in `init`/`review`, enabled Skills, and MCP Prompts. Commands marked `subtask` run in a forked child session and return the result to the parent.
- Configured agents appear in the project Agent picker. Model, temperature, top-p, provider options, variant, maximum steps, toolsets, permission scope, primary/subagent mode, disabled state, and default Agent are applied per project. Delegate-only batches execute concurrently up to `maxParallelChildren`, retain result order, and propagate cancellation.
- Provider extensions appear only in their project model picker. Same-named extensions in concurrent projects use isolated registries. `credentialRef` is an environment-variable name, never a credential value. Command providers receive one bounded JSON request on stdin and return one `ChatResponse` JSON object on stdout; Aivo does not invoke a shell.
- `RefreshProviderEcosystemCatalog` explicitly refreshes the models.dev-compatible provider/model directory into an atomic offline cache. Native transports remain native; supported OpenAI-compatible entries are added dynamically. Set `AIVO_MODELS_URL` to change the source and `AIVO_MODELS_CACHE` to change the cache file.
- Built-in LSP definitions cover more than 30 common languages/frameworks, including Deno, Vue, Svelte, Astro, Elixir, Zig, Swift, Kotlin, Dart, OCaml, Terraform, Typst, Haskell, and Julia. Servers start at the nearest matching monorepo root; project definitions can add, override, disable, and revision-restart them.
- The Git branch button opens the Worktree manager. It supports automatic Aivo branch names, existing branches, detached worktrees, discovered external worktrees, and explicitly confirmed startup commands. Reset restores the creation base, cleans ignored files and submodules, and removal deletes only an Aivo-owned branch by default. Destructive actions require confirmation and are blocked while a bound session is active.

Effective configuration and source diagnostics are available through the `GetEffectiveRuntimeConfig` desktop RPC. The unified command RPCs are `ListCommandCatalog` and `InvokeCommand`.
