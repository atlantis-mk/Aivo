# Reduce documentation governance overhead

## Problem or goal

The repository preserves durable decisions well, but ordinary Work pays much of the same authoring and state-management cost as security, migration, contract, and release changes. Requirement acceptance, trace evidence, test policy, and Work evidence are also repeated across several manually maintained documents.

## Expected behavior

Governance provides three lanes: direct changes for local reversible work, Light Work for continuity without a product or architecture decision, and Controlled Work for durable behavior or high-risk boundaries. New Work declares a profile and whether it changes a primary specification. Requirement definitions remain authoritative while Traceability becomes a generated index. Work creation, start, verification evidence, completion, and archive sealing are supported by repository commands.

## Non-goals

This Work does not weaken security, trust, credential, persistence, schema, public API/RPC/IPC, compatibility, platform, or release controls. It does not modify sealed Work, redefine product Requirements, or claim that existing Implementing Work is verified without applicable evidence.

## Impact

`AGENTS.md`, document governance, the Work templates, test policy, Traceability, documentation checks, package scripts, and unsealed Work metadata change. Product code, persistence, transport, desktop behavior, user data, credentials, and public runtime contracts do not change.

## Implementation constraints

Sealed Work and existing archive entries remain immutable. The new checker must accept their historical schema while requiring the new schema for templates and unsealed Work. Generated Traceability must be deterministic and fail validation when stale. Completion automation must record structured evidence, update status, seal atomically, and restore files if a post-update check fails.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| GOV-1 | N/A | Define Direct, Light, and Controlled lanes plus specification deltas | N/A | Complete |
| GOV-2 | N/A | Make Traceability deterministic and generated from Requirements and Work metadata | N/A | Complete |
| GOV-3 | N/A | Add Work creation, start, and finish commands with structured evidence | N/A | Complete |
| GOV-4 | N/A | Migrate only unsealed Work and preserve archived bytes | N/A | Complete |
| GOV-5 | N/A | Verify governance scripts and documentation integrity | N/A | Complete |

## Acceptance and evidence

- Documentation checks validate canonical governance links rather than duplicated prose.
- New unsealed Work requires `profile` and `spec_delta`; sealed historical Work remains valid without modification.
- Traceability regeneration is deterministic and `docs:check` refuses stale output.
- Script tests cover profile rules, generation, state transitions, evidence capture, archive sealing, and rollback behavior.
- The repository documentation and script gates pass before this Work becomes Verified and is sealed.

Implementation result: governance now has Direct, Light, and Controlled lanes; the Light template is five short sections; `spec_delta` controls primary-spec work; Traceability is deterministic and stale-checked; and `work:new`, `work:start`, and `work:finish` own routine lifecycle mutations. All 19 previously unsealed Work records received explicit Controlled/behavior-change metadata, while all 29 sealed Work directories retained their archived bytes and historical schema.

The active-Work audit found no Work that could be truthfully closed by moving release-only evidence: 18 product Work records still name feature-specific visual or interaction acceptance, and the operator-managed publication Work changes the release mechanism itself and still requires one real workflow run. Their states therefore remain unchanged rather than treating deferred evidence as passed.

## Security and data lifecycle

Light Work cannot cross security, trust, secrets, data ownership, persistence/schema, public contracts, compatibility/migration, platform, dependency/license, irreversible, or release boundaries. Verification evidence stores command summaries, timestamps, durations, and commit identity only; it must not capture environment variables, raw command output, credentials, prompts, or private data.

## Compatibility and migration

The documentation schema moves from `0.1.2-active` to `0.1.3-active`. Unsealed Work is migrated forward. Sealed Work remains readable under its archived schema and is never rewritten. Existing product, data, package, API, RPC, IPC, and release formats are unchanged.

## Bug root cause (type=bug only)

N/A.
