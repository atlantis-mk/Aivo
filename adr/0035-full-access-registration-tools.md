# ADR-0035: Let full access cover trusted registration tools

- Status: Accepted
- Date: 2026-09-01
- Related Work: `CHG-2026-065-full-access-registration-tools`
- Closes OPEN: none

## Context

The explicit full-access permission mode allows ordinary write and shell tools after Core has prepared and validated their execution context. Conversational MCP and Skill-resource registration tools previously always required a second exact native confirmation, even after the user selected full access. That made the label behave inconsistently for trusted built-in registration executors and produced repeated approval prompts during user-directed installation flows.

## Decision

- Aivo MUST treat `full_access` as permission to run every trusted registered tool whose Core preflight and non-bypassable validation succeed, including `aivo_tools_register_mcp` and `aivo_tools_register_resource`.
- Aivo MUST keep `request_approval` behavior unchanged: conversational registration tools still require exact native confirmation before durable state changes.
- Full access MUST NOT bypass malformed input refusal, sensitive path denial, stale Tool Snapshot refusal, mode/scope denial, credential isolation, proposal expiry, replay refusal, integrity verification, or failed probe/install cleanup.
- Historical `auto_approve` rules MUST continue to resolve to `request_approval`.
- Exact registration approval or denial requests MUST remain non-rememberable when a request is created under request-approval mode.

## Rationale

- The user-facing full-access mode should suppress approval prompts for operations that are otherwise authorized by Core policy.
- Registration proposal validation already runs before permission-mode evaluation, so full access can remove the prompt without removing Host-owned proposal bounds.
- Keeping request-approval exact confirmation preserves the stricter mode for users who want per-operation prompts.

## Consequences

- Full-access sessions can install a validated Skill resource or register a validated MCP source without an additional approval card.
- Security tests must cover the distinction between skipping the prompt and skipping validation.
- No persistence migration is required because the existing permission-mode rule remains the authority.

## Rejected Alternatives

- Keep exact confirmation for registration tools in full access: this preserves the old safety prompt but contradicts the explicit full-access label and user expectation.
- Reintroduce automatic approval: this would recreate an ambiguous third mode already removed by ADR-0030.

## Verification

`AT-TOOL-001`, `AT-EXTENSION-001`, `AT-EXTENSION-002`, and `CT-SECURITY-001` cover full-access prompt suppression, request-approval confirmation, validation failure behavior, stale proposal refusal, and historical `auto_approve` fail-closed behavior.
