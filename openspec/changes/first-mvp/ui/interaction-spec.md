# first-mvp UI Interaction Spec

## Referenced OpenSpec Files

- `openspec/product.md`
- `openspec/config.yaml`
- `openspec/changes/first-mvp/proposal.md`
- `openspec/changes/first-mvp/design.md`
- `openspec/changes/first-mvp/specs/**/*.md`
- `openspec/changes/first-mvp/tasks.md`

## Design Source

- Pencil file: `openspec/changes/first-mvp/ui/design.pen`
- Screenshots:
  - `ui/screenshots/00-welcome-first-run-entry.png`
  - `ui/screenshots/01-initialize-provider-first-run.png`
  - `ui/screenshots/02-project-selection-after-provider-setup.png`
  - `ui/screenshots/03-desktop-workbench-running-task.png`
  - `ui/screenshots/04-plan-review-awaiting-approval.png`
  - `ui/screenshots/05-sensitive-action-confirmation.png`
  - `ui/screenshots/06-completed-task-review.png`
  - `ui/screenshots/07-narrow-layout-task-review.png`

## Screen Inventory

1. `00 Welcome - Minimal Assistant Entry`: first-run welcome screen with centered assistant greeting, six capability entry points, and a single primary next action.
2. `01 Initialize Provider - First Run`: provider configuration with direct choices for OpenAI, Claude Code, Gemini, Codex-compatible, and Custom API, including validation, blocked state, and remote-transfer notice.
3. `02 Project Selection - App Shell`: shallow desktop app shell after provider setup, with left project/conversation navigation and a centered task composer bound to the selected project.
4. `02B Project Selection - Prompt Submitted`: post-submit state derived from the project selection shell, with the composer pinned near the bottom and the submitted prompt plus assistant return/thinking content above it.
5. `03 Conversation Workbench - Running`: primary conversation workbench for the selected project, active task output, status rows, and bottom composer.
6. `04 Plan Approval Modal`: plan approval shown as a centered modal over the app shell before local side effects run.
7. `05 Sensitive Action Modal`: scoped sensitive-action confirmation shown as a centered modal over the app shell.
8. `06 Completed Task Review - Conversation`: completed task review inside the conversation workbench with delivery notes and follow-up composer.
9. `07 Narrow Layout - Conversation`: responsive narrow conversation layout with top title, task result body, and bottom composer.

## User Flow Summary

The user starts at a minimal centered welcome screen when setup is incomplete. `下一步` advances to provider setup, where the user directly selects OpenAI, Claude Code, Gemini, Codex-compatible, or Custom API. Official/common providers show simpler provider-specific fields. Custom API shows protocol, base URL, credential reference, and model/profile fields. After one provider validates, Aivo advances into the shallow app shell: the left sidebar shows projects and recent conversations, while the main area shows a centered task composer for the selected project. Submitting a task opens the conversation workbench. Aivo creates a task, shows a proposed plan, and waits for explicit plan approval in a modal over the shell. Approved execution moves into the conversation output. Sensitive actions pause execution and open a confirmation modal over the same shell. The user can approve, deny, cancel the task, inspect logs/artifacts during execution, and reopen completed or interrupted tasks after restart.

## State Matrix

