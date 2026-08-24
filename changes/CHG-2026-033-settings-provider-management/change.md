# Manage model providers from desktop settings

## Problem or goal

Initialization already lets a user connect a model Provider, while the `/settings` route is empty. After initialization there is no durable desktop surface to inspect configured Providers, add another one through the same connection flow, or remove a Provider configuration.

## Expected behavior

- `REQ-PROVIDER-001`: Settings lists currently configured/connected Providers with safe status, model, and account-count summaries from `GetProviderCatalog`.
- A user can add or connect a Provider with the same Provider picker, API-key/OAuth/custom configuration, model refresh, validation, and connection dialogs used during initialization.
- A user can refresh Aivo's shared public Provider/model ecosystem catalog from the add-Provider section. Success reloads the settings catalog so subsequent Provider connection dialogs use the updated internal model lists; failure preserves the prior catalog and remains actionable.
- A user can also refresh one configured Provider's remote model list from its settings card. Core reuses the persisted Provider configuration and credential reference, replaces that Provider's cached list and displayed list-default model on success, preserves the application's active model preference, and leaves the prior catalog intact on failure.
- A user can remove a whole Provider configuration only after native confirmation. Removal calls `DeleteProvider`, refreshes the catalog, clears related Provider auth/config/cache state in Core, and retains no renderer copy of credentials.
- Loading, empty, error, busy, wide, and narrow states remain usable.

## Non-goals

- Adding a new Provider protocol, auth method, RPC, persistence format, diagnostics dashboard, usage/cost view, or schema migration.
- Returning, pre-filling, exporting, or storing Provider credentials in renderer state beyond the existing transient connection input.
- Redesigning unrelated settings categories or changing account-level removal during initialization.

## Impact

- Renderer: implements the settings Provider management surface and composes the existing initialization connection state/actions/dialogs.
- Desktop service: exposes the already-supported `DeleteProvider` RPC through the typed renderer service.
- The renderer reuses the existing `RefreshProviderEcosystemCatalog`, `GetProviderCatalog`, and `RefreshProviderModels` RPCs. Core lets the existing Provider-model refresh accept a configured Provider identity and rehydrate its saved non-secret connection fields before resolving the existing Core-owned credential. Ecosystem entries use the existing OpenAI-compatible, Anthropic, or Google model-list strategy when their recognized transport supports it; other transports remain static. Unsupported static refresh attempts and remote endpoint, authentication, or parsing failures surface immediate feedback while retaining the previous model cache. RPC shapes, Electron main/preload, persistence schema, Provider runtime, public contracts, extensions/MCP, LSP, terminals, worktrees, dependencies, and packaging are unchanged.
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
| `PROVIDER-SETTINGS-CATALOG-REFRESH-001` | `REQ-PROVIDER-001`, `NFR-SECURITY-001`, `NFR-UI-001` | Add-section ecosystem refresh with local catalog reload, updated add-flow model lists, busy/error feedback, and no Provider credential use | `AT-PROVIDER-001`, `CT-SECURITY-001`, `AT-UI-001` | Complete |
| `PROVIDER-SETTINGS-PROVIDER-REFRESH-001` | `REQ-PROVIDER-001`, `NFR-SECURITY-001`, `NFR-UI-001` | Per-card remote refresh with persisted Core config, cache/default-list replacement, busy/error feedback, and no active-preference mutation | `AT-PROVIDER-001`, `CT-SECURITY-001`, `AT-UI-001` | Complete |
| `PROVIDER-SETTINGS-DELETE-001` | `REQ-PROVIDER-001`, `NFR-SECURITY-001` | Confirmed whole-Provider deletion through existing Core RPC | `AT-PROVIDER-001`, `CT-SECURITY-001` | Complete |
| `PROVIDER-SETTINGS-VERIFY-001` | `NFR-UI-001`, `NFR-SECURITY-001` | Focused tests, lint/build, docs checks, and responsive screenshots | `AT-PROVIDER-001`, `AT-UI-001`, `CT-SECURITY-001` | In progress |

## Acceptance and evidence

- Happy path covers initial catalog load, public ecosystem refresh with immediate add-flow model availability, configured-Provider remote refresh, connecting a built-in or custom Provider, inspecting its safe summary, and confirmed deletion.
- Boundaries cover no configured Providers, catalog failure, connection failure, ecosystem and configured-Provider refresh failures, unsupported configured refresh, repeated/cross-refresh prevention, deletion failure, repeated deletion prevention, multiple accounts, missing model metadata, and narrow overflow.
- Cancellation is the existing dialog close or delete-confirm cancel path. Provider operations are bounded existing Core calls; teardown, timeout, compatibility, migration, and rollback changes are N/A.
- Applicable gates are focused desktop tests, `pnpm docs:check`, `pnpm lint`, `pnpm build`, `git diff --check`, and wide/narrow screenshots.

### Verification evidence

- `apps/desktop/tests/provider-settings-model.test.ts` passes four focused catalog filtering, default ordering, safe-summary, actionable-readiness, refreshability, and secret-free configured-refresh input cases.
- `core/app/provider_ecosystem_catalog_test.go` proves the public Provider/model catalog is refreshed, cached, registered immediately, and restored for offline startup without Provider credentials.
- `core/app/provider_models_test.go` proves an identity-only configured-Provider refresh reuses the persisted endpoint, safe custom headers, and Core credential, replaces the cached/model-list default, and preserves the active application model preference.
- `pnpm scripts:test` passes 68 desktop model tests plus governance and extension UI-test coverage; `pnpm test:core` passes all Core packages with both refresh paths present.
- `pnpm docs:check`, `pnpm lint`, `pnpm build`, and `git diff --check` pass. Lint reports only the repository's existing shared-UI Fast Refresh warnings; Vite reports the existing large-barrel and chunk-size advisories.
- User-owned wide/narrow visual acceptance remains pending, so this Work stays `Implementing` and is not sealed.

## Security and data lifecycle

Provider API keys and OAuth values retain their existing lifecycle: transient password/OAuth UI state, one privileged connection request, Core-owned secure storage/reference, and secret-free catalog results. The settings list renders only Provider identity, configured/ready status, model identity, connection method, and account count. Ecosystem refresh fetches the configured public catalog URL and uses no configured Provider credential. Configured-Provider refresh sends only Provider identity and display name; Core rehydrates saved non-secret connection fields and resolves the existing credential without returning it. Deletion sends only the Provider ID; secret cleanup remains Core-owned. No credential enters localStorage, SQLite plaintext, logs, diagnostics, screenshots, clipboard, crash output, or model context.

## Compatibility and migration

No schema, data, RPC, IPC, or Provider-format migration. Existing configurations appear through the current catalog without conversion. Rolling back the renderer leaves Provider data and existing initialization behavior unchanged.

## Bug root cause (type=bug only)

N/A; this Work adds the accepted settings management surface.
