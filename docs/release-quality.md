# Release Quality

Aivo release verification is split between local smoke checks, manually dispatched package diagnostics, and tag-driven native release publication. Public source is licensed under `PolyForm-Noncommercial-1.0.0`; package-registry `private: true` metadata does not change that license.

## Local Gates

Run the normal quality gates before shipping:

```bash
npm run test:core
npm run lint
npm run build
npm run package -- --dir
npm run smoke:release
```

`npm run smoke:release` always rebuilds the Go core first, then starts the freshly built `build/aivo-core/aivo-core` binary on an operating-system-assigned loopback port, validates its versioned readiness endpoint, and waits for `/health`. When a packaged app exists for the current platform, the smoke also verifies that the app bundle contains the core binary.

## Manual CI packaging

`.github/workflows/release-quality.yml` is manually dispatched and runs a three-platform matrix:

- macOS: `npm run package:mac`
- Windows: `npm run package:win`
- Linux: `npm run package:linux`

Each job runs `test:core`, `lint`, `build`, full platform packaging, `smoke:release` with `AIVO_EXPECT_INSTALLERS=1`, and uploads the generated installer artifacts.

## Tagged release publication

`.github/workflows/publish-release.yml` runs when a `v<SemVer>` tag is pushed or when the operator manually dispatches it for an existing tag. The tag version must match both package manifests and must have a same-name release record under `releases/`. That record must contain exactly one non-empty H1; the workflow uses it as the GitHub Release title and the complete file as the Release body. Bilingual highlights, the system/chip/format download table, integrity guidance, and installation warnings precede the retained internal release record. The operator owns release readiness, so this workflow does not run documentation, script, Core test, lint, desktop build, package-smoke, or native-handoff quality gates before publication. It directly builds native packages on these GitHub-hosted targets:

- macOS Apple Silicon on `macos-15`
- macOS Intel on `macos-15-intel`
- Windows x64 on `windows-2025`
- Linux x64 on `ubuntu-24.04`

Each platform job normalizes its package names. The publication job requires six packages—DMG and ZIP for each macOS architecture, an NSIS setup executable for Windows x64, and an AppImage for Linux x64—plus `SHA256SUMS`.

R2 publication uses immutable `aivo/releases/v<version>/` objects by default. An existing object is reused only when its `sha256` metadata matches; missing or different digest metadata refuses the release. `aivo/channels/stable/latest.json` is uploaded only after all immutable objects and is verified by authoritative readback. The GitHub Release is created or resumed only after R2 succeeds, and every existing GitHub asset must have the same SHA-256 digest.

Required GitHub Actions secrets:

- `R2_ACCESS_KEY_ID`
- `R2_SECRET_ACCESS_KEY`

Required GitHub Actions variables:

- `R2_ACCOUNT_ID`
- `R2_BUCKET`
- `R2_PUBLIC_BASE_URL`

Optional `R2_PREFIX` defaults to `aivo`. R2 credentials should be scoped to the chosen bucket and the Aivo prefix. Release jobs never print credentials.

## Desktop update channel

Packaged desktops check the public `aivo/channels/stable/latest.json` object after startup and on explicit user request. Electron main derives the normalized package name for its own platform/architecture and accepts only a newer stable SemVer. Before downloading, it requires the manifest entry to use the immutable `aivo/releases/v<version>/` R2 HTTPS path and cross-checks the exact name, size, and SHA-256 against the same-tag non-draft GitHub Release in `atlantis-mk/Aivo`.

The downloaded bytes are streamed into an Aivo-owned temporary directory, bounded by the declared size and a hard package limit, and rechecked for exact size and SHA-256. Only then may a native user action open the DMG or NSIS package, or reveal the Linux AppImage. This update flow does not make unsigned artifacts signed, does not bypass OS warnings, and does not silently install or downgrade. A release is update-ready only after `GATE-8` publication consistency and `GATE-9` target-platform handoff acceptance pass.

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

Windows code-signing and Linux distribution signing are not yet configured. A tagged package cannot be described as a fully signed stable release until the applicable target-OS signing evidence is recorded in its Release record.
