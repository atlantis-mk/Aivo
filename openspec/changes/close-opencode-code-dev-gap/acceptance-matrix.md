## Desktop Code-Development Replacement Acceptance Matrix

Readiness is scoped to desktop code-development workflows only. CLI, TUI, SDK, GitHub Action, enterprise collaboration, and non-code workflows are excluded from scoring.

| Scenario | Project | Prompt | Provider | Agent mode | Permissions | Commands / checks | Files changed | Outcome | Known limitations |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Small fix | Aivo repository | Fix a focused backend/runtime issue | Automated Go tests | assistant | request approval / test fixtures | `pnpm test:core` | Backend runtime/test files | Passing automated coverage | Manual desktop run not recorded |
| Multi-file feature | Aivo repository | Add session execution control and LSP fallback tools | Automated Go tests | assistant | request approval / test fixtures | `pnpm test:core`, `pnpm build` | Domain, app, persistence, bridge service files | Passing automated coverage | Manual desktop run not recorded |
| Debug-to-build flow | Aivo repository | Diagnose and repair compile/test failures | Local toolchain | assistant | local shell | `pnpm test:core`, `pnpm lint`, `pnpm build` | Runtime and tests | Passing | Existing lint/build warnings remain non-blocking |
| Interrupted resume | Temporary runtime database | Recover running tool calls after restart | Automated Go tests | assistant | local persistence | `TestStartupRecoveryMarksRunningToolCallsInterrupted`, `TestExecutionControlInterruptCompactCursorAndQueuedInput` | Session runtime persistence | Passing | Provider resume continuation beyond durable state remains provider-dependent |
| MCP/plugin participation | Fixture plugin and MCP tests | Register, execute, diagnose, and recover external tools | Automated Go tests | assistant | plugin/MCP fixture permissions | `pnpm test:core` | Existing plugin/MCP runtime files plus registry integration | Passing existing fixture coverage | OAuth behavior is covered by HTTP MCP tests, not a live third-party server |
| LSP lookup | Temporary Go and TypeScript workspaces | Find diagnostics, definition, references, and symbols | Fake stdio LSP plus fallback scan | assistant | safe read-only tools | `TestBoundedLSPManagerStartsFakeGoServer`, `TestBoundedLSPManagerStartsFakeTypeScriptServer`, `TestLSPFallbackToolsReturnStructuredResults` | Bounded LSP manager and LSP tools | Passing LSP startup and fallback coverage | Live third-party language server behavior remains environment-dependent |
| Safety denial/approval | Temporary workspace | Exercise write/shell permission gates | Automated Go tests | assistant/debug/plan modes | request approval / saved approval / full access | `pnpm test:core` | Permission engine and tool runtime fixtures | Passing existing coverage | Manual UI approval transcript not recorded |

## Latest Verification

- `pnpm test:core`: passed.
- `pnpm lint`: passed with existing Fast Refresh and hook-dependency warnings.
- `pnpm build`: passed with existing large-barrel-module and chunk-size warnings.

## Replacement Readiness

Status: incomplete.

Reason: automated coverage passes for the required backend/runtime scenarios, but manual desktop acceptance results have not been recorded for every matrix row. Aivo must keep replacement readiness false until each required scenario has a passing manual result or an explicitly accepted limitation.