| Surface | Default | Loading | Empty | Error | Success | Disabled / Permission |
| --- | --- | --- | --- | --- | --- | --- |
| Welcome/setup entry | Welcome screen with setup progress and primary initialization action | App setup state loading | Not applicable | Setup state read error | Advances to provider setup | Workbench skipped until setup completion |
| Provider initialization | Provider list and provider-specific configuration panel | Validation in progress | No provider configured | Safe validation error or unavailable local command | Provider ready and continue enabled | Task creation blocked until provider validates |
| Project selector | Left project list plus centered task composer for the selected project | Skeleton rows in sidebar/main picker | Empty project list with open-folder action | Safe inaccessible-path message | Selected project context shown in composer | Task composer disabled until project selected |
| Task composer | Textarea/input with primary create action | Create action shows pending state | Empty text validation message | Safe task creation error | Task record appears with initial status | Disabled when no project selected or task is already running |
| Plan review | Centered modal over the app shell with ordered proposed steps and approve/decline actions | Plan generation indicator | No plan attached message | Plan generation/attach failure | Approved plan badge and execution starts | Local side effects blocked before approval |
| Execution timeline | Ordered steps and log rows | Live progress row | No logs yet message | Failed step row with safe error summary | Completed step rows and artifacts | New steps blocked while task canceled or waiting confirmation |
| Confirmation gate | Centered modal over the app shell with action, scope, and approval controls | Approval/denial action pending | No pending confirmations | Invalidated or stale confirmation state | Approved/denied history visible | Sensitive action blocked until approved |
| Artifacts/review | Artifact list, diff/log/note entries | Artifact loading row | No artifacts yet | Missing artifact reference message | Saved artifacts and verification summary | Secret values redacted |
| Completed task | Review summary, delivery notes, artifacts | History loading | No completed tasks | Recovery/read error | Reopenable task history | Destructive follow-up actions require confirmation |

## Interaction Rules

- Project selection:
  - Project selection is not available until provider initialization has completed.
  - The project selector uses the same shallow app shell as the workbench: left sidebar for projects/recent conversations, main area for task entry.
  - `Open project` launches the native/local directory picker.
  - Invalid or inaccessible paths show an inline error and do not create a project record.
  - Non-Git directories are allowed and show Git metadata as unavailable.

- Task creation:
  - Empty or whitespace-only task descriptions show validation text and do not create a task.
  - Task creation is blocked until provider setup is complete and a project is selected.
  - Submitted tasks are associated with the selected project ID.
  - After the user submits from the project selection composer, the composer moves to the lower main surface, the submitted text is shown as a user message above it, and assistant return content starts in the same surface with a visible thinking/loading state.

- Welcome and provider initialization:
  - `下一步` moves from the minimal welcome screen to provider setup.
  - Capability entries on the welcome screen are explanatory entry points for what Aivo can help with; they do not bypass provider setup in the MVP.
  - Provider setup shows direct choices for OpenAI, Claude Code, Gemini, Codex-compatible, and Custom API.
  - Direct provider choices use provider-specific defaults and fewer fields.
  - Custom API fields include display name, protocol, base URL, credential reference, and default model or profile.
  - API-backed providers show an external data-transfer notice before validation and before task execution.
  - Local command-backed providers show local installation/availability requirements.
  - `Validate provider` checks readiness and stores only non-secret metadata plus credential reference labels.
  - Raw API keys, tokens, and shell secrets must not be persisted in SQLite, logs, screenshots, or generated artifacts.

- Plan approval:
  - Plan approval is the required checkpoint before local side effects and appears as a modal over the current app shell.
  - `Approve plan` transitions to execution.
  - `Decline` stops the task or returns it to a revision state without executing side effects.
  - Sensitive actions still require separate confirmations after plan approval.

- Sensitive confirmations:
  - Sensitive confirmations appear as modals over the current app shell, not as separate full-screen pages.
  - Confirmation UI must show action type, target, proposed effect, risk summary, and project/task scope.
  - `Approve` authorizes only the exact action shown.
  - `Deny` records denial and prevents execution.
  - Changed action details or task cancellation invalidate pending confirmations.
  - Credential values and secrets must be redacted; show references or labels only.

- Task execution:
  - Running steps emit visible timeline/log rows.
  - Failed steps show safe, contextual error summaries and preserve logs.
  - `Cancel` propagates app-layer cancellation and prevents new steps from starting.
  - Restarted sessions reopen waiting, failed, canceled, running, and completed tasks from persisted state.

- Keyboard/focus:
  - Primary controls are keyboard reachable in visual order.
  - Modal confirmation traps focus while open and returns focus to the invoking task surface on close.
  - Destructive actions use distinct styling and require explicit button activation.

## Implementation Notes

