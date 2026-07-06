# Aivo

Aivo is a local-first Electron desktop workbench for code-to-delivery tasks.

## Requirements

- Go 1.23+
- Electron CLI 2.12+
- Node.js 24+
- pnpm 11+

## Setup

```bash
cd frontend
pnpm install
```

## Development

```bash
pnpm dev
```

## Checks

```bash
go test ./...
cd frontend && pnpm typecheck
cd frontend && pnpm build
```

## Desktop Build

```bash
pnpm build
```

The Electron app uses Go module `aivo`, bundle identifier `com.aivo.desktop`, a React + TypeScript + Vite frontend, and pnpm for frontend package management.
