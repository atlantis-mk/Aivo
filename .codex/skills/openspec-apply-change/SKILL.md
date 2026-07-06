---
name: openspec-apply-change
description: Implement tasks from an OpenSpec change. Use when the user wants to start implementing, continue implementation, or work through tasks.
license: MIT
metadata:
  author: openspec
  version: "1.0"
  generatedBy: "1.3.1"
---

Implement tasks from an OpenSpec change.

**Input**: Optionally specify a change name. If omitted, check if it can be inferred from conversation context. If vague or ambiguous you MUST prompt for available changes.

**Steps**

1. **Select the change**

   If a name is provided, use it. Otherwise:
   - Infer from conversation context if the user mentioned a change
   - Auto-select if only one active change exists
   - If ambiguous, run `openspec list --json` to get available changes and use the **AskUserQuestion tool** to let the user select

   Always announce: "Using change: <name>" and how to override (e.g., `/opsx:apply <other>`).

2. **Check status to understand the schema**
   ```bash
   openspec status --change "<name>" --json
   ```
   Parse the JSON to understand:
   - `schemaName`: The workflow being used (e.g., "spec-driven")
   - Which artifact contains the tasks (typically "tasks" for spec-driven, check status for others)

3. **Get apply instructions**

   ```bash
   openspec instructions apply --change "<name>" --json
   ```

   This returns:
   - `contextFiles`: artifact ID -> array of concrete file paths (varies by schema - could be proposal/specs/design/tasks or spec/tests/implementation/docs)
   - Progress (total, complete, remaining)
   - Task list with status
   - Dynamic instruction based on current state

   **Handle states:**
   - If `state: "blocked"` (missing artifacts): show message, suggest using openspec-continue-change
   - If `state: "all_done"`: congratulate, suggest archive
   - Otherwise: proceed to implementation

4. **Read context files and inject global context**

   Read every file path listed under `contextFiles` from the apply instructions output.
   The files depend on the schema being used:
   - **spec-driven**: proposal, specs, design, tasks
   - Other schemas: follow the contextFiles from CLI output

   Before implementation, explicitly check for these global OpenSpec context files:
   - `openspec/product.md`
   - `openspec/architecture.md`
   - `openspec/standards.md`

   For each global context file that exists:
   - Read the full file into the working context before making code changes.
   - Extract the concrete product constraints, architecture constraints, standards, forbidden patterns, testing expectations, and Definition of Done items that affect implementation.
   - Treat those extracted points as active implementation constraints, not optional background reading.
   - If a global file is long, still read it first; then keep a concise working summary in context for the rest of the implementation.

   For each global context file that is missing:
   - Record it as "not present" in the progress output.
   - Continue normally; missing global context is not a blocker by itself.

   Extract all `#### Scenario:` blocks from specs files. Treat them as the verification checklist for the implementation.

   Also check for optional UI handoff context at `openspec/changes/<name>/ui/`.
   If the directory exists, use it as implementation reference with this priority:
   - If `ui/design.pen` exists, treat it as the authoritative UI design source.
   - Access `ui/design.pen` only through Pencil MCP tools. Do not read, grep, copy, or edit `.pen` files with filesystem tools.
   - Use Pencil MCP to inspect the actual layout, tokens, components, responsive behavior, and visual states from `design.pen` before implementing UI.
   - Treat design content as reference material for structure, hierarchy, spacing, component states, interaction intent, and realistic content density.
   - Do not copy sample records, mock metrics, placeholder names, fake IDs, fake statuses, or illustrative business data from the UI design into production code unless the OpenSpec requirements explicitly require that exact content.
   - Bind implemented UI to the app's real data sources, fixtures, API contracts, seed data, or existing mock infrastructure according to the codebase conventions.
   - If the design shows example data but the implementation data contract is unclear, pause and ask or derive the contract from specs/backend code; do not invent persistent mock data just to match the design.
   - Use text/Markdown files such as `interaction-spec.md` only as behavioral notes that complement `design.pen`, not as the visual source of truth.
   - Use exported screenshots or images only as secondary visual QA/reference material. Do not implement an approximate UI from screenshots when `design.pen` is available.
   - If `ui/design.pen` does not exist, fall back to available UI handoff files such as `interaction-spec.md` and screenshots.
   - If `ui/` does not exist, continue normally; absence of UI handoff context is not a blocker.

5. **Show current progress**

   Display:
   - Schema being used
   - Global context loaded: product, architecture, standards, or "not present" for each missing file
   - Progress: "N/M tasks complete"
   - Remaining tasks overview
   - Dynamic instruction from CLI

