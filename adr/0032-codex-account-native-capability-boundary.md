# ADR-0032: Separate Codex account-native capabilities from model declarations

- Status: Accepted
- Date: 2026-08-28
- Owners: Aivo maintainers
- Work: CHG-2026-060-codex-account-native-runtime

## Context

Aivo already derives model-specific capabilities from provider model catalogs and bridges declared local or hosted tools. ChatGPT Codex OAuth exposes a second capability class that is not a property of an individual model catalog entry: account-routed hosted tools such as Codex image generation, namespace-tool transport support, and OpenAI-bundled system skills. Treating these as static model declarations makes availability stale, allows fallback routes to retain tools they cannot execute, and risks charging or authenticating the wrong OpenAI surface.

Codex image generation uses the authenticated ChatGPT Codex account, a Codex backend image endpoint, and Codex usage limits. Bundled system skills are Host-distributed instruction packages rather than remote executable authority. Both must remain distinct from API-key OpenAI integrations and from arbitrary instructions returned by a provider.

## Decision

1. Aivo represents model-declared, provider-account, and Host-bundled skill capabilities as separate layers. Codex account capabilities are code-owned adapter facts for the built-in OpenAI Responses OAuth route and are recomputed for every effective route attempt.
2. Account-native tools use a dedicated `provider_account` activation policy. They are exposed only when the selected route is the built-in ChatGPT Codex OAuth route and the capability is enabled by that adapter. A fallback to API key, another provider, or another transport rebuilds the tool set and cannot retain the account tool.
3. Codex image generation mirrors the supported Codex contract: the `image_gen.imagegen` namespace identity, `gpt-image-2`, generation and edit endpoints, no more than five edit inputs, OAuth bearer and ChatGPT account headers, a bounded response body, and bounded decoded image output. Raw base64 images and authorization material are never logged or persisted in tool-call records. Successful output is saved as a local artifact and supplied to the active model as an image attachment.
4. Aivo indexes every Skill in the locally installed Codex `.system` catalog, including its supporting files, as a read-only Host Skill and refreshes changed content by directory hash. If that catalog is absent, Aivo materializes a reviewed versioned fallback set in Aivo-owned storage. It does not silently download or overwrite upstream packages. Provider-dependent Skills are filtered by the same effective route and cannot grant permissions or introduce tools.
5. Responses namespace calls preserve both the namespace and inner tool name in the domain call contract. Runtime resolution validates that the namespace matches the selected registration, preventing ambiguous or cross-namespace dispatch.
6. Remote model or provider strings never create executable code or new tool implementations. New hosted operations require a reviewed adapter; unknown declared capabilities remain observable but unavailable.

## Consequences

- Codex OAuth users receive the supported account-native tools and bundled skills without importing or configuring them manually.
- API-key OpenAI routes remain behaviorally and financially separate from ChatGPT Codex usage.
- Aivo must maintain tests for account routing, fallback isolation, image size and count limits, namespace parsing, system-skill synchronization, and fallback-version updates.
- Locally installed Codex system Skills refresh automatically. Updating Aivo's reviewed fallback Skills or adding another account-hosted operation requires an explicit source update, while ordinary model catalog changes remain dynamic.

## Alternatives considered

- Treat account tools as model-catalog capabilities. Rejected because the catalog does not own account surfaces and fallback routing would be unsafe.
- Inject skill instructions directly into every prompt. Rejected because it wastes context, bypasses progressive disclosure, and obscures route dependencies.
- Reuse the public OpenAI Images API for Codex OAuth. Rejected because authentication, endpoints, usage accounting, and availability are different products.
