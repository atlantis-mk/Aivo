# AI Work Packages

Direct change is the default, including ordinary same-task behavior and specification changes. Create Work only for unfinished cross-task state, open decisions requiring approval, controlled security/data/contract/migration/dependency/platform/release boundaries, irreversible coordination, or severe unclear bugs. The exact threshold is in `../docs/09-document-governance.md`.

Common IDs:

- Feature or behavior: `CHG-2026-002-versioned-session-api`
- Implementation deviation: `BUG-2026-001-cancelled-terminal-keeps-running`
- Other durable work uses the same stable pattern and `type: security|dependency|migration|technical_debt|governance`.

Create schema-v2 Work with `pnpm work:new -- <WORK-ID> --title "..." --type <type> --goal "..."`. It is one small `change.yaml`. Start it with `pnpm work:start -- <WORK-ID>` and run `pnpm docs:trace` after Requirement or routing changes.

Finish schema-v2 Work with `pnpm work:finish -- <WORK-ID>`. It runs applicable checks, marks Done, refreshes Traceability, and validates without copying command evidence or creating a hash archive. Legacy Work retains its historical completion and archive path; existing sealed files remain permanently read-only.

Full rules are in `../docs/09-document-governance.md`.
