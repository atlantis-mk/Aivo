# Aivo 对比 opencode 代码开发能力验收表

范围：只评估桌面端代码开发工作流。CLI、TUI、SDK、GitHub Action、云端协作、企业管理、非代码生产力工作流不在本表范围内。

使用方式：把这张表作为手动替换验收 checklist。只有在 Aivo 桌面端真实测试过对应能力，并满足通过标准后，才勾选 `Aivo 是否匹配 opencode`。证据栏应记录 session ID、项目路径、provider、命令输出、变更文件、截图或日志等可复核信息。

| 领域 | 能力项 | opencode 基线行为 | Aivo 需要匹配的行为 | 手动测试覆盖 | 通过标准 | Aivo 是否匹配 opencode | 证据 / 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 项目接入 | 打开已有 Git 项目 | 能在选定仓库内工作，并理解项目文件 | 用户可以选择项目，看到仓库上下文，并启动绑定到该 workspace 的代码会话 | 打开 Aivo 仓库和一个外部 fixture 仓库 | workspace root 正确、文件可读、路径不混乱 | [ ] | |
| 项目接入 | 最近项目 | 之前使用过的项目可以快速重新打开 | 最近项目列表可持久化，并可重新进入项目 | 打开两个项目，退出并重启 app，再从最近项目进入 | 最近项目保留路径和状态 | [ ] | |
| 项目接入 | Git 状态感知 | 能展示或推理当前变更文件和分支状态 | Workbench 展示 Git 元信息、变更文件、非 Git 状态、不可访问路径状态 | 分别测试 Git 仓库、非 Git 文件夹、已删除或不可访问文件夹 | 状态准确且用户知道下一步该怎么做 | [ ] | |
| 仓库读取 | 文件树和定向读取 | 能检查相关文件，而不是盲目加载整个仓库 | Agent 能通过有边界的工具列出、搜索、读取文件 | 要求解释某个模块，并只使用仓库上下文 | 能引用正确文件，避免无关读取 | [ ] | |
| 仓库读取 | 快速文本搜索 | 能搜索 symbol、函数、字符串和错误信息 | 搜索工具能在 workspace 内找到匹配代码 | 搜索已知函数、路由、配置 key、错误字符串 | 结果包含正确路径和片段 | [ ] | |
| 仓库读取 | 大输出处理 | 能处理长文件、搜索结果、命令输出而不丢失任务状态 | Timeline 保留完整输出，模型上下文使用有边界摘要 | 读取或搜索 noisy 目录，或运行大输出命令 | UI 可访问完整输出；agent 摘要仍然有用 | [ ] | |
| 规划 | 自然语言创建任务 | 能把用户请求转成实现计划 | Workbench 任务输入生成结构化计划 | 提交小 bug 修复和多文件 feature 请求 | 计划说明文件、风险、命令、验证步骤 | [ ] | |
| 规划 | 副作用前审批 | 在要求审批时，不会先编辑文件或运行高风险命令 | 用户先审阅并批准计划，再执行文件写入或高风险 shell 行为 | 拒绝一次计划，再批准一次计划 | 拒绝不会产生变更；批准后继续执行 | [ ] | |
| 规划 | 执行中用户转向 | 用户能在任务执行中纠正方向 | `delivery=steer` 在下一个安全 provider-turn 边界生效 | 启动任务，在工具运行时发送纠正指令 | Agent 采纳转向且状态不损坏 | [ ] | |
| 规划 | 排队后续输入 | 用户能在任务忙碌时排队后续请求 | `delivery=queue` 等当前 continuation 空闲后再执行 | 长命令运行期间提交第二个请求 | 第二个请求在当前任务到达安全点后运行 | [ ] | |
| 编辑 | 应用 patch | 能精确应用 patch 并报告失败 | `apply_patch` 支持 preflight、stale 检查、partial failure 报告 | 应用一个有效 patch 和一个故意 stale 的 patch | 有效 patch 成功；stale patch 被清晰拒绝 | [ ] | |
| 编辑 | 直接编辑文件 | 能安全修改已有文件 | `edit_file` 报告目标、旧/新状态、stale hash 冲突和结果 | 编辑已知函数，并强制制造 stale 编辑冲突 | 正确编辑成功；stale 编辑不会覆盖新内容 | [ ] | |
| 编辑 | 新建文件 | 能在权限检查后创建文件 | `write_file` 只在策略和权限允许后创建文件 | 新增一个小测试 fixture 或文档文件 | 文件内容正确，并记录在 timeline 中 | [ ] | |
| 编辑 | 外部路径保护 | 不会静默写入允许范围之外的路径 | 文件工具检测外部路径，并要求审批或拒绝 | 尝试写入 selected workspace 外部路径 | 触发正确审批或拒绝路径 | [ ] | |
| 编辑 | turn 级 diff | 用户能检查某一轮产生的所有变更 | Workbench 展示写入、编辑、patch 造成的 changed files 和 diff | 完成一个小 bug 修复 | diff 完整、可读，且和文件系统变更一致 | [ ] | |
| 编辑 | 回滚或恢复 | 用户能从某一轮文件变更中恢复 | turn 级 restore/revert 在支持场景可用，不支持时清晰报告 | 做一个受控变更，然后执行 revert | 文件恢复到之前内容，或不支持原因明确 | [ ] | |
| Shell | 在 workspace 运行命令 | 能在正确 cwd 下运行项目命令 | `bash` 使用被追踪的 cwd 和 command policy | 运行 `pwd`、包管理器命令、项目测试命令 | cwd 正确且命令输出被保留 | [ ] | |
| Shell | 运行测试 | 能运行测试并总结失败 | `run_tests` 统一测试命令执行、失败和输出 | 运行 passing 和 failing 测试 fixture | 失败摘要指向可操作错误 | [ ] | |
| Shell | 超时处理 | 长命令不会永久挂住会话 | shell/test 工具强制 timeout 并报告状态 | 运行受控 sleep/timeout 命令 | timeout 可见，会话仍可继续使用 | [ ] | |
| Shell | 后台进程状态 | 能处理 dev server 或长时间运行命令 | 后台命令暴露状态和保留输出 | 启动 dev server，查看状态，再停止或交给系统管理 | UI 展示运行状态和最新输出 | [ ] | |
| Shell | stdin 处理 | 需要 stdin 的命令被显式处理或 gated | stdin 使用体现在 permission metadata 中 | 运行一个请求输入的命令 | 用户能看到并控制 stdin 行为；不会隐藏式卡死 | [ ] | |
| Shell | 环境变量处理 | 敏感 env 使用受控 | env keys 体现在 permission metadata 中，secret 被脱敏 | 使用 fake secret env key 运行命令 | secret 值不会被日志明文记录；权限元数据安全展示 key | [ ] | |
| Shell | 网络命令策略 | 需要网络时按策略审批 | network metadata 按 scope 触发审批或拒绝 | 运行受控网络命令 | 网络访问按策略被批准或拒绝 | [ ] | |
| 代码智能 | Diagnostics | 能读取编译、lint、类型诊断 | `lsp_diagnostics` 在 LSP 可用时返回结构化 diagnostics | 打开含已知 diagnostics 的 Go 和 TypeScript fixture | diagnostic path、range、severity、message 正确 | [ ] | |
| 代码智能 | Symbols | 能跨代码库查找 symbol | `lsp_symbol_search` 优先使用 LSP，必要时扫描 fallback | 搜索函数、类型、route component、Go method | 返回正确位置；如走 fallback，状态明确 | [ ] | |
| 代码智能 | 跳转定义 | 能从引用定位定义 | `lsp_definition` 返回定义位置 | 测试 Go method 调用和 TS import/component 引用 | 定义指向正确文件和范围 | [ ] | |
| 代码智能 | 查找引用 | 能查找 symbol 使用处 | `lsp_references` 返回引用位置 | 查找 Go function 和 TS component/util 的引用 | 结果准确到足以支持编辑决策 | [ ] | |
| 代码智能 | LSP 不可用 fallback | language server 不可用时不阻塞代码任务 | 工具报告 unavailable/fallback 状态并继续任务 | 在 fixture 中禁用或不安装 language server | Agent 继续使用扫描 fallback，或清晰说明限制 | [ ] | |
| 调试 | 只读诊断 | 能在编辑前调查失败行为 | Agent 可以 run/read/search，但不改文件 | 给一个 failing test，要求只诊断不修复 | 没有文件变更；诊断指出可能根因和证据 | [ ] | |
| 调试 | 诊断后修复 | 能在诊断后经批准进入实现 | Agent 提出修复，必要时等待批准，再编辑并验证 | 只读诊断后批准修复 | 修复范围收敛且通过验证 | [ ] | |
| 验证 | lint/build/test 循环 | 能在变更后运行验证 | Agent 运行合适命令并报告结果 | 完成小任务和中等任务 | 最终状态包含命令和 pass/fail 摘要 | [ ] | |
| 验证 | 失败处理 | 能解释并修复失败命令 | Agent 总结失败并更新计划 | 注入 failing test/build error | 下一步相关，不重复无效命令 | [ ] | |
| Diff 审查 | 审查变更文件 | 用户能在完成前检查结果 | Workbench timeline 展示文件变更和 diff 展开 | 完成多文件 feature | 用户能检查每个 changed file 和完整 diff | [ ] | |
| Diff 审查 | 完整命令输出保留 | 用户能展开保留的命令输出 | Timeline 保留模型摘要之外的 stdout/stderr | 运行 verbose test 或 build 命令 | 命令结束后仍可查看完整输出 | [ ] | |
| 会话连续性 | 中断执行 | 用户能停止活动任务 | `InterruptSessionExecution` 将当前执行标记为 interrupted | 在长命令或工具序列中中断 | 会话干净停止，状态可见 | [ ] | |
| 会话连续性 | 恢复执行 | 用户能在中断后继续 | `ResumeSessionExecution` 从 durable state 安全恢复 | 恢复被中断会话 | Agent 知道 last command、变更、todos、下一步 | [ ] | |
| 会话连续性 | app 重启恢复 | 重启后不会静默重放运行中的 tool call | 启动时将未完成 tool call 标记为 interrupted | 活动或模拟 running tool call 期间重启 app | 有副作用的 tool 不被重放；用户看到恢复状态 | [ ] | |
| 会话连续性 | 上下文压缩 | 长会话仍可继续使用 | `CompactSessionContext` 生成 checkpoint/summary/recent-event context | 跑一个长多步骤任务并压缩 | 重要决策、diff、命令、todos 在压缩后保留 | [ ] | |
| 会话连续性 | event cursor 列表 | UI 能增量获取事件 | `ListSessionEventsAfterCursor` 返回有序事件切片 | 生成多个事件后按 cursor 获取 | 没有丢失、重复、乱序事件 | [ ] | |
| 权限 | 文件写入审批 | 文件写入按 scope 请求审批 | 权限弹窗包含 action、tool、paths、source | ask mode 下尝试 edit/write/patch | prompt 精确，决策被执行 | [ ] | |
| 权限 | Shell/test 审批 | Shell 命令按 scope 请求审批 | prompt 包含 command key、cwd、risk、stdin/env/network metadata | 运行安全命令和高风险命令 | 安全/高风险决策符合配置策略 | [ ] | |
| 权限 | 已保存审批 | 记住的审批精确匹配，不泛化过度 | saved approval 按 workspace、session、tool、action、path/command key 匹配 | 批准一个路径/命令，再尝试相似但不同的路径/命令 | 只有精确批准的动作跳过 prompt | [ ] | |
| 权限 | 拒绝行为 | 被拒绝的动作不会执行 | 工具返回标准化 denial 结果，会话仍可转向 | 拒绝一次写入和一次高风险 shell 命令 | 文件/命令无变更；agent 请求替代路径 | [ ] | |
| 权限 | 外部 roots | 外部目录访问受保护 | externalRoots metadata 触发审批或拒绝 | 在受控 fixture 中读/写 workspace 外部路径 | 外部访问明确且符合策略 | [ ] | |
| 权限 | Plugin/MCP 权限 | 外部工具进入同一权限模型 | Plugin/MCP tool 请求包含 source metadata | 运行需要审批的 fixture plugin 和 MCP tool | prompt 准确展示 source/sourceID/tool/action | [ ] | |
| Plugin runtime | Plugin tool 注册 | Plugin 工具进入统一 tool catalog | Tool catalog entry 包含 source、sourceID、registrationID、riskLevel、toolsets | 启用 fixture plugin | 工具可发现且 metadata 完整 | [ ] | |
| Plugin runtime | Plugin 执行 | Plugin 工具能参与代码任务 | Agent 可调用 fixture plugin，并在会话中使用结果 | 要求 agent 在代码任务中使用 fixture plugin | tool result 出现在 timeline，并影响任务 | [ ] | |
| Plugin runtime | stale plugin 拒绝 | registration 变化后旧 tool call 被拒绝 | stale registrationID 调用安全失败 | 测试中修改或 reload plugin registration | stale call 被清晰拒绝 | [ ] | |
| MCP runtime | MCP stdio/http 工具 | MCP 工具可注册并执行 | MCP stdio/http 工具进入同一 catalog 和 timeline | 运行 fixture stdio MCP 和 HTTP MCP 工具 | 工具执行成功，或以标准化 diagnostics 失败 | [ ] | |
| MCP runtime | MCP auth/OAuth 错误 | 鉴权失败可见且可恢复 | auth/probe/execution 错误出现在 diagnostics/settings 中 | 运行 fixture auth failure 场景 | 错误可操作，且不破坏会话 | [ ] | |
| MCP runtime | MCP prompts/resources | prompts/resources 只通过用户显式动作插入 | UI/service 将选定 MCP resource 插入 session context | 插入 fixture prompt/resource | 上下文插入可见，且不会自动注入 | [ ] | |
| Workbench UI | Timeline 完整性 | 用户能看到 agent 做了什么 | Timeline 展示消息、tool calls、权限、命令、diff、diagnostics、验证结果 | 完成一个中等任务 | Timeline 足以审计会话 | [ ] | |
| Workbench UI | 恢复摘要 | 用户重新进入会话时能看到有用状态 | Resume 页展示 last command、open todos、changed files、latest checkpoint、next suggested action | 恢复一个历史会话 | 摘要和实际先前工作一致 | [ ] | |
| Workbench UI | 错误和空状态 | UI 能优雅处理缺失数据 | 非 Git、不可访问、无 provider、无 project、tool 失败状态明确 | 覆盖每种 blocked state | 用户知道发生了什么以及下一步怎么做 | [ ] | |
| Provider 处理 | Provider/model 选择 | 用户能为代码任务选择可用模型/provider | Session 存储 agentMode、modelRef、toolsets、permissionScope | 如有两个 provider，分别用同一小任务测试 | provider 选择可见，任务使用所选 provider | [ ] | |
| Provider 处理 | Tool continuation 边界 | tool result 在下一次模型调用前已持久化 | tool result 先持久化，再继续 provider continuation | 在 tool result 后强制重启或制造失败 | 会话能恢复，且不丢失已完成 tool result | [ ] | |
| 安全 | Secret 脱敏 | secret 不暴露在日志或 UI 输出中 | 已知 secret-like 值在命令/tool 输出中被脱敏 | 在 env/output fixture 中使用 fake token/API key | secret 值不会明文展示或持久化 | [ ] | |
| 安全 | 危险 shell 拒绝 | 危险命令被阻止或要求强审批 | command policy 捕获 destructive 或 broad-risk 命令 | 在 fixture/temp dir 中尝试受控危险命令 | prompt/rejection 正确，且无非预期破坏 | [ ] | |
| 端到端 | 小 bug 修复 | 能执行聚焦的单文件代码修复 | 规划、编辑、测试、diff 审查、最终总结都工作正常 | 在 fixture 项目修复已知单文件 bug | 变更正确且通过验证 | [ ] | |
| 端到端 | 多文件 feature | 能修改 3-5 个文件完成 feature | 计划审批、patch review、测试和 timeline 保持连贯 | 实现一个触及 3-5 个文件的小 feature | 没有漏改；验证通过 | [ ] | |
| 端到端 | 测试失败调试 | 能诊断并修复 failing tests | Agent 先调查失败，经批准后编辑，再验证 | 提供 failing test fixture | scoped fix 后测试通过 | [ ] | |
| 端到端 | 中断恢复 | 能在任务中断或重启后恢复 | interrupt/resume/restart flow 保留 durable state | 活动任务中断，重启 app，再恢复 | 无静默重放；用户能继续完成任务 | [ ] | |
| 端到端 | Plugin/MCP 辅助任务 | 外部工具能参与真实代码工作 | fixture plugin/MCP tool 作为任务执行的一部分被使用 | 要求代码任务依赖 fixture 外部工具输出 | tool result 影响最终代码或决策 | [ ] | |
| 端到端 | LSP 辅助任务 | LSP 实质性改善代码导航 | Agent 在任务中使用 diagnostics/definition/references | 要求跨 Go/TS 代码查引用再修改 | references/definitions 指导正确编辑 | [ ] | |

## 替换就绪判定规则

只有满足以下条件，才可以把 Aivo 标记为已替换 opencode 的桌面端代码开发工作流：

1. 上表所有必需行都已打钩，或在 `证据 / 备注` 中明确记录并接受限制。
2. 端到端行必须来自真实桌面端会话，不能只依赖自动化测试。
3. 测试后 `pnpm test:core`、`pnpm lint`、`pnpm build` 都通过。
4. 现有 `acceptance-matrix.md` 从 `Status: incomplete` 更新为通过状态，并附带手动验收证据。