- Use shadcn/ui components as the baseline: Button, Dialog, Card/Panel equivalents, Badge, Textarea/Input, Select or segmented controls for Custom API protocol selection, Alert, and Table/List patterns.
- Use TanStack Router for first-run setup, provider setup, workbench, task review, artifact review, and settings navigation as those surfaces are implemented.
- Keep route modules under frontend `src/routes`, feature UI and hooks under `src/features/*`, typed Electron clients under `src/services`, shared shadcn/ui components under `src/components`, and small shared utilities under `src/lib`.
- Configure shadcn/ui with the `new-york` style, CSS variables, `lucide-react` icons, and project aliases matching the frontend source layout.
- Keep raw Aivo bridge handlers behind typed frontend services/hooks; React components consume typed view models.
- Use the current Aivo visual language: mostly white surfaces, restrained light gray sidebar and composer surfaces, black primary send/continue actions, blue accents for task artifacts/status, orange accent for permission/access state.
- Design frames represent the Electron web content area only. Do not implement an artificial desktop wallpaper, black outer border, rounded app-window shell, shadowed window container, or custom minimize/maximize/close controls; native Electron/system window chrome owns those details.
- First-run welcome and provider initialization remain minimal centered setup surfaces.
- After provider setup, desktop layout uses two dominant zones: a persistent left sidebar for projects/conversations and a main conversation/task surface with centered content and bottom composer.
- Plan approval and sensitive action confirmation are modal overlays on top of the app shell.
- Narrow layout keeps the conversation hierarchy: top title, task result body, and bottom composer; sidebar content moves behind compact navigation in implementation.
- Avoid marketing hero treatment; the first screen remains a usable project/task workbench.
- Persisted data dependencies: setup state, provider metadata, credential reference labels, projects, task status, steps/tool runs, confirmations, artifacts, logs, verification results, and resumable state.

## Requirement Traceability

| Requirement area | UI coverage |
| --- | --- |
| Runnable desktop/welcome/workbench foundation | Screens 00, 01, 02, 03, and 07 show setup, app shell, conversation workbench, and responsive states. |
| First-run welcome | Screen 00 covers setup entry and setup progress. |
| Provider configuration | Screen 01 covers direct choices for OpenAI, Claude Code, Gemini, Codex-compatible, plus Custom API with protocol selection, validation, blocked state, and remote-transfer notice. |
| Common loading/empty/error/success states | State matrix defines required states; Screen 02 covers selected-project context and blocked/empty project behavior. |
| Open local project/list recent projects | Screen 02 and Screen 03 sidebar project/conversation patterns. |
| Git/non-Git repository metadata | Screen 03 selected project metadata uses branch and dirty status; non-Git becomes unavailable metadata text. |
| Project context required for tasks | Screen 02 blocked task creation state. |
| Create task and reject empty task | Screen 02B shows immediate submitted-input feedback; Screen 03 composer and Screens 01/02 disabled or blocked behavior. |
| Review plan before execution | Screen 04 modal plan approval checkpoint. |
| Observable task execution | Screen 03 conversation output and step status. |
| Logs/artifacts/diffs/verification | Screens 03 and 06 conversation result/review surfaces. |
| Cancel/resume/recover task | Screen 03 cancel/resume controls; Screen 06 completed reopenable review; state matrix covers recovery. |
| Identify and confirm sensitive actions | Screens 03 pending confirmation card and 05 modal. |
| Block execution until approval | Screen 05 modal and interaction rules. |
| Invalidate stale confirmations | State matrix and confirmation interaction rules. |
| Preserve local confirmation history without secrets | Screens 03/06 review surfaces and redaction rule. |
| SQLite persistence/restart recovery | State matrix and completed/recovered task review behavior. |

## Open Questions / Assumptions

- Assumption: the MVP desktop target is at least 1440px wide for the primary three-zone layout.
- Assumption: narrow layout at 390px represents compact desktop content width; implementation may use sheets/drawers for sidebar/inspector.
- Product mark is `Aivo` and bundle identity is `com.aivo.desktop`; final app icon remains pending.
- Direct provider adapters validate provider-specific defaults plus credential reference and selected model/profile. Custom API adapters validate protocol, custom base URL, credential reference, and selected model/profile with a non-secret readiness check when available.
