# Agent project extension contract

## Registration

The Host compiles and installs trusted built-in extension `aivo.projects` version `1.0.0`. Its three tools use Manifest v1, `activation: default`, the `coding` toolset supplied by the extension adapter, and frozen registration/schema identities. They are namespaced extension tools, not additional core primitives.

## Project summary and query

`ProjectSummary` contains `id`, `name`, `description`, `rootPath`, and `timeUpdated`. `aivo.projects.query` accepts either an exact `projectId` or the list/search fields `query`, `limit`, and `cursor`. The default limit is 20 and the maximum is 50. List/search excludes sidebar-hidden projects, orders by `time_updated DESC, id DESC`, and uses an opaque validated keyset cursor. The result contains `projects`, optional `nextCursor`, and the current session's associated project even when it is hidden.

## Registration

`aivo.projects.add` accepts one absolute `rootPath`. The Host cleans the path, requires an existing accessible directory, and registers it without creating or cloning content. Repetition returns the same project. Registering a hidden existing root restores it. The result reports `created`, `existing`, or `restored` and never associates the session implicitly.

## Immutable association

`aivo.projects.associate` accepts `projectId` and the exact `rootPath` returned by query/add. It derives the session from execution context. Only an unbound coding session whose coding context still equals the configured initial workspace may bind. A same-project retry succeeds with `changed: false`; a different existing association, detachment, or rebinding is forbidden. A session with an active interactive terminal or a specialized workspace/worktree fails before approval or mutation.

The application prepares the target coding context, then the persistence adapter conditionally updates an empty `sessions.project_id` and upserts `coding_contexts` in one transaction. If the conditional update loses a race, the adapter reloads the winner and returns an idempotent success only when it matches the requested project; otherwise it returns a conflict. After success, the Host publishes the full updated Session and rebuilds workspace-dependent context at the next model-call boundary.

## Permissions and failures

Query uses `projects.read`. Add and associate use `projects.write`; permission metadata identifies the operation and exact root with a non-filesystem resource key so remembered approval cannot grant project-wide authority. Request-approval asks, auto-approve/full-access allow, and read-only modes deny writes. Preflight validation rejects impossible associations before prompting.

Stable failure codes are `invalid_arguments`, `absolute_path_required`, `project_path_not_found`, `project_not_directory`, `project_not_found`, `project_reference_mismatch`, `session_required`, `coding_session_required`, `project_already_bound`, `workspace_specialized`, `workspace_busy`, `invalid_cursor`, `cancelled`, and `project_update_failed`.
