# Make stable publication operator-managed

## Problem or goal

The stable publication workflow currently treats repository documentation, scripts, Core tests, lint, desktop build, package smoke, and native handoff checks as mandatory release gates. The operator already performs the desired testing and wants an explicit tag or manual publication command to publish directly without CI deciding release readiness.

## Expected behavior

`NFR-RELEASE-001` makes the human operator the release-readiness authority. Pushing a stable version tag or manually dispatching the existing stable workflow starts native packaging and publication directly. The publication workflow does not run `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, `pnpm build`, `pnpm smoke:release`, or native handoff verification as blocking gates.

Publication mechanics remain fail-closed: tag/package/release-record identity must match, all four native package jobs must build the required assets, immutable R2 objects cannot be overwritten with different bytes, the stable manifest is published last and read back, and GitHub assets must match their digests. These checks protect artifact integrity rather than decide whether the operator is allowed to release.

## Non-goals

This Work does not add prerelease channels, partial-platform stable publication, silent installation, signing claims, alternate package formats, automatic version editing, or a new updater contract. It does not delete test commands; it removes them only from the publication decision path.

## Impact

GitHub Actions stable publication and release governance change. Renderer, Electron privilege boundaries, Go domain/application/persistence/transport, schema/data, local API/RPC/IPC, providers, extensions/MCP, updater selection, and runtime dependencies are unchanged.

## Implementation constraints

The explicit operator trigger is the existing stable `vX.Y.Z` tag push or manual `publish-release.yml` dispatch. Native packaging must still succeed because packages cannot be published without being built. Publication retains target completeness, immutable-object, digest, stable-manifest ordering/readback, and resumability checks. Quality commands remain available for voluntary use and separate diagnostics.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| RELEASE-1 | NFR-RELEASE-001 | Define operator-owned release readiness and advisory checks | CT-RELEASE-002 | Complete |
| RELEASE-2 | NFR-RELEASE-001 | Remove quality/smoke gates from stable publication workflow | CT-RELEASE-002 | Complete |
| RELEASE-3 | NFR-RELEASE-001 | Preserve mechanical integrity and recovery checks | CT-RELEASE-001, CT-RELEASE-002 | Complete |

## Acceptance and evidence

- Static tests prove the stable publication workflow omits documentation, script, Core, lint, build, package-smoke, and native-handoff gates.
- Existing publication tests continue to prove tag/version/release-record binding, four-platform completeness, immutable R2 behavior, stable-manifest-last ordering, digest-bound GitHub assets, and recovery.
- `pnpm docs:check` and `pnpm scripts:test` pass locally as implementation verification; they are not invoked by the publication workflow.
- One operator-triggered stable workflow run is required before this Work may become `Verified` and be sealed.

Local evidence on `2026-08-25`: focused release tests passed 10/10; `pnpm docs:check` passed across 97 Markdown, 49 YAML, 22 Requirements, 25 Test IDs, 22 ADRs, 48 Work Packages, and 29 archived Work Packages; `pnpm scripts:test` passed 22 script/governance tests, 80 desktop tests, and four example-extension tests. Ruby Psych parsed the stable release workflow YAML and `git diff --check` passed. The Work remains `Implementing` pending one operator-triggered native publication run.

## Security and data lifecycle

No credential ownership changes. R2 credentials remain scoped to publication jobs and logs remain secret-free. Removing readiness gates increases the operator's responsibility for behavioral, security, migration, signing, and platform acceptance; mechanical artifact integrity remains enforced.

## Compatibility and migration

No schema, data, API/RPC/IPC, settings, package-name, R2-key, GitHub-asset, or updater contract change. Existing stable releases and recovery runs remain compatible.

## Bug root cause (type=bug only)

N/A.
