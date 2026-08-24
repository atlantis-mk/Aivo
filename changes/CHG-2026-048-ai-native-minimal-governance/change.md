# Adopt AI-native minimal governance

## Problem or goal

The previous governance revision removed duplicated prose and automated routine lifecycle work, but it still creates Work for ordinary behavior changes and keeps process evidence that Git, tests, and current specifications already preserve more directly. A fully AI-operated repository benefits more from compact current truth and executable constraints than from detailed process archives.

## Behavior or governance delta

Direct change becomes the default, including ordinary same-task behavior and specification changes. Work is reserved for unfinished cross-task state, open decisions requiring approval, controlled security/data/contract/migration/platform/release boundaries, severe unclear bugs, and irreversible coordination. New Work uses one schema-v2 `change.yaml`, three live states (`Draft`, `Active`, `Done`), and no generated command-evidence file or hash archive. Git owns completed history; current specifications and tests own present truth.

## Non-goals

This Work does not rewrite or delete historical sealed Work, weaken product security boundaries, remove ADRs for durable high-risk decisions, or assume current uncommitted changes are durable Git history. Legacy Work and archive validation remain supported until those records naturally leave the active set.

## Controlled impact

Documentation governance, Work creation/start/finish scripts, Traceability generation, documentation validation, templates, and tests change. Product runtime code, user data, persistence, public API/RPC/IPC, credentials, packages, and release artifacts do not change.

## Tasks and acceptance

- Make Direct the default and narrow Work creation to durable uncertainty or controlled boundaries.
- Replace new Work profiles and specification-delta ceremony with one concise schema-v2 YAML.
- Collapse new lifecycle states to Draft, Active, and Done.
- Stop producing `verification.json` and archive hashes for schema-v2 Work; preserve legacy validation.
- Generate active and completed Traceability from Work status and historical archive metadata.
- Pass documentation, lifecycle, archive-compatibility, release-script, desktop model, and extension example tests.

## Evidence policy

Command results are reported in the task and CI rather than copied into new Work. This legacy governance Work will use the existing completion path once, because it was accepted under revision `0.1.3-active`; its sealed record demonstrates backward compatibility.

Implementation result: Direct is now the default for ordinary same-task behavior and specification changes. New Work is a single schema-v2 YAML with Draft/Active/Done states and only goal, routing, boundary, risk, and next-action metadata. Its completion path writes no body, command-evidence file, or archive entry. Documentation validation accepts schema-v2 Done Work for Release references while preserving every legacy schema and archive digest. Deterministic tests exercise new creation/start/finish/rollback behavior and the legacy finish/archive compatibility path.

## Bug root cause (type=bug only)

N/A.
