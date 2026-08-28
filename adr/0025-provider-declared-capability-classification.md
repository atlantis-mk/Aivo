# ADR-0025: Classify Provider-declared tool capabilities

- Status: Accepted
- Date: 2026-08-25
- Related Work: `CHG-2026-052-provider-declared-capabilities`
- Closes OPEN: none

## Context

Provider model catalogs expose different machine-readable declarations. Anthropic identifies selected Provider-hosted tools, Mistral exposes `capabilities.function_calling`, OpenRouter exposes `supported_parameters`, and Cerebras exposes a richer public capability object. The latter declarations prove that a model can receive client-defined functions; they do not grant or identify a Provider-hosted executor.

Aivo also has legacy static model metadata. Unioning a returned explicit denial with that static fallback would make the remote declaration ineffective and could send tools to a model that explicitly reports no function-calling support.

## Decision

- Core MUST use a dedicated Provider parser for every supported model catalog that returns a documented machine-readable tool capability declaration.
- Core MUST record whether generic client-tool support is explicitly known separately from its boolean supported value. Missing, null, malformed, and unrecognized fields remain unknown rather than false.
- An explicit dynamic generic-tool declaration MUST override the corresponding legacy static value for the same model. Static metadata remains only a compatibility fallback when the Provider declaration is unknown.
- Generic `tools`, `function_calling`, or `supported_parameters` declarations authorize only serialization of Aivo-owned client tools. They MUST NOT create a Provider-hosted tool or local execution authority.
- Provider-hosted tools continue to require the explicit native-tool declaration and allowlisted Provider/transport/version mapping established by ADR-0024.
- Core MUST perform one bounded best-effort first-use refresh when supported declaration metadata is missing or stale. Failure preserves the prior cache and does not by itself block inference.
- Provider-specific public capability endpoints MAY be used only through a code-owned adapter for the matching built-in Provider. Renderer or remote catalog data cannot select an arbitrary metadata endpoint or parser.

## Consequences

- Mistral, OpenRouter, and Cerebras receive dedicated parsing instead of the lossy generic OpenAI-compatible parser.
- A model that explicitly denies function calling fails capability validation or falls back before Aivo sends client tools.
- Providers whose documented catalogs return only identities, modalities, generation actions, or static marketing metadata remain unknown and are not inferred.

## Verification

- `AT-PROVIDER-001`: Provider-specific parsing, known-versus-unknown persistence, stale refresh, and explicit denial precedence.
- `AT-TOOL-001`: dynamically supported client tools pass, explicit denial is refused, and generic declarations never create hosted tools.
- `CT-SECURITY-001`, `CT-RELIABILITY-001`: unknown fields cannot author tools, public metadata endpoints are code-owned, refresh is bounded, and failure preserves cache.
