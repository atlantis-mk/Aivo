# 修复跨平台打包暴露的进程生命周期缺陷

## Problem or goal

Public 发布的原生 CI 暴露了两个既有可靠性缺陷。Linux 上，快速连续的 PTY 控制请求可能在第一次输入写入完成前到达；写入完成逻辑随后无条件清除当前请求，错误删除了第二个请求并以 `output_idle` 返回。Windows 上，`sandbox_process.go` 依赖运行时 `GOOS` 分支隔离 Unix `syscall` 字段和函数，但 Go 仍会在编译 Windows 目标时解析这些符号，导致 `Setpgid` 与 `syscall.Kill` 编译失败。

## Expected behavior

- `NFR-RELIABILITY-001`: 一次 PTY 写入只能清除它实际解析的输入请求，不能清除已并发到达的后继请求。
- `NFR-RELIABILITY-001`: Unix 使用独立进程组终止子进程树；Windows 使用该平台可编译的进程终止实现。
- Linux 重复 PTY 测试、Windows Core 交叉编译和目标平台打包均通过。

## Non-goals

不改变输入授权模式、公开 API/RPC/IPC、持久化 schema、Renderer 行为、命令沙箱策略或 Windows Job Object 设计；不扩大任何进程权限。

## Impact

仅影响 Go application 层的 PTY 请求清理和 sandbox 进程终止平台实现。Electron/renderer、transport、domain、persistence、Provider、MCP/LSP、用户数据、依赖和格式均不变。发布门禁会重新验证 Windows 编译与 Linux 生命周期测试。

## Implementation constraints

- 写入前捕获的请求 ID 是唯一可在该次写入后清除的请求；不同 ID 的后继请求必须保留。
- 平台差异必须用 Go build tags 隔离，不能依赖不可编译符号周围的运行时条件。
- nil/已退出进程保持幂等，Unix 的进程组 `SIGTERM`/`SIGKILL` 与 `ESRCH` 处理保持不变。

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `PTY-RACE-001` | `NFR-RELIABILITY-001` | 后继输入请求不被前一次写入清除 | `CT-RELIABILITY-001` | In Progress |
| `PROCESS-OS-001` | `NFR-RELIABILITY-001` | Unix/Windows 进程终止实现按 build tag 编译 | `CT-RELIABILITY-001` | In Progress |

## Acceptance and evidence

- 精确单元回归证明旧请求的完成不能清除不同 ID 的新请求。
- PTY 交互测试在 Linux CI 和本机重复运行通过。
- `GOOS=windows GOARCH=amd64 go test ./app -run '^$'` 或等价 Windows 原生构建通过。
- `pnpm test:core`、`pnpm docs:check`、`pnpm lint`、`pnpm build` 与原生发布质量 CI 通过。

## Security and data lifecycle

终端内容仍只存在于现有有界 session buffer；不新增日志、持久化、网络或凭据流。进程终止仍由现有 session/sandbox owner 发起，后继输入请求继续受原有 lease 决策保护。

## Compatibility and migration

无 schema、API/RPC/IPC、设置或用户数据迁移。回滚会恢复 Windows 编译失败和 PTY 后继请求偶发丢失，但不修改持久化数据。

## Bug root cause

PTY 写入完成代码清理的是当时的 `s.inputRequest`，而不是写入开始时捕获的请求；控制管道和 PTY 输出由独立 goroutine 处理，因此后继请求可以在清理前合法到达。Windows 缺陷来自误认为运行时 `GOOS` 分支能够代替编译期平台隔离。现有测试主要在 macOS 本机运行，且 PTY 时序通常没有触发该窗口；Windows Core 测试也未在原生打包前单独编译。受影响版本为 `0.0.0-development`，修复版本为下一个开发构建。
