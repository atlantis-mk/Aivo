# ADR-0024: Dynamically bridge Provider-native server tools

- Status: Accepted
- Date: 2026-08-25
- Related Work: `CHG-2026-051-dynamic-provider-native-tools`
- Closes OPEN: none

## Context

Provider model catalogs differ materially. Some return structured model capability metadata, while others return only identity or generation-method metadata. Aivo currently maintains static model capability lists and adapts a Provider-hosted tool only when an equivalent logical tool already reached the request. This lets returned capability metadata go unused and leaves supported hosted tools disconnected unless a user or another source manually activates them.

Provider-hosted tools execute on the configured Provider's infrastructure rather than in Aivo's local execution environment. Automatically declaring one therefore changes the Provider request and may add Provider-side processing or usage charges, but it does not grant a local filesystem, process, credential, extension, or MCP privilege.

## Decision

- Core MUST parse and persist explicit native-tool support returned by an authenticated Provider model catalog when a dedicated parser understands that Provider response.
- Core MUST distinguish an explicitly returned native-tool set from unknown metadata. An absent, partial, malformed, or unrecognized capability response MUST NOT be interpreted as support.
- Core MUST make one bounded best-effort capability sync before first use of a model whose authoritative native-tool metadata is missing or older than the Host-owned freshness window. Refresh failure MUST preserve the prior cache and MUST NOT block the model request solely because capability discovery failed.
- Core MUST automatically declare an explicitly supported Provider-native server tool without requiring user activation only when an allowlisted adapter matches the Provider, transport, canonical capability, and protocol tool version.
- Core MUST NOT infer automatic Provider-native tools from a model name, marketing description, generic tool support, or Aivo's legacy static capability list.
- Explicit application configuration MUST be able to disable an automatically bridged native tool. Unknown disable names MUST have no effect.
- Provider-hosted server tools MUST remain outside the local executable Registry and local Tool Snapshot because Aivo does not execute their calls. Their request serialization and response handling remain Provider-adapter responsibilities.
- Static capability metadata MAY remain a compatibility fallback for generic model validation, but it MUST NOT be sufficient evidence for automatic native-tool bridging.

## Rationale

- Provider-returned metadata updates without an Aivo release and is stronger evidence than model-name inference.
- An explicit-known versus unknown distinction prevents incomplete `/models` responses from silently enabling privileges or disabling compatible requests.
- An allowlisted adapter keeps remote untrusted strings from authoring arbitrary tool declarations or protocol versions.
- Lazy first-use synchronization upgrades already configured Providers and periodically revalidates stale metadata without requiring a settings action, while the freshness window avoids a remote catalog request on every turn.

## Consequences

- Provider-specific model parsers and adapters remain necessary; there is no cross-Provider capability schema.
- The first request after upgrading, connecting, or passing the capability freshness window for a dynamically discoverable Provider may include one bounded model-catalog call.
- Hosted tools can create Provider-side usage charges and must remain visible in Provider diagnostics and request tests.
- Providers that do not expose authoritative native-tool metadata continue to use existing behavior until a trustworthy discovery source is added.

## Rejected alternatives

- Treat every static capability as automatically enabled: silently becomes stale and can expose unsupported or changed Provider protocols.
- Probe every tool by issuing generation requests: adds cost, latency, and side effects and cannot run safely on every startup.
- Require a user toggle for every discovered tool: leaves the supported capability disconnected and contradicts automatic discovery.
- Register hosted tools as local executors: misrepresents execution ownership and bypasses the Provider adapter's server-tool response contract.

## Verification

- `AT-PROVIDER-001`: dynamic parsing, caching, automatic first-use refresh, and cache-preserving failure.
- `AT-TOOL-001`: allowlisted automatic declaration, explicit disablement, unknown-metadata refusal, and Provider serialization.
- `CT-SECURITY-001`, `CT-RELIABILITY-001`: no arbitrary remote tool injection, no local execution authority, bounded retry, and stable fallback behavior.
