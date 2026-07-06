## Standards Scope

<!-- What parts of the system these standards apply to. -->

## Stack Summary

<!-- Detected or chosen languages, frameworks, UI component systems, package managers, test tools, linters, and formatters. -->

| Area | Stack / Tooling | Notes |
| --- | --- | --- |
| Frontend | <!-- e.g. React + TypeScript --> | <!-- notes --> |
| Backend | <!-- e.g. Node + Fastify --> | <!-- notes --> |
| Database | <!-- e.g. Postgres + Prisma --> | <!-- notes --> |
| Testing | <!-- e.g. Vitest + Playwright --> | <!-- notes --> |

## Universal Coding Standards

- Keep implementation aligned with `openspec/architecture.md`.
- Keep product behavior aligned with `openspec/product.md`.
- Prefer small, cohesive modules over large mixed-responsibility files.
- Do not duplicate business logic across layers or features.
- Handle errors explicitly with useful context.
- Do not introduce new dependencies without a documented reason.
- Do not add global state or cross-layer calls unless allowed by `openspec/architecture.md`.

## Frontend Standards

<!-- Include only if the project has a frontend. Adapt to the detected framework/language. -->

- Keep business logic out of presentational components.
- Use the chosen UI component system and its documented composition patterns before creating custom primitives.
- Keep component state local unless shared state is required by the workflow.
- Keep API access behind a clear client/service boundary.
- Handle loading, empty, error, and success states for user-facing flows.
- Preserve accessibility basics: labels, keyboard reachability, and meaningful error text.

## Backend Standards

<!-- Include only if the project has a backend. Adapt to the detected framework/language. -->

- Keep route/controller code thin.
- Put business rules in service/domain modules.
- Validate inputs at system boundaries.
- Keep database access behind repository/query boundaries when the architecture calls for it.
- Make side effects explicit and testable.

## Data and API Standards

<!-- Include if the change touches persistence, APIs, events, jobs, or integrations. -->

- Use stable identifiers and explicit ownership.
- Make migrations reversible or document rollback limitations.
- Keep request/response shapes explicit.
- Handle authorization and sensitive data at boundaries.
- Preserve idempotency for retries where required.

## Testing Standards

- Add or update tests for every changed spec scenario that can be automated.
- Cover non-happy-path behavior when the scenario includes errors, empty states, permissions, retries, or validation.
- Prefer focused tests close to the changed behavior.
- Keep fixtures readable and deterministic.
- Do not mark implementation complete until relevant tests pass or unrun tests are explicitly reported.

## Definition of Done

A task is done only when:

- Implementation satisfies the relevant spec scenarios.
- Relevant tests pass, or unrun tests are explicitly reported with reason.
- Lint/type/check commands pass where available.
- Code follows `openspec/architecture.md` and this standards document.
- No obvious duplicate code, dead code, or temporary workaround remains unmarked.
- Errors, empty states, and edge cases relevant to the scenario are handled.
- Task checkbox is updated only after verification.

## Review Checklist

- Does this violate `openspec/architecture.md`?
- Does this violate stack-specific standards in this document?
- Does frontend UI follow the chosen component system, tokens, aliases, and styling conventions?
- Does this introduce a new pattern where an existing one should be used?
- Is any business logic duplicated?
- Are functions/components/modules too large or mixed-responsibility?
- Are errors and edge cases handled explicitly?
- Are tests meaningful and tied to spec scenarios?
- Are any spec scenarios unverified?

## Banned Anti-Patterns

- Cross-layer calls that bypass the architecture.
- Giant components, services, or utility modules.
- Silent catch blocks or swallowed errors.
- Broad `any`/untyped data at important boundaries when the language supports stronger types.
- Hardcoded configuration that belongs in environment/config.
- Duplicate implementations of the same business rule.
- New global state without an architecture decision.
- New dependency without rationale.

## Tooling Commands

<!-- Fill with actual project commands when known. -->

- Install: <!-- command -->
- Lint: <!-- command -->
- Typecheck: <!-- command -->
- Test: <!-- command -->
- E2E: <!-- command -->
- Build: <!-- command -->

## Needs Confirmation

<!-- Standards choices that were recommended but not explicitly confirmed. -->
