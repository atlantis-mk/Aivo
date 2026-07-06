## ADDED Requirements

### Requirement: Initialize SQLite metadata store
Aivo SHALL initialize a local SQLite metadata database for MVP application state.

#### Scenario: App starts without database
- **WHEN** Aivo starts and the metadata database does not exist
- **THEN** Aivo creates the database, applies initial migrations, and records the current schema version

#### Scenario: App starts with current database
- **WHEN** Aivo starts and the metadata database already has the current schema version
- **THEN** Aivo opens the database without reapplying completed migrations

### Requirement: Version database migrations
Aivo SHALL manage schema changes through explicit versioned migrations.

#### Scenario: Pending migrations exist
- **WHEN** the metadata database schema version is older than the application migration set
- **THEN** Aivo applies pending migrations in order before using persistence-dependent features

#### Scenario: Migration fails
- **WHEN** a migration fails
- **THEN** Aivo reports a startup or persistence error and does not silently continue with a partially migrated schema

### Requirement: Persist MVP metadata entities
Aivo SHALL persist metadata for setup state, provider configurations, projects, tasks, task steps or tool runs, confirmations, artifacts, logs, verification results, and resumable workflow state.

#### Scenario: Task metadata is saved
- **WHEN** a task is created, updated, completed, failed, canceled, or resumed
- **THEN** Aivo persists the task status and relevant timestamps in SQLite

#### Scenario: Artifact metadata is saved
- **WHEN** an artifact is produced for a task
- **THEN** Aivo persists artifact ID, task ID, artifact type, filesystem reference, and creation timestamp in SQLite

#### Scenario: Provider metadata is saved
- **WHEN** a provider configuration is created or updated
- **THEN** Aivo persists non-secret provider metadata and setup readiness state in SQLite

### Requirement: Keep large content on filesystem
Aivo SHALL store source files and large generated artifacts on the filesystem and persist references in SQLite.

#### Scenario: Generated file artifact is stored
- **WHEN** a task creates a generated file artifact
- **THEN** the file content is written to the filesystem and SQLite stores metadata and a reference to that file

#### Scenario: Project source remains external
- **WHEN** Aivo indexes or references a selected project
- **THEN** it does not copy the full project source tree into SQLite

### Requirement: Avoid sensitive data persistence
Aivo SHALL avoid storing secrets, credentials, raw sensitive file contents, or unnecessary personal data in SQLite logs, artifacts, snapshots, or config.

#### Scenario: Credential use is recorded
- **WHEN** a task uses a credential reference
- **THEN** Aivo persists only the credential reference or redacted label and does not persist the credential value

#### Scenario: Log contains sensitive value
- **WHEN** a log entry candidate contains a known secret value from an approved credential source
- **THEN** Aivo redacts the sensitive value before persisting or displaying the log entry

### Requirement: Support task recovery after restart
Aivo SHALL use persisted metadata to reconstruct project and task review state after application restart.

#### Scenario: Restart after completed task
- **WHEN** the app restarts after a task completed
- **THEN** Aivo can show the task history, logs, artifacts, and final status from persisted metadata

#### Scenario: Restart after waiting confirmation
- **WHEN** the app restarts while a task was waiting for confirmation
- **THEN** Aivo can show the task and pending confirmation as waiting for user action unless the confirmation was invalidated
