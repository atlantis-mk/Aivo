# Release Quality

Aivo release verification is intentionally split between local smoke checks and CI platform packaging.

## Local Gates

Run the normal quality gates before shipping:

```bash
npm run test:core
npm run lint
npm run build
npm run package -- --dir
npm run smoke:release
```

`npm run smoke:release` always rebuilds the Go core first, then starts the freshly built `build/aivo-core/aivo-core` binary and waits for `/health`. When a packaged app exists for the current platform, the smoke also verifies that the app bundle contains the core binary.

## CI Packaging

`.github/workflows/release-quality.yml` runs on pull requests and pushes to `main` with a three-platform matrix:

- macOS: `npm run package:mac`
- Windows: `npm run package:win`
- Linux: `npm run package:linux`

Each job runs `test:core`, `lint`, `build`, full platform packaging, `smoke:release` with `AIVO_EXPECT_INSTALLERS=1`, and uploads the generated installer artifacts.

## macOS Signing And Notarization

The desktop package config enables hardened runtime, entitlements, signing, and electron-builder notarization. Unsigned local builds are allowed for development, but release CI should provide signing and notarization credentials.

Preferred App Store Connect API key secrets:

- `MACOS_CSC_LINK`
- `MACOS_CSC_KEY_PASSWORD`
- `APPLE_API_KEY`
- `APPLE_API_KEY_ID`
- `APPLE_API_ISSUER`

Apple ID fallback secrets:

- `MACOS_CSC_LINK`
- `MACOS_CSC_KEY_PASSWORD`
- `APPLE_ID`
- `APPLE_APP_SPECIFIC_PASSWORD`
- `APPLE_TEAM_ID`

If no valid Developer ID identity is available, electron-builder prints a skipped-signing warning. That is acceptable for local `--dir` development builds, but not for signed release artifacts.
