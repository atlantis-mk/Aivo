# ADR-0021: Use a Core-owned validated file-backed prompt registry

- Status: Accepted
- Date: 2026-08-23
- Related Work: `CHG-2026-038-prompt-management`
- Closes OPEN: none

## Context

Prompt text is split among code literals, renderer quick actions, project files, and SQLite Agent-mode definitions. Users need editable Markdown without making the unprivileged renderer or arbitrary files runtime authority. ADR-0013 rejected renderer-owned Markdown editing and assigned complete Agent definitions to SQLite; a privileged typed file registry now provides the missing safe ownership model.

## Decision

- Core MUST own built-in prompt contracts, managed global files, validation, active revisions, reference checks, and runtime rendering.
- Built-in defaults MUST be embedded Markdown. Global overrides MUST be bounded regular files below the platform prompt root; Renderer MUST access them only through typed operations.
- Invalid working content MUST NOT replace the last validated revision. An execution MUST retain one immutable prompt snapshot.
- Prompt policy, structured wrappers, permissions, tools, credentials, and output parsers MUST remain code-owned and MUST NOT be weakened by prompt text.
- Agent-mode rows MUST reference prompt IDs rather than own prompt bodies after schema v9. Project runtime files remain the final project-scoped overlay.
- Required prompts MUST NOT be disabled or deleted. Optional built-ins MAY be disabled, and unreferenced custom Agent/quick prompts MAY be deleted.

## Rationale

- Markdown gives users a transparent editable format while Core validation and content-addressed active revisions prevent partial or invalid injection.
- Embedded defaults preserve upgrade/reset behavior; removable overrides avoid copying defaults into durable user state.
- Stable prompt IDs remove duplicate body ownership while keeping Agent reference validation transactional.

## Consequences

- Prompt storage gains a managed filesystem lifecycle plus schema-v9 Agent payload migration and recovery tests.
- Hidden worker metadata remains protected under ADR-0013, while its prompt document may be edited/reset through this separately validated registry.
- Old clients may submit prompt text, but Core converts it to a document and does not persist it in the Agent row.

## Rejected alternatives

- Renderer edits arbitrary Markdown paths: violates the privilege boundary and cannot provide safe activation or reference checks.
- SQLite stores every prompt body: fails the requested Markdown source of truth and external editing workflow.
- Live filesystem watching: adds cross-platform partial-write and concurrency races without being required for the first release.

## Verification

`AT-PROMPT-001`, `AT-AGENT-001`, `AT-WORKSPACE-001`, `CT-SECURITY-001`, `CT-RELIABILITY-001`, and `AT-UI-001`.
