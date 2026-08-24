# ADR-0019: Generate MCP descriptions from a safe complete tool projection

- Status: Accepted
- Date: 2026-08-11
- Related Work: `CHG-2026-035-mcp-description-generation`
- Closes OPEN: none

## Context

Source-group auxiliary selection benefits from concise MCP descriptions, while a meaningful description should reflect the source's actual discovered capabilities. MCP configuration may contain credentials, private endpoints, commands, arguments, environment values, headers, and filesystem roots that are unnecessary for summarization. Discovered tool descriptions are also untrusted external data. The application therefore needs an explicit boundary between the complete capability evidence used for generation and configuration or authority data that must remain local.

## Decision

- Description generation MUST be an explicit action available for an existing MCP source in its edit surface and MUST return an unsaved draft only.
- Core MUST load the source's complete current stored discovered-tool catalog and MUST project only each tool's bounded name and description into the auxiliary-model request.
- Core MUST NOT include endpoint, transport command or arguments, environment, headers, roots, authentication material, credential references, or other MCP configuration in that request.
- Tool names and descriptions MUST be delimited and treated as untrusted data, never as instructions. The auxiliary request MUST declare no executable tools.
- A tool-less source or a catalog that cannot represent every tool within Host-owned count and size bounds MUST fail explicitly; Core MUST NOT silently truncate members or infer a description from the MCP source name.
- The operation MUST use the configured auxiliary model, inherit request cancellation, normalize the result to bounded plain text, and MUST NOT mutate the MCP source or its tools.
- Only the existing explicit MCP save operation MAY persist a generated draft.

## Rationale

The stored discovery catalog is the closest Host-owned evidence of what the source currently offers. A complete, minimal projection produces useful summaries without coupling the model to connection details or granting it authority. Draft-only behavior preserves user control and makes retries and provider failures reversible.

## Consequences

Tool metadata is disclosed to the user's configured auxiliary provider when the user invokes generation, so the UI action and failure states must be clear. Very large catalogs are refused instead of partially summarized. Stale catalogs remain possible until the existing probe/refresh flow runs; generation deliberately does not add network or process activity.

## Rejected alternatives

- Send the full MCP server DTO: exposes unrelated private configuration and increases prompt-injection surface.
- Generate from only the MCP name or current description: can hallucinate capabilities and does not satisfy complete-tool summarization.
- Chunk, summarize, and merge automatically: adds multiple provider calls and partial-failure semantics without a current product requirement.
- Save immediately after generation: removes user review and couples a fallible provider call to durable configuration mutation.

## Verification

`AT-EXTENSION-001` verifies complete catalog projection, normalized draft output, edit-form behavior, and non-mutation. `CT-SECURITY-001` verifies exclusion of MCP configuration and executable tools plus untrusted-data framing. `CT-RELIABILITY-001` verifies refusal, provider failure, cancellation, and repeatability. Repository gates are `pnpm docs:check`, `pnpm test:core`, `pnpm lint`, and `pnpm build`.
