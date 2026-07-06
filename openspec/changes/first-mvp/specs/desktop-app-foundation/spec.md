## ADDED Requirements

### Requirement: Runnable desktop application
Aivo SHALL provide a runnable Electron desktop application initialized in the repository with Go backend code, React + TypeScript frontend code, and documented commands for local development.

#### Scenario: Start local development app
- **WHEN** a developer runs the documented local development command after installing dependencies
- **THEN** the Electron desktop app launches and renders the Aivo workbench UI without requiring an external application server

#### Scenario: Development command is documented
- **WHEN** a developer opens the repository setup documentation
- **THEN** the documentation lists the install and local development commands required for the MVP app

### Requirement: Backend module boundaries
The Go backend SHALL separate core models and rules, use-case orchestration, and concrete side effects into `domain`, `app`, and `infra` boundaries.

#### Scenario: Business logic is outside Aivo bridge adapter
- **WHEN** an Aivo bridge method handles a project, task, confirmation, log, or artifact request
- **THEN** it validates boundary input, calls an app-layer use case, and returns a UI-safe response without embedding workflow business rules in the adapter

#### Scenario: Domain package is side-effect free
- **WHEN** domain package imports are inspected
- **THEN** the domain package does not import Electron, React, SQLite drivers, filesystem APIs, process execution APIs, network clients, or provider SDKs

### Requirement: Typed frontend service boundary
The frontend SHALL access backend capabilities through typed client or service modules rather than direct generated Aivo bridge calls from arbitrary React components.

#### Scenario: UI requests project list
- **WHEN** a React component needs project data
- **THEN** it obtains that data through a frontend project service or hook with typed request and response shapes

#### Scenario: Raw Aivo bridge calls are isolated
- **WHEN** frontend source is inspected
- **THEN** generated Aivo bridge handler imports are limited to the frontend client/service boundary

### Requirement: Workbench UI foundation
The frontend SHALL provide a welcome, initialization, and workbench-first interface built with React, TypeScript, and shadcn/ui conventions.

#### Scenario: First screen before setup is welcome
- **WHEN** the app starts and setup is incomplete
- **THEN** the first screen presents a functional welcome entry into initialization, not a marketing landing page

#### Scenario: First screen after setup is workbench
- **WHEN** the app starts after setup is complete
- **THEN** the first screen presents project selection or the last active workbench state

#### Scenario: Common workflow states are visible
- **WHEN** project or task data is loading, empty, failed, or available
- **THEN** the UI renders distinct loading, empty, error, or success states for that surface

### Requirement: Quality gates are configured
The MVP SHALL define and document relevant typecheck, test, and build commands once the project tooling is initialized.

#### Scenario: Commands exist after initialization
- **WHEN** the application source and manifests are initialized
- **THEN** the repository exposes documented commands for Go checks, frontend type checks, automated tests, and desktop build or development verification

#### Scenario: Unavailable checks are reported
- **WHEN** a required quality gate cannot run because tooling is not configured or a dependency is unavailable
- **THEN** the implementation notes state which command was not run and why
