# Aivo requirement traceability

This table maps current primary Requirements to their owning specs and verification. Work status remains owned by each `change.yaml` and is not duplicated here.

| Requirement | Primary spec/context | ADR | Test ID | Evidence location | Status |
| --- | --- | --- | --- | --- | --- |
| REQ-PROVIDER-001 | `03-functional-requirements.md#REQ-PROVIDER-001 Provider configuration and health`, `provider-backend.md` | - | AT-PROVIDER-001 | `core/app/provider_*_test.go`, unchanged canonical tool-name declaration/history/response/stream tests, provider smoke | Implemented baseline |
| REQ-PROJECT-001 | `03-functional-requirements.md#REQ-PROJECT-001 Local project context`, `runtime-configuration.md` | - | AT-PROJECT-001 | `core/app/service_projects.go`, project/config tests | Implemented baseline |
| REQ-PROJECT-002 | `03-functional-requirements.md#REQ-PROJECT-002 Initial workspace for unscoped conversations`, `05-data-model.md` | ADR-0001 | AT-PROJECT-002 | initialization/session workspace tests, migration tests, responsive setup acceptance | Active |
| REQ-PROJECT-003 | `03-functional-requirements.md#REQ-PROJECT-003 Agent project catalog and immutable session binding`, `04-architecture.md#Agent tools and extensions`, `05-data-model.md` | ADR-0002 | AT-PROJECT-003 | project store/service/extension/permission tests and desktop permission/session-state acceptance | Active |
| REQ-SESSION-001 | `03-functional-requirements.md#REQ-SESSION-001 Conversation and task execution` | - | AT-SESSION-001 | session runtime tests, renderer session-scoped/one-shot tool activation tests, cancelled-turn context/interaction isolation tests, `service_session_submit.go` | Implemented baseline; v2 redesign pending |
| REQ-WORKSPACE-001 | `03-functional-requirements.md#REQ-WORKSPACE-001 Sidebarless chat workspace` | - | AT-WORKSPACE-001 | lint/build, responsive screenshots, canvas-only renderer acceptance | Active |
| REQ-TOOL-001 | `03-functional-requirements.md#REQ-TOOL-001 Controlled local tool execution` | ADR-0002 | AT-TOOL-001 | four-primitive registry/schema path guidance, execution-environment, atomic mutation, Bash artifact, permission cancellation, and historical-rendering tests | Active |
| REQ-TOOL-002 | `03-functional-requirements.md#REQ-TOOL-002 Contextual tool activity inspector` | - | AT-TOOL-002 | lint/build, responsive screenshots, tool-inspector interaction acceptance | Active |
| REQ-EXTENSION-001 | `03-functional-requirements.md#REQ-EXTENSION-001 Skills, plugins, and MCP` | ADR-0002 | AT-EXTENSION-001 | manifest/protocol global-name validation, Host pre-call Skill/plugin/Manifest/MCP catalog resolution, canonical Skill summary and validated instruction/context materialization, registration snapshot, session activation isolation, collision refusal, unchanged Provider identity, extension lifecycle, MCP upstream-name separation, and isolated-view tests | Active |
| REQ-WORKTREE-001 | `03-functional-requirements.md#REQ-WORKTREE-001 Worktree and parallel work lifecycle` | - | AT-WORKTREE-001 | `core/app/worktree_service_test.go`, `agent_parallel_test.go` | Implemented baseline; defaults pending |
| NFR-SECURITY-001 | `06-security-privacy.md` | ADR-0002 | CT-SECURITY-001 | extension trust/credential/Web isolation tests, accurate containment-state tests, and release secret review | Active |
| NFR-RELIABILITY-001 | `04-architecture.md`, `05-data-model.md` | ADR-0002 | CT-RELIABILITY-001 | cancelled-turn isolation plus primitive/extension cancellation, interaction teardown, backpressure, draining, restart, and artifact cleanup tests | Active |
| NFR-UI-001 | `03-functional-requirements.md#NFR-UI-001 Responsive and complete states` | - | AT-UI-001 | lint/build, manual acceptance, Work screenshots | Active |
| NFR-UI-002 | `03-functional-requirements.md#NFR-UI-002 Desktop visual scale and theme` | - | AT-UI-002 | theme-token inspection, lint/build, responsive Work screenshots | Active |
| NFR-OBSERVABILITY-001 | `04-architecture.md#Observability`, `06-security-privacy.md#Logging and diagnostics` | - | CT-OBSERVABILITY-001 | focused logging/redaction tests per changed operation | Active |
