# Data migration and rollback plan

## Current data boundary

The default store is SQLite at `~/.aivo/aivo.db`. The archived core creates a
schema version row with version `1` and currently combines GORM `AutoMigrate`
with targeted legacy migrations.

The database and credentials can contain sensitive user information. They are
not copied into the repository or committed as part of source preparation.

## Mandatory pre-migration behavior

Before v2 performs its first write to a v1 database, it must:

1. Close active database writers or acquire an exclusive migration lock.
2. Verify SQLite integrity and record the detected schema/application version.
3. Create a timestamped local backup using SQLite's safe backup mechanism.
4. Verify the backup can be opened and has the expected core record counts.
5. Run migration in a transaction where SQLite permits it.
6. Record success only after post-migration invariants pass.
7. Preserve the backup path for an explicit user-visible rollback action.

File copying an actively written SQLite database is not an acceptable backup.

## Migration requirements

- Migrations are monotonic, idempotent where practical, and never silently
  discard unsupported data.
- Destructive changes use expand/migrate/contract phases across releases.
- Provider secrets stay in the existing secure storage boundary and never enter
  fixtures, logs, or diagnostics.
- Project paths, session/event ordering, tool-call relationships, permission
  state, worktrees, plugins, MCP servers, and skills receive explicit invariants.
- Large session histories migrate in bounded batches with progress reporting and
  restart behavior.

## Test matrix

At minimum, migration tests cover:

- an empty v1 database;
- a populated database containing every v1 entity type;
- long session history and interrupted agent execution;
- provider records with and without authentication metadata;
- missing project paths and stale worktrees;
- migration cancellation or simulated failure;
- reopening the migrated database;
- restoration from the pre-migration backup.

Use sanitized fixtures derived from schema shapes, never a developer's live
database.

## Rollback boundary

Source rollback uses tag `aivo-v1-archive-2026-07-31`. Data rollback restores
the verified pre-migration backup only while no Aivo process is using the live
database. If v2 creates data that v1 cannot represent, the UI must warn that
rolling back loses post-migration changes and should offer an export first.