6. **Implement tasks (loop until done or blocked)**

   For each pending task:
   - Show which task is being worked on
   - Make the code changes required
   - Keep changes minimal and focused
   - Keep changes aligned with the loaded `openspec/product.md`, `openspec/architecture.md`, and `openspec/standards.md` constraints
   - Use UI design data only to understand visual density and state examples; do not hardcode design sample data unless required by specs or established project fixtures
   - Track which spec scenarios the task is expected to satisfy
   - Mark task complete in the tasks file: `- [ ]` → `- [x]`
   - Continue to next task

   **Pause if:**
   - Task is unclear → ask for clarification
   - Implementation reveals a design issue → suggest updating artifacts
   - Error or blocker encountered → report and wait for guidance
   - User interrupts

7. **On completion or pause, show status**

   Display:
   - Tasks completed this session
   - Overall progress: "N/M tasks complete"
   - Tests run and their results
   - Spec scenarios verified, not verified, or requiring manual verification
   - Engineering standards review summary
   - If all done: suggest archive
   - If paused: explain why and wait for guidance

   When all tasks are complete:
   - Run the relevant test suite or the most focused available tests.
   - If no test command is discoverable, explicitly report that tests were not run and why.
   - Self-check every spec `#### Scenario:` against the implementation.
   - Self-check implementation against `openspec/standards.md` when present.
   - Do not claim the change is fully complete unless tests ran successfully or the unrun tests are explicitly reported, and every spec scenario is either verified or listed as requiring manual verification.

**Output During Implementation**

```
## Implementing: <change-name> (schema: <schema-name>)

Working on task 3/7: <task description>
[...implementation happening...]
✓ Task complete

Working on task 4/7: <task description>
[...implementation happening...]
✓ Task complete
```

**Output On Completion**

```
## Implementation Complete

**Change:** <change-name>
**Schema:** <schema-name>
**Progress:** 7/7 tasks complete ✓

### Completed This Session
- [x] Task 1
- [x] Task 2
...

### Tests
- `<command>`: passed/failed/not run

### Spec Scenario Verification
- `<scenario>`: verified by `<test/manual check>`
- `<scenario>`: requires manual verification because `<reason>`

### Engineering Standards Review
- Architecture alignment: pass/fail/notes
- Standards alignment: pass/fail/notes
- Known risks: none / `<risk>`

All tasks complete! Ready to archive this change.
```

**Output On Pause (Issue Encountered)**

```
## Implementation Paused

**Change:** <change-name>
**Schema:** <schema-name>
**Progress:** 4/7 tasks complete

### Issue Encountered
<description of the issue>

**Options:**
1. <option 1>
2. <option 2>
3. Other approach

What would you like to do?
```

**Guardrails**
- Keep going through tasks until done or blocked
- Always read context files before starting (from the apply instructions output)
- If task is ambiguous, pause and ask before implementing
- If implementation reveals issues, pause and suggest artifact updates
- Keep code changes minimal and scoped to each task
- Update task checkbox immediately after completing each task
- Pause on errors, blockers, or unclear requirements - don't guess
- Use contextFiles from CLI output, don't assume specific file names
- Always check `openspec/product.md`, `openspec/architecture.md`, and `openspec/standards.md`; read each one that exists before implementing, and report any that are missing
- Do not start implementation until existing global context files have been loaded and summarized as active constraints
- Use optional `openspec/changes/<name>/ui/` handoff context when present, but never require it
- When `openspec/changes/<name>/ui/design.pen` exists, implement UI from that Pencil MCP design source, not from `interaction-spec.md` or screenshot approximations
- UI design content is reference-only for layout, hierarchy, states, and density; do not copy illustrative mock data, placeholder records, fake metrics, or sample business content into implementation unless explicitly required by OpenSpec or existing fixture conventions
- Prefer real app data contracts, existing fixtures, existing mock layers, or backend/API schemas over data shown in the design
- Treat `.pen` files as Pencil-only design sources; do not access them with normal filesystem readers
- Run relevant tests after implementation and report results
- Verify each spec `#### Scenario:` after implementation or explicitly list why it remains manual/unverified
- Check `openspec/standards.md` before claiming completion and report standards alignment

**Fluid Workflow Integration**

This skill supports the "actions on a change" model:

- **Can be invoked anytime**: Before all artifacts are done (if tasks exist), after partial implementation, interleaved with other actions
- **Allows artifact updates**: If implementation reveals design issues, suggest updating artifacts - not phase-locked, work fluidly
