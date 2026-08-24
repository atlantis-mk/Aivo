# ADR-0022: Trusted desktop update boundary

- Status: Accepted
- Date: 2026-08-25
- Related Work: `CHG-2026-044-desktop-auto-update`
- Closes OPEN: none

## Context

Aivo already publishes immutable packages to R2 and matching GitHub Release assets, with a mutable R2 stable-channel pointer. Desktop update code can download and execute privileged native packages, while current packages may be unsigned. A digest contained only beside the downloaded object does not independently establish publisher intent if that one channel is compromised.

## Decision

Electron main MUST own a fixed-channel updater and expose only check, download, install/handoff, state, and cancellation capabilities through preload. It MUST derive the artifact name from its own platform and architecture, require stable SemVer advancement, validate the pinned R2 HTTPS path, and require matching name, size, and SHA-256 from the fixed GitHub repository's same-tag Release before download. It MUST verify downloaded size and SHA-256 before any OS handoff. Renderer input MUST NOT select origins, paths, commands, versions, or digests.

Installation MUST remain user-confirmed and OS-visible. Unsigned artifacts MUST NOT be described as signed or installed silently. macOS and Windows open the verified native installer package; Linux reveals the verified AppImage because safe in-place replacement depends on how the user installed the current AppImage.

## Rationale

- This reuses the existing publication contract without adding an updater dependency or a private signing service.
- Independent R2 and GitHub metadata agreement detects accidental or single-channel asset substitution before native code is opened.
- Fixed capabilities prevent a compromised renderer from converting the updater into an arbitrary downloader or process launcher.
- Explicit handoff preserves OS security prompts and accommodates currently unsigned packages.

## Consequences

- Updates require both public services to be reachable and consistent.
- This is defense in depth, not a replacement for platform code signing and notarization.
- Full automatic replacement, delta updates, alternate channels, and offline enterprise feeds require a later ADR.
- Platform handoff behavior requires target-OS acceptance before release readiness.

## Rejected alternatives

- `electron-updater` generic-provider metadata: the current stable manifest is the canonical release contract and does not publish its platform YAML files; adopting it now adds a production dependency without solving unsigned-package trust.
- R2 manifest digest alone: it does not independently confirm the GitHub release asset selected by the existing publication workflow.
- Silent installer execution: rejected while packages are unsigned and because it hides platform security and elevation decisions.
- Renderer-selected URLs or paths: rejected because it creates an arbitrary privileged network/file execution primitive.

## Verification

`AT-UPDATE-001`, `CT-SECURITY-001`, `CT-RELIABILITY-001`, target-OS package handoff acceptance, and the existing `GATE-8` publication integrity check.
