# Aivo Monorepo

This repository contains the migrated Aivo product split into browser and Electron applications, with shared shadcn-style UI primitives.

## Layout

- `apps/web` — browser application.
- `apps/electron` — Electron application migrated from `Aivo-old/apps/desktop`.
- `packages/ui` — shared shadcn-style UI components and theme tokens.
- `vendor/codex` — Git submodule for the [Codex repository](https://github.com/atlantis-mk/codex), retained for source-level reference and optional local builds.

## Commands

```bash
pnpm install
pnpm dev:web
pnpm dev:electron
pnpm build
```

The Electron development and release packaging flows resolve the latest published stable [OpenAI Codex release](https://github.com/openai/codex/releases), verify each asset's SHA-256 checksum, and use native binaries from `apps/electron/.aivo-runtime`. They do not download prereleases.
