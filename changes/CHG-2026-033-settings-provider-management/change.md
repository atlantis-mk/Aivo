# Manage model providers from desktop settings

## Problem or goal

Initialization already lets a user connect a model Provider, while the `/settings` route is empty. After initialization there is no durable desktop surface to inspect configured Providers, add another one through the same connection flow, or remove a Provider configuration.

## Expected behavior

- `REQ-PROVIDER-001`: Settings lists currently configured/connected Providers with safe status, model, and account-count summaries from `GetProviderCatalog`.
- A user can add or connect a Provider with the same Provider picker, API-key/OAuth/custom configuration, model refresh, validation, and connection dialogs used during initialization.
- A user can remove a whole Provider configuration only after native confirmation. Removal calls `DeleteProvider`, refreshes the catalog, clears related Provider auth/config/cache state in Core, and retains no renderer copy of credentials.
- Loading, empty, error, busy, wide, and narrow states remain usable.

## Non-goals

- Adding a new Provider protocol, auth method, RPC, persistence format, diagnostics dashboard, usage/cost view, or schema migration.
- Returning, pre-filling, exporting, or storing Provider credentials in renderer state beyond the existing transient connection input.
- Redesigning unrelated settings categories or changing account-level removal during initialization.

## Impact

- Renderer: implements the settings Provider management surface and composes the existing initialization connection state/actions/dialogs.
- Desktop service: exposes the already-supported `DeleteProvider` RPC through the typed renderer service.
- Core, Electron main/preload, persistence, Provider runtime, public contracts, schema/data, extensions/MCP, LSP, terminals, worktrees, dependencies, and packaging are unchanged.
- Security: Provider secrets continue to cross only the existing one-time privileged connection request and catalog reads remain secret-free.

## Implementation constraints

- Core's catalog remains the source of truth; the renderer must not infer or persist a second Provider registry.
- Connection behavior must reuse the initialization flow so OAuth events, custom Provider validation, model selection, credential handling, and failure messages do not diverge.
- Provider deletion must use `DeleteProvider`, require confirmation, prevent repeated submission while pending, and update the shared catalog only from the returned Core result.
- Built-in Providers remain discoverable after deletion but are no longer shown as configured; custom Provider configuration removal follows Core ownership and cleanup.

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `PROVIDER-SETTINGS-UI-001` | `REQ-PROVIDER-001`, `NFR-UI-001` | Responsive settings list, add/connect dialogs, summaries, empty/loading/error states | `AT-PROVIDER-001`, `AT-UI-001` | Complete |
| `PROVIDER-SETTINGS-DELETE-001` | `REQ-PROVIDER-001`, `NFR-SECURITY-001` | Confirmed whole-Provider deletion through existing Core RPC | `AT-PROVIDER-001`, `CT-SECURITY-001` | Complete |
| `PROVIDER-SETTINGS-VERIFY-001` | `NFR-UI-001`, `NFR-SECURITY-001` | Focused tests, lint/build, docs checks, and responsive screenshots | `AT-PROVIDER-001`, `AT-UI-001`, `CT-SECURITY-001` | In progress |

## Acceptance and evidence

- Happy path covers initial catalog load, connecting a built-in or custom Provider, immediate catalog refresh, inspecting its safe summary, and confirmed deletion.
- Boundaries cover no configured Providers, catalog failure, connection failure, deletion failure, repeated deletion prevention, multiple accounts, missing model metadata, and narrow overflow.
- Cancellation is the existing dialog close or delete-confirm cancel path. Provider operations are bounded existing Core calls; teardown, timeout, compatibility, migration, and rollback changes are N/A.
- Applicable gates are focused desktop tests, `pnpm docs:check`, `pnpm lint`, `pnpm build`, `git diff --check`, and wide/narrow screenshots.

### Verification evidence

- `apps/desktop/tests/provider-settings-model.test.ts` passes three focused catalog filtering, default ordering, safe-summary, and actionable-readiness cases.
- `pnpm docs:check`, `pnpm lint`, `pnpm build`, and `git diff --check` pass. Lint reports only the repository's existing shared-UI Fast Refresh warnings; Vite reports the existing large-barrel and chunk-size advisories.
- User-owned wide/narrow visual acceptance remains pending, so this Work stays `Implementing` and is not sealed.

## Security and data lifecycle

Provider API keys and OAuth values retain their existing lifecycle: transient password/OAuth UI state, one privileged connection request, Core-owned secure storage/reference, and secret-free catalog results. The settings list renders only Provider identity, configured/ready status, model identity, connection method, and account count. Deletion sends only the Provider ID; secret cleanup remains Core-owned. No credential enters localStorage, SQLite plaintext, logs, diagnostics, screenshots, clipboard, crash output, or model context.

## Compatibility and migration

No schema, data, RPC, IPC, or Provider-format migration. Existing configurations appear through the current catalog without conversion. Rolling back the renderer leaves Provider data and existing initialization behavior unchanged.

## Bug root cause (type=bug only)

N/A; this Work adds the accepted settings management surface.
