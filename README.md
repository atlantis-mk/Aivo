# Aivo Monorepo

This repository contains the migrated Aivo product split into browser and Electron applications, with shared shadcn-style UI primitives.

## Layout

- `apps/web` — browser application.
- `apps/electron` — Electron application migrated from `Aivo-old/apps/desktop`.
- `packages/ui` — shared shadcn-style UI components and theme tokens.
- `vendor/codex` — Git submodule for the [Codex repository](https://github.com/atlantis-mk/codex). Its Rust runtime is referenced in place and was not copied from the old workspace.

## Commands

```bash
pnpm install
pnpm dev:web
pnpm dev:electron
pnpm build
```

The Electron development flow builds the Codex runtime from `vendor/codex/codex-rs` before launching.
