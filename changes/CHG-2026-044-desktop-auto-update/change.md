# Add trusted desktop automatic updates

## Problem or goal

Packaged Aivo releases are published to a stable R2 manifest and matching GitHub Release assets, but an installed desktop cannot discover or obtain them. Add automatic startup checks, an explicit Settings surface, verified background download, and user-confirmed installer handoff for supported packaged targets.

## Expected behavior

`REQ-UPDATE-001` owns the observable flow. A packaged app checks the fixed stable channel once after startup, reports current and available versions, verifies the selected platform asset against both the R2 manifest and the matching GitHub Release, downloads with bounded progress, verifies exact size and SHA-256, then opens or reveals the verified package only after an explicit user action. Failure remains actionable and retryable.

## Non-goals

Silent installation, privilege escalation, downgrade, pre-release channels, delta updates, persistent rollout settings, arbitrary update feeds, and treating an unsigned package as signed are excluded. Linux AppImage replacement and macOS drag-install automation are not performed; the OS-native package flow remains visible to the user.

## Impact

Electron main owns update network access, validation, temporary files, cancellation, and OS handoff. Preload adds a capability-oriented IPC surface and renderer Settings adds update status/actions. Go, persistence/schema, providers, extensions, terminals, worktrees, and public HTTP/RPC are unchanged. No production dependency is added. Release artifacts and the existing stable manifest remain the distribution source.

## Implementation constraints

The renderer MUST NOT provide an update URL, file path, digest, version, or command. Electron main MUST pin the R2 channel, repository identity, platform/architecture asset name, HTTPS origins, maximum manifest/package sizes, timeouts, and one in-flight operation. It MUST compare stable SemVer only, refuse downgrade or ambiguity, cross-check GitHub asset name/size/digest, stream to a Host-owned temporary directory, remove partial files after failure or cancellation, and verify bytes before handoff. App shutdown cancels owned work. Logs and diagnostics contain safe summaries only.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| UPDATE-01 | REQ-UPDATE-001 | Accepted Requirement, ADR, security, release, and Traceability contract | AT-UPDATE-001 | Complete |
| UPDATE-02 | REQ-UPDATE-001 | Main/preload update coordinator with fixed-source validation, download, and handoff | AT-UPDATE-001, CT-SECURITY-001, CT-RELIABILITY-001 | Complete |
| UPDATE-03 | REQ-UPDATE-001, NFR-UI-001 | Responsive Settings state/action surface and startup notification | AT-UPDATE-001, AT-UI-001 | Complete |
| UPDATE-04 | NFR-RELEASE-001 | Focused tests and applicable full gates | AT-UPDATE-001 | In progress: target-OS handoff evidence pending |

## Acceptance and evidence

- Focused tests cover SemVer ordering, platform/architecture selection, manifest/GitHub mismatch refusal, fixed-origin refusal, exact-size/digest verification, progress, repetition, timeout/cancellation cleanup, and renderer inability to choose a URL/path.
- Packaged startup checks are non-blocking; development builds do not automatically contact the channel.
- Settings covers idle, checking, up-to-date, available, downloading, ready, unsupported, and error states at wide and narrow sizes.
- Target-platform package handoff must be accepted on macOS arm64/x64, Windows x64, and Linux x64 before this Work becomes Verified. Unsigned-package OS warnings are expected until signing is configured.

Implementation evidence on 2026-08-25:

- `pnpm docs:check`: passed with 22 Requirements, 24 Test IDs, 22 ADRs, and 45 Work Packages.
- `pnpm scripts:test`: passed 13 governance/release tests, 78 desktop tests including six `AT-UPDATE-001` tests, and four example-extension tests.
- `pnpm test:core`: passed all Go packages.
- `pnpm lint`: passed with only the repository's existing Fast Refresh warnings in shared/generated-adjacent files.
- `pnpm build`: passed TypeScript and Vite production build.
- A live fixed-source metadata check resolved R2 stable `v0.1.0`, selected the macOS x64 asset, and matched the same-tag GitHub Release digest.
- `pnpm package -- --dir`: produced the unsigned macOS x64 app/DMG/ZIP successfully; `app.asar` contains `electron/desktop-updater.cjs`, `main.cjs`, `preload.cjs`, and the Settings bundle.
- `pnpm smoke:release`: passed packaged-core startup and `/health` readiness.
- Pending before `Verified`: responsive screenshots plus real installer/AppImage handoff on macOS arm64/x64, Windows x64, and Linux x64. The current macOS package evidence is unsigned and does not satisfy signing/notarization acceptance.

## Security and data lifecycle

Only public release metadata and package bytes cross the network. The coordinator stores no credentials, device identifiers, analytics, prompt data, project paths, or persistent update state. Partial and rejected packages are deleted. A verified package remains only in the application temporary update directory until the OS or later cleanup removes it. Safe logs may contain version, platform, phase, and a bounded error category but not response bodies or arbitrary paths.

## Compatibility and migration

No schema or stored-data migration. Existing packaged builds remain compatible and simply lack the new client. If either public source is unavailable or inconsistent, the installed version continues running. Rollback is the existing user-visible OS package installation flow; the updater never downgrades automatically.
