# ADR-0026: Bridge Codex-declared local execution tools

- Status: Accepted
- Date: 2026-08-26
- Related Work: `CHG-2026-053-codex-native-tool-bridge`
- Closes OPEN: none

## Context

The authenticated ChatGPT Codex model catalog returns model-owned `shell_type` and `apply_patch_tool_type` declarations. Codex uses these fields to decide whether its client-defined shell and freeform patch tools are compatible with a selected model. Aivo already owns permission-checked local Bash execution and a freeform `apply_patch` implementation, but only the four stable primitives are registered in the default coding environment and the patch implementation is not executable through a normal session snapshot.

These catalog fields describe model/wire compatibility; they do not grant local process or filesystem authority. OpenAI API-key and ChatGPT OAuth connections also share the `openai` Provider identity and persisted model-cache slot, so route authentication must remain part of the decision.

## Decision

- Core MUST parse only the code-owned Codex catalog fields `shell_type` and `apply_patch_tool_type`, retaining an explicitly returned supported or disabled value separately from an unknown, missing, malformed, or unrecognized value.
- `shell_type: "unified_exec"` authorizes serialization of Aivo's existing canonical `bash` tool for that model. `shell_type: "disabled"` explicitly suppresses it. No remote value may create a new command schema or executor.
- `apply_patch_tool_type: "freeform"` authorizes activation and Responses custom-tool serialization of the canonical local `apply_patch` executor. An explicit `null` disables it. Missing, malformed, and unrecognized values remain unknown and MUST NOT activate it.
- This bridge MUST apply only to the built-in OpenAI Provider using an OAuth credential and the ChatGPT Codex Responses transport. OpenAI API-key and custom compatible routes MUST ignore Codex declarations even when they encounter a shared cached model record.
- Core MUST perform the same bounded best-effort first-use catalog synchronization and freshness validation used by other declared-capability Providers. Refresh failure preserves prior cache and cannot newly enable a local tool.
- `read`, `bash`, `edit`, and `write` remain the four stable default primitives. `apply_patch` is a registered non-default Provider-declaration tool: it is not an auxiliary/manual candidate and enters a request and immutable Tool Snapshot only when an eligible route explicitly declares it.
- A declaration changes exposure only. Registry identity, global/session disablement, Agent mode, Execution Environment ownership, immutable Tool Snapshot validation, permission evaluation, cancellation, bounded output, and deterministic teardown remain authoritative for execution.
- Fallback preparation MAY freeze the union of explicitly declared local Provider tools across eligible candidate routes, but each actual Provider request MUST filter that union against its active route. A returned call is executable only when it is also present in the frozen snapshot.

## Rationale

- The authenticated model catalog can evolve without a new Aivo release while a code-owned enum allowlist prevents arbitrary remote tool authoring.
- Reusing canonical Aivo executors preserves permission and audit behavior and avoids a Provider-specific privilege path.
- Route authentication prevents OAuth metadata from changing API-key behavior despite the shared Provider ID.
- A non-default registration preserves the minimal four-tool contract while making the Codex-trained freeform protocol available when explicitly supported.

## Consequences

- Codex OAuth may make one bounded model-catalog request before first use or after the freshness window.
- The coding Registry contains a dormant `apply_patch` registration that is absent from ordinary candidates, Provider requests, and snapshots until route activation.
- A Codex model with missing or unfamiliar declaration metadata receives neither the conditional patch tool nor an assumed shell tool until a recognized declaration arrives.

## Rejected alternatives

- Infer from model names or static tables: becomes stale and cannot represent explicit disablement.
- Treat Codex declarations as hosted tools: execution is local and would misstate ownership and bypass local snapshot semantics.
- Expose `apply_patch` to every Provider or manual resolver: changes the default/selection contract without Provider evidence.
- Alias `apply_patch` to `write`: violates canonical tool identity and does not preserve the freeform patch protocol.

## Verification

- `AT-PROVIDER-001`: catalog parsing, known-versus-supported values, OAuth-only synchronization, freshness, failure preservation, and API-key isolation.
- `AT-TOOL-001`: conditional shell/patch exposure, non-default registration, exact custom-tool serialization, snapshot membership, and permission enforcement.
- `CT-SECURITY-001`, `CT-RELIABILITY-001`: malformed/unknown refusal, no remote schema authoring, fallback filtering, cancellation, and stable cache fallback.
