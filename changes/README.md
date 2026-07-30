# AI Work Packages

Use `../AGENTS.md` section 1.2 to decide whether Work is required. The decision depends on durable decisions, contracts, risk, migration, reversibility, and whether verification can finish now—not on whether a change is UI, bug, or refactoring.

Common IDs:

- Feature or behavior: `CHG-2026-002-versioned-session-api`
- Implementation deviation: `BUG-2026-001-cancelled-terminal-keeps-running`
- Other durable work uses the same stable pattern and `type: security|dependency|migration|technical_debt|governance`.

Copy `_template/` and fill both files. Low-risk Work may keep each body section to one short paragraph or explicit N/A. Accepted behavior is merged into primary specs and Traceability before implementation; Work does not replace current specifications.

After `Verified` or `Rejected`, run `pnpm work:archive -- <WORK-ID>`. Historical `Released` Work is also completed and must be sealed. Once sealed, the directory and existing manifest entry are permanently read-only; corrections require a new Work. Releases reference only sealed Work.

Full rules are in `../docs/09-document-governance.md`.
