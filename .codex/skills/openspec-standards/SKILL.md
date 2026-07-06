---
name: openspec-standards
description: Discuss, create, or update the global OpenSpec engineering standards document at openspec/standards.md using option-based questions. Use after product and architecture context exist, or when frontend, backend, language-specific, testing, review, Definition of Done, dependency, error handling, or anti-pattern rules need to be fixed so AI-assisted coding does not produce messy or inconsistent code.
---

# OpenSpec Standards

Define stack-specific engineering standards that every later implementation must follow.

This skill creates or updates:

- `openspec/standards.md`: durable coding, testing, review, and Definition of Done rules.
- `openspec/config.yaml` context: concise standards facts shown to AI when generating artifacts.

It does not create a change and does not generate proposal/spec/design/tasks.

## Workflow

1. Read `openspec/product.md` if it exists.
2. Read `openspec/architecture.md` if it exists.
3. Inspect the codebase and manifests to discover actual languages, frameworks, package managers, test tools, linters, and folder conventions.
4. Read `openspec/standards.md` if it exists.
5. If no standards document exists, create it from `openspec/templates/standards.md`.
6. Ask option-based questions only for high-impact standards choices that cannot be discovered.
7. Generate stack-specific sections for the actual project, such as frontend, backend, API, database, tests, and tooling.
8. Update concise standards facts in `openspec/config.yaml` context.
9. Summarize confirmed standards and items needing confirmation.

## Option-Based Questions

Ask only questions that materially affect code quality or consistency.

Use 1-3 questions per round. Each question must include:

- 2-4 mutually exclusive options.
- One recommended option.
- A short impact statement for each option.
- A way for the user to provide a custom answer.

Good standards question areas:

- Frontend component boundaries and state management.
- Backend service/module layering.
- Language-specific strictness, typing, and error handling.
- API validation and response shape.
- Database migration and query rules.
- Testing pyramid and required checks.
- Dependency approval rules.
- Review checklist and Definition of Done.
- Anti-patterns to ban.

When a structured question tool is available, use it for blocking standards choices. If it is unavailable, ask in chat with the same option format and wait for the user. Do not silently decide blocking engineering standards.

## Stack-Specific Generation

Generate only sections relevant to the detected or chosen stack.

Examples:

- React/TypeScript frontend: component size, hook usage, state ownership, form validation, API boundary, accessibility, styling conventions.
- Node/TypeScript backend: route/service/repository layering, schema validation, async error handling, logging, dependency injection boundaries.
- Python backend: module boundaries, type hints, Pydantic/dataclass rules, exception handling, database/session boundaries.
- Database: migrations, indexes, identifiers, transactions, seed/test data.
- Tests: unit/integration/e2e split, test naming, fixture strategy, required commands.

If stack is unknown, include a short `Stack Decisions / Needs Confirmation` section instead of inventing detailed rules.

## standards.md Content

Maintain `openspec/standards.md` as the durable engineering baseline.

It should include:

- Standards scope and stack summary.
- Universal coding standards.
- Stack-specific frontend standards when relevant.
- Stack-specific backend standards when relevant.
- Data/API standards when relevant.
- Testing standards.
- Definition of Done.
- Review checklist.
- Banned anti-patterns.
- Tooling commands.
- Needs confirmation.

Future feature changes should update this file only when they change engineering conventions, tools, quality gates, or code review policy.

## config.yaml Context

Keep `openspec/config.yaml` context short and reusable.

Include stable standards facts only:

- Primary languages/frameworks.
- Required quality gates.
- Important banned patterns.
- Test commands when known.

Do not copy the full standards document into config. The config context should be a short summary; `openspec/standards.md` is the full source of truth.

## Output

When complete, report:

- Whether `openspec/standards.md` was created or updated.
- Whether `openspec/config.yaml` context was updated.
- Detected or chosen stack sections.
- Key confirmed standards.
- Standards still needing confirmation.
- Suggested next command, usually `/opsx:product-propose <change idea>`.
