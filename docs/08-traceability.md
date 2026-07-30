# Aivo requirement traceability

This table maps current primary Requirements to their owning specs and verification. Work status remains owned by each `change.yaml` and is not duplicated here.

| Requirement | Primary spec/context | ADR | Test ID | Evidence location | Status |
| --- | --- | --- | --- | --- | --- |
| REQ-PROVIDER-001 | `03-functional-requirements.md#REQ-PROVIDER-001 Provider configuration and health`, `provider-backend.md` | - | AT-PROVIDER-001 | `core/app/provider_*_test.go`, provider smoke | Implemented baseline |
| REQ-PROJECT-001 | `03-functional-requirements.md#REQ-PROJECT-001 Local project context`, `runtime-configuration.md` | - | AT-PROJECT-001 | `core/app/service_projects.go`, project/config tests | Implemented baseline |
| REQ-SESSION-001 | `03-functional-requirements.md#REQ-SESSION-001 Conversation and task execution` | - | AT-SESSION-001 | `core/app/session_*_test.go`, `service_session_submit.go` | Implemented baseline; v2 redesign pending |
| REQ-TOOL-001 | `03-functional-requirements.md#REQ-TOOL-001 Controlled local tool execution` | - | AT-TOOL-001 | `core/app/tool_runtime_*_test.go`, permission and terminal tests | Implemented baseline |
| REQ-EXTENSION-001 | `03-functional-requirements.md#REQ-EXTENSION-001 Skills, plugins, and MCP` | - | AT-EXTENSION-001 | `core/app/skill_*_test.go`, `plugin_*_test.go`, `mcp_*_test.go` | Implemented baseline; IA pending |
| REQ-WORKTREE-001 | `03-functional-requirements.md#REQ-WORKTREE-001 Worktree and parallel work lifecycle` | - | AT-WORKTREE-001 | `core/app/worktree_service_test.go`, `agent_parallel_test.go` | Implemented baseline; defaults pending |
| NFR-SECURITY-001 | `06-security-privacy.md` | - | CT-SECURITY-001 | Credential/auth tests and release secret review | Active |
| NFR-RELIABILITY-001 | `04-architecture.md`, `05-data-model.md` | - | CT-RELIABILITY-001 | Runtime/process cancellation tests; migration evidence per Work | Active |
| NFR-UI-001 | `03-functional-requirements.md#NFR-UI-001 Responsive and complete states` | - | AT-UI-001 | lint/build, manual acceptance, Work screenshots | Active |
| NFR-OBSERVABILITY-001 | `04-architecture.md#Observability`, `06-security-privacy.md#Logging and diagnostics` | - | CT-OBSERVABILITY-001 | focused logging/redaction tests per changed operation | Active |
