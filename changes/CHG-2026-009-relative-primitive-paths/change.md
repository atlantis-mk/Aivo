# Guide primitive file tools to workspace-relative paths

## Problem or goal

In macOS user acceptance, asking Aivo to edit `notes.txt` produced an initial `edit` call with an absolute workspace path. Permission validation rejected it as `path escapes workspace root`; the model then retried with `notes.txt` and succeeded. Expected behavior is one correctly formed call. Actual primitive schemas declare `path` only as a non-empty string and do not tell the model to remove the workspace-root prefix. `read` also accepts absolute paths inside the workspace, making the three core file tools inconsistent.

## Expected behavior

- `REQ-TOOL-001`: `read`, `edit`, and `write` describe workspace file inputs as workspace-relative paths with a concrete example and an explicit prohibition on absolute workspace paths.
- Absolute workspace paths fail with an actionable message directing the caller to remove the workspace-root prefix; `..` traversal continues to use the escape error.
- `read` retains the existing absolute-path exception only for exact Host-returned retained-output references.
- `NFR-RELIABILITY-001`: valid relative-path calls continue to execute once without a validation/retry round trip.

## Non-goals

- Do not accept absolute paths for `edit` or `write`.
- Do not weaken symlink, traversal, sensitive-file, permission, or Execution Environment checks.
- Do not change retained Bash artifact ownership or introduce fuzzy path correction.

## Impact

- Core tool schemas and local path helpers change; provider, renderer, Electron, persistence schema, transport, extensions, dependencies, packaging, and platform scope do not.
- Error text becomes more actionable while preserving the existing `invalid_path`/permission-denied result categories.

## Implementation constraints

- The Agent-visible schema is the primary prevention mechanism; runtime validation remains authoritative.
- All three primitives use one shared relative-path description and one shared absolute-path error to avoid semantic drift.
- The retained-output exception must be checked before rejecting an absolute `read` path.
- No ADR revision or migration is required.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `BUG-PATH-001` | `REQ-TOOL-001` | Add explicit workspace-relative `path` descriptions to `read`, `edit`, and `write` | `AT-TOOL-001` | Complete |
| `BUG-PATH-002` | `REQ-TOOL-001` | Return actionable absolute-path errors while preserving traversal and retained-output behavior | `AT-TOOL-001` | Complete |
| `BUG-PATH-003` | `NFR-RELIABILITY-001` | Add pre-fix failing/post-fix passing schema and execution regression tests | `CT-RELIABILITY-001` | Complete |

## Acceptance and evidence

- A regression must fail before the fix because the three `path` properties lack relative-path descriptions and absolute `read` is accepted.
- Post-fix tests prove all three schema descriptions contain the shared rule, valid relative paths still work, absolute workspace paths return the actionable message, and traversal still reports escape.
- Existing retained-output, primitive, permission, alternate-environment, and full Core tests pass.
- Repetition is covered by deterministic schema equality; cancellation/timeout/teardown remain unchanged and are covered by the existing suite.
- Migration/rollback, dependency loss, UI screenshots, installer, signing, and package smoke are N/A for this schema-description and validation correction.
- Pre-fix evidence on 2026-08-03: `go test ./app -run '^TestPrimitivePathContractGuidesWorkspaceRelativePaths$' -count=1` failed with `read path description = "", want explicit relative-path rule and example` before product-code changes.
- Post-fix focused evidence: `TestPrimitivePathContractGuidesWorkspaceRelativePaths` passed together with text read, atomic edit, and serialized write tests. It proves all three schemas contain the shared relative rule and `"notes.txt"` example; absolute workspace inputs return `invalid_path` with `remove the workspace-root prefix`; permission preflight uses the same text; traversal remains an escape error; and an exact Host retained-output ref remains readable.
- `pnpm test:core` passed on 2026-08-03 for Core app, CLI, persistence, and HTTP transport packages.
- Repository gates passed on 2026-08-03: `pnpm docs:check`, `pnpm scripts:test`, `pnpm lint`, `pnpm build`, and `git diff --check`. Lint reported only existing Fast Refresh warnings; build reported only existing large-barrel and chunk-size advisories.
- Verification platform: macOS 14.8.7, Darwin 23.6.0 x86_64, Go 1.26.0 darwin/amd64.

## Security and data lifecycle

No new data is persisted or logged. Error messages contain no absolute user path; they provide a fixed relative example. Existing workspace, symlink, sensitive-file, permission, and retained-artifact validation remains authoritative.

## Compatibility and migration

No persistence, API/RPC/IPC, settings, or file-format migration. Absolute workspace inputs to `read` become invalid, matching the already-required mutation contract; exact retained-output references remain compatible. Rollback restores inconsistent guidance and can reintroduce the failed-call retry.

## Bug root cause (type=bug only)

Affected version: `0.0.0-development`. Primitive JSON schemas omitted semantic descriptions for `path`; mutation validation returned a generic escape message for absolute inputs, and `read` silently normalized absolute workspace paths. Existing tests exercised relative happy paths, traversal, retained output, and path containment but did not assert the model-visible schema guidance or a consistent absolute-workspace-path contract. The fix uses one shared model-visible description and absolute-path error across all three primitives and permission preflight, while preserving the Host retained-output exception. The new regression failed before the fix and passed after it. Fix version: `0.0.0-development`.
