---
name: openspec-architecture
description: Discuss, create, or update the global OpenSpec architecture document at openspec/architecture.md using option-based technical questions. Use after product direction is defined and before implementation planning, or when tech stack, module boundaries, data model rules, API principles, testing strategy, security constraints, or code organization need to be fixed so AI-assisted coding does not drift.
---

# OpenSpec Architecture

Define the global technical architecture that every later OpenSpec change must respect.

This skill creates or updates:

- `openspec/architecture.md`: durable technical architecture, module boundaries, and code organization rules.
- `openspec/config.yaml` context: concise architecture facts shown to AI when generating artifacts.

It does not create a change and does not generate proposal/spec/design/tasks.

## Workflow

1. Read `openspec/product.md` if it exists for product context.
2. Read `openspec/architecture.md` if it exists.
3. If no architecture document exists, create it from `openspec/templates/architecture.md`.
4. Inspect the codebase when present before asking repo-discoverable questions.
5. Use option-based questions for high-impact architecture choices that cannot be discovered.
6. Update `openspec/architecture.md` with confirmed choices and clearly marked constraints.
7. Add or update concise architecture facts in `openspec/config.yaml` context.
8. Summarize what was established and what still needs confirmation.

## Option-Based Questions

Ask only questions that materially affect code structure or implementation direction.

Use 1-3 questions per round. Each question must include:

- 2-4 mutually exclusive options.
- One recommended option.
- A short impact statement for each option.
- A way for the user to provide a custom answer.

Good architecture question areas:

- Tech stack and runtime boundaries.
- Monolith vs modular monolith vs services.
- Frontend/backend separation.
- Module boundaries and dependency direction.
- Data ownership, persistence, and migration strategy.
- API/interface style.
- State, side effects, retries, and idempotency.
- Auth, permissions, sensitive data, and audit requirements.
- Testing strategy and quality gates.
- Code organization rules.

When a structured question tool is available, use it for blocking architecture choices. If it is unavailable, ask in chat with the same option format and wait for the user. Do not silently decide blocking architecture direction.

## Assumptions

If the user asks for one-shot architecture generation and a missing detail is not blocking, choose the recommended option and mark it under `Constraints / Needs Confirmation`.

Do not present assumptions as confirmed architecture decisions.

## architecture.md Content

Maintain `openspec/architecture.md` as the durable technical baseline.

It should include:

- Architecture overview.
- Tech stack.
- System boundaries.
- Module boundaries.
- Data model principles.
- API / interface principles.
- State, side effects, and error handling rules.
- Security and permissions constraints.
- Testing strategy.
- Code organization rules.
- Architecture decisions.
- Constraints / needs confirmation.

Future feature changes should update this file only when they change architecture-level direction, module boundaries, major dependencies, persistence strategy, or testing policy.

## config.yaml Context

Keep `openspec/config.yaml` context short and reusable.

Include stable architecture facts only:

- Tech stack.
- System shape.
- Core module boundaries.
- Dependency direction.
- Testing expectations.
- Important constraints.

Do not copy the full architecture document into config. The config context should be a short summary; `openspec/architecture.md` is the full source of truth.

## Output

When complete, report:

- Whether `openspec/architecture.md` was created or updated.
- Whether `openspec/config.yaml` context was updated.
- Key confirmed architecture choices.
- Constraints still needing confirmation.
- Suggested next command, usually `/opsx:product-propose <change idea>`.
