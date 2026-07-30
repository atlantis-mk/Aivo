# Aivo v2 preparation and delivery tasks

## 0. Source protection and baseline

- [x] Preserve all pre-v2 modified, untracked, and deleted files in commit
  `bb528c8`.
- [x] Create archive branch `codex/aivo-v1-archive`.
- [x] Create immutable reference tag `aivo-v1-archive-2026-07-31`.
- [x] Start `codex/aivo-v2` from the archive commit.
- [x] Run `pnpm test:core`.
- [x] Run `pnpm lint` and record existing warnings.
- [x] Run `pnpm build` and record existing warnings.
- [ ] Push the archive branch/tag only after explicit publication approval.

## 1. Product definition gate

- [ ] Resolve all P0 items in `decision-log.md`.
- [ ] Write the primary user journey and three failure/recovery journeys.
- [ ] Mark each inventory row retain/refactor/redesign/remove.
- [ ] Define v2 launch scope and explicit exclusions.
- [ ] Define measurable acceptance criteria for the first slice.

## 2. Contract and data gate

- [ ] Map current RPC methods to domain resources and owners.
- [ ] Define the unified v2 response/error envelope.
- [ ] Define versioned contracts for the first slice.
- [ ] Inventory every persisted table and sensitive field.
- [ ] Introduce explicit schema-version transitions beyond version 1.
- [ ] Build sanitized v1 migration fixtures and rollback tests.
- [ ] Define structured logging fields and redaction rules.

## 3. UX foundation gate

- [ ] Map the v2 navigation and object hierarchy.
- [ ] Design wide and narrow layouts for the first slice.
- [ ] Specify loading, empty, error, offline, permission, and long-content states.
- [ ] Define accessibility keyboard paths and focus restoration.
- [ ] Capture the current v1 flow for comparison before replacing it.

## 4. Vertical slice delivery

Repeat for each approved slice:

- [ ] Finalize domain language and lifecycle.
- [ ] Implement persistence changes and migration tests.
- [ ] Implement application service and versioned contract.
- [ ] Update preload/desktop service bindings without hand-editing generated Go
  bridge files.
- [ ] Implement responsive feature UI outside shared UI primitives.
- [ ] Add cancellation, retry, recovery, and permission behavior.
- [ ] Run focused Go tests, `pnpm test:core`, `pnpm lint`, and `pnpm build`.
- [ ] Capture wide/narrow screenshots and complete manual acceptance evidence.
- [ ] Update specs, decision log, and compatibility-removal conditions.

## 5. Release readiness

- [ ] Verify installation and upgrade from the archived v1 release.
- [ ] Verify database backup, migration, failure recovery, and rollback.
- [ ] Run provider smoke diagnostics for launch providers.
- [ ] Run release smoke checks on supported operating systems.
- [ ] Confirm no credentials, local auth stores, databases, or generated secrets
  are tracked.
- [ ] Prepare release notes with breaking changes and recovery instructions.
- [ ] Remove compatibility code only after its acceptance evidence is complete.
