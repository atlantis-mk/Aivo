# Aivo v2 redesign

This directory is the preparation package for the Aivo v2 redesign. It defines
the protected starting point, current capabilities, architectural guardrails,
data migration rules, decisions still needed, and an executable delivery plan.

## Documents

- `baseline.md`: immutable v1 archive reference and verification results.
- `proposal.md`: motivation, goals, scope, and success criteria.
- `capability-inventory.md`: current capability disposition for v2 planning.
- `architecture.md`: target boundaries and compatibility rules.
- `data-migration.md`: user-data protection, migration, and rollback strategy.
- `decision-log.md`: product decisions that must be made before implementation.
- `tasks.md`: staged delivery checklist and quality gates.

## Working rule

The v1 archive is read-only. New implementation work happens on
`codex/aivo-v2` and should be delivered as vertical slices. A slice is complete
only when its domain model, persistence, transport contract, desktop behavior,
error states, tests, and migration impact have all been addressed.
