# Provider Backend

The provider backend is the production control plane for model providers. The desktop UI should treat these APIs as the source of truth instead of duplicating provider logic.

## Core Responsibilities

- Provider registry: built-in and custom provider definitions, aliases, transports, auth methods, model metadata, and capability metadata.
- Credential handling: API keys and OAuth tokens are stored through secret references; plaintext credentials should not be persisted in SQLite.
- Model discovery: remote model refresh, Provider-specific declaration parsing for Anthropic, Mistral, OpenRouter, Cerebras, and authenticated ChatGPT Codex catalogs, persisted known-versus-supported capability metadata, stale-cache validation fallback, and allowlisted Codex context/reasoning/verbosity/modality/tier/parallel/search/Lite request compatibility.
- Codex capability bridge: only an OpenAI OAuth/Codex route may use recognized declarations to filter the existing `bash` tool, inject ordinary hosted search, or activate the canonical local search executor for Responses Lite/explicit local routing. Lite body/header placement and bounded source preservation are code-owned; OpenAI API-key routes and remote prompt/policy fields are ignored.
- Codex account bridge: separately from model declarations, the built-in OAuth adapter exposes namespace-tool transport and the account-native `image_gen.imagegen` operation using Codex authentication and usage. Generated output is bounded and saved as a local artifact; API-key and fallback routes never inherit the account tool. A connected account also auto-synchronizes installed or bundled read-only Codex system Skills through the Host Skill resolver.
- Runtime routing: active model resolution, fallback model routing, model capability checks, retry policy, rate-limit cooldown, and streaming fallback buffering.
- Diagnostics: provider validation, integration checks, health snapshots, call events, usage aggregation, and structured provider errors.

## Main RPC Methods

- `GetProviderCatalog`: list providers, models, readiness, health, accounts, model refresh state, and profile metadata.
- `RefreshProviderEcosystemCatalog`: refresh and persist Aivo's shared public Provider/model directory used by Provider pickers and connection model options.
- `SaveProvider`: save or update a provider without switching the active default model.
- `ConnectProvider`: save a provider and make it the active/default provider.
- `DeleteProvider`: remove provider config, auth, cached model metadata, validation, and health. Call events are retained for audit.
- `StartProviderAuth`, `GetProviderAuthStatus`, `CancelProviderAuth`, `DeleteProviderAccount`: interactive auth and account lifecycle.
- `RefreshProviderModels`: fetch and cache remote model metadata.
- `ValidateProvider`: validate config, credentials, and model listing.
- `CheckProviderIntegration`: structured readiness report for config, auth, models, route resolution, capabilities, policy, health, usage, and recent call events.
- `ListProviderCallEvents`: recent provider call attempts for diagnostics.
- `GetProviderUsage`: aggregate calls, failures, tokens, and cost estimates.
- `UpdateModelPreferences`: update active model, fallback models, reasoning effort, service tier, and provider runtime policy.

## Runtime Policy

`AppConfig.providerPolicy` controls runtime behavior:

```json
{
  "enableFallback": true,
  "bufferStreamingFallback": true,
  "maxRetries": 1,
  "retryBaseDelayMs": 100,
  "rateLimitCooldownSeconds": 30
}
```

- `enableFallback`: when false, only the primary route is attempted.
- `bufferStreamingFallback`: when true, streaming deltas from earlier fallback candidates are buffered until that candidate succeeds, preventing partial output leaks.
- `maxRetries`: retry count for non-streaming retryable errors. `0` disables retry.
- `retryBaseDelayMs`: linear retry delay base.
- `rateLimitCooldownSeconds`: local cooldown after provider 429 responses.

## Smoke Check

Run a backend-only provider integration check against the persisted local config:

```bash
cd /Users/atlan/Documents/Aivo/core
go run ./cmd/aivo-core provider-smoke --provider openai --model gpt-5.5
```

The command prints `CheckProviderIntegration` JSON and exits non-zero when the provider is not ready.

Use `--include-model-list` only when debugging model parser output, because it can produce a large response.

## Usage And Cost

Provider call events prefer provider-native usage fields when present. When usage is missing, the backend estimates tokens from request text, tool schemas, and output text, and marks the event as `estimated=true`.

Cost is calculated from `ModelInfo.pricing` when available. Unknown pricing produces `costMicros=0` rather than guessing rates.

## Backend Completion Bar

Before desktop UI work, backend provider behavior should pass:

```bash
cd /Users/atlan/Documents/Aivo/core
go test ./...
go run ./cmd/aivo-core provider-smoke --provider <configured-provider>
```
