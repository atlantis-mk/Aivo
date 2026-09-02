# Aivo

Aivo is a local-first AI agent desktop workspace built from the open-source
Codex runtime. This repository is an independent fork and is not affiliated
with or endorsed by OpenAI.

## Current status

The repository currently tracks the Codex runtime as its baseline. Aivo's own
multi-provider layer and desktop shell are under active development.

Do not use OpenAI's `codex` installers, npm package, Homebrew cask, desktop
application, or release assets as an Aivo distribution. Aivo releases will be
published from this repository only after its own packaging pipeline exists.

## Build the runtime from source

The current baseline can be built locally as a Rust workspace:

```shell
git clone https://github.com/atlantis-mk/Aivo.git
cd Aivo/codex-rs
cargo build
cargo run --bin codex -- "explain this codebase to me"
```

For toolchain requirements, formatting, and tests, see
[docs/install.md](./docs/install.md). The executable and internal crate names
remain `codex` / `codex-*` for now so upstream synchronization stays
reviewable.

## Run the desktop shell

The Electron desktop shell uses the locally built runtime in development:

```shell
cargo build --manifest-path codex-rs/Cargo.toml --bin codex
pnpm install --frozen-lockfile
pnpm --filter @aivo/desktop dev
```

The shell runs with a sandboxed renderer and exposes only its typed runtime
status controls through preload IPC. It is not yet a distributable installer.

## Development direction

1. Add a maintainable multi-provider abstraction on top of the runtime.
2. Build an Aivo desktop client that communicates with the local app-server.
3. Establish Aivo-owned packaging, signing, and release workflows for desktop
   distributions.

## Resources

- [Contributing](./docs/contributing.md)
- [Licensing in this fork](./AIVO-LICENSE-NOTICE.md)
- [Build instructions](./docs/install.md)
- [Upstream Codex repository](https://github.com/openai/codex)
- [Upstream Codex documentation](https://developers.openai.com/codex)

Codex-derived code remains available under the [Apache-2.0 License](LICENSE).
The separate terms for designated standalone Aivo code are described in
[AIVO-LICENSE-NOTICE.md](./AIVO-LICENSE-NOTICE.md).
