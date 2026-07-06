---
name: openspec-init-project
description: Initialize or update an OpenSpec project's global product context. Use when starting a new product, defining project situation, product positioning, target users, roadmap, UX principles, global non-goals, success criteria, or when the user wants AI-assisted development to stay aligned before creating any change proposals.
---

# OpenSpec Init Project

Define the global product context that every later OpenSpec change must respect.

This skill creates or updates:

- `openspec/product.md`: global product planning and direction.
- `openspec/config.yaml` context: concise project facts shown to AI when generating artifacts.

It does not create a change and does not generate change-local `acceptance.md`.

## Workflow

1. Read `openspec/config.yaml`.
2. Read `openspec/product.md` if it exists.
3. If no product document exists, create it from `openspec/templates/product.md`.
4. Use option-based questions for missing high-impact product facts.
5. Update `openspec/product.md` with confirmed choices and clearly marked assumptions.
6. Add or update concise `context:` in `openspec/config.yaml` so future OpenSpec artifacts see the product situation.
7. Summarize what was established and what still needs confirmation.

## Option-Based Questions

Ask only questions that materially affect the whole product, not one feature.

Use 1-3 questions per round. Each question must include:

- 2-4 mutually exclusive options.
- One recommended option.
- A short impact statement for each option.
- A way for the user to provide a custom answer.

Good project-level question areas:

- Product category and positioning.
- Primary target user and explicit non-target user.
- Core problem and job-to-be-done.
- Product scope: internal tool, consumer app, SaaS, developer tool, marketplace, etc.
- Experience direction: dense operations UI, simple consumer app, analytical dashboard, creative tool, etc.
- AI/model responsibility if the product includes AI behavior.
- Roadmap grouping: Now / Next / Later.
- Global non-goals.
- Product-level success criteria.

When a structured question tool is available, use it for blocking project-level choices. If it is unavailable, ask in chat with the same option format and wait for the user. Do not silently decide blocking global product direction.

## Assumptions

If the user asks for one-shot initialization and a missing detail is not blocking, choose the recommended option and mark it under `Assumptions / Needs Confirmation`.

Do not present assumptions as confirmed user decisions.

## product.md Content

Maintain `openspec/product.md` as the durable product baseline.

It should include:

- Product positioning.
- Target users and non-target users.
- Core problem.
- Product vision.
- Core workflows.
- Product principles.
- Roadmap / planned capabilities.
- Experience direction.
- Global non-goals.
- Product success criteria.
- Assumptions / needs confirmation.

Future feature changes should update this file only when they change product-level direction, not for routine implementation details.

## config.yaml Context

Keep `openspec/config.yaml` context short and reusable.

Include stable facts only:

- Product name or working name.
- Product category.
- Primary users.
- Core product principles.
- Important constraints.

Do not copy the full product document into config. The config context should be a short summary; `openspec/product.md` is the full source of truth.

## Output

When complete, report:

- Whether `openspec/product.md` was created or updated.
- Whether `openspec/config.yaml` context was created or updated.
- Key confirmed product choices.
- Assumptions still needing confirmation.
- Suggested next command, usually `/opsx:architecture` or `/opsx:product-propose <change idea>`.
