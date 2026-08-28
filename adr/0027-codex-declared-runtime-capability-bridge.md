# ADR-0027: Bridge Codex-declared runtime capabilities

- Status: Accepted
- Date: 2026-08-26
- Related Work: `CHG-2026-055-codex-declared-runtime-capabilities`
- Closes OPEN: none

## Context

The authenticated ChatGPT Codex model catalog carries request-compatibility metadata beyond the local shell and patch declarations covered by ADR-0026. Current models declare context limits, reasoning levels, input modalities, service tiers, verbosity, parallel-tool support, web-search representation, and whether the Responses Lite wire contract is required. Aivo currently ignores most of those fields, relies on static model search metadata, and can serialize hosted search only when another local tool specification happens to seed the request.

The catalog also contains remote instruction text, experimental tool names, Codex-specific tool modes, and multi-agent policy. Those values are not protocol compatibility facts: consuming them directly would let remote metadata replace Aivo prompts, create executable identities, or change Host-owned Agent and authorization policy.

## Decision

- Core MUST parse allowlisted Codex catalog fields through a typed code-owned adapter and retain unknown, malformed, and unrecognized values without inferring support.
- Core MUST use declared context length, reasoning efforts/default, input modalities, service tiers/default, verbosity/default, parallel-tool support, image-detail support, web-search type, and Responses Lite selection only on the built-in OpenAI OAuth Codex route.
- A recognized `web_search_tool_type` MUST automatically expose the code-owned Codex search path when web search is enabled. Ordinary Responses uses the Provider-hosted `web_search` declaration; Responses Lite uses Aivo's existing permission-controlled canonical `web_search` executor backed by the authenticated Codex `alpha/search` operation because Lite does not accept hosted tools.
- Search mode MUST preserve `cached`, `indexed`, `live`, and `disabled` semantics. `text_and_image` MAY request text and image results; unrecognized search types MUST NOT enable search.
- Responses Lite requests MUST use the required header, place client tools in a developer `additional_tools` input item, omit top-level hosted tools, disable parallel tool calls, and request all-turn reasoning context. Aivo MUST NOT select Lite through model-name inference.
- Model-declared reasoning efforts and service tiers MUST constrain values sent on the wire. Unsupported user preferences fall back to the declared default or are omitted rather than being translated to a different effort.
- Structured URL citations and search-result URLs MUST be deduplicated, bounded, and retained with the final assistant text without logging raw Provider payloads.
- Catalog `model_messages`, `experimental_supported_tools`, `tool_mode`, `multi_agent_version`, auto-review flags, and instruction-inclusion flags MUST NOT create tools, replace prompts, alter Agent policy, or grant authority. Supporting a future field requires a new explicit code mapping and applicable governance review.
- OpenAI API-key and custom compatible routes MUST ignore Codex-only declarations even if they share a cached model ID.

## Rationale

- Typed allowlists allow the authenticated catalog to evolve while preventing remote capability authoring.
- Separating ordinary hosted search from Lite standalone search follows the Provider wire contracts and avoids registering a new default tool.
- Persisting model facts once lets model selection, compaction, attachment validation, and request construction use one authoritative source.

## Consequences

- First use can perform the existing bounded Codex catalog refresh before request preparation.
- New Codex models become usable without a static Aivo model release when their declarations use recognized values.
- Aivo continues to own prompts, Agent modes, tool identities, Registry membership, snapshots, permissions, and execution.

## Rejected alternatives

- Union dynamic declarations with static capability tables: explicit changes and new models remain stale.
- Register web search as an unconditional default local tool: changes the four-primitive contract and grants network exposure without route evidence.
- Execute arbitrary `experimental_supported_tools`: remote strings would become executable authority.
- Import `model_messages` as system policy: Provider metadata would replace the validated Aivo prompt boundary.

## Verification

- `AT-PROVIDER-001`: complete Codex metadata parsing, persistence, merge precedence, freshness, and OAuth/API-key isolation.
- `AT-TOOL-001`: automatic hosted/Lite search exposure, search modes/content types, snapshot and permission enforcement, and unknown-value refusal.
- `AT-SESSION-001`: request parameter constraints, Lite body/header compatibility, source preservation, streaming deduplication, and fallback behavior.
- `CT-SECURITY-001`, `CT-RELIABILITY-001`: remote policy fields remain inert, responses are bounded, failed discovery preserves cache, and no declaration bypasses Host authority.
