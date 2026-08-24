# 重建 Public 源码仓库并自动发布安装包

## Problem or goal

Aivo 当前 GitHub 仓库为 Private，仓库没有源码许可证，打包工作流只支持手动生成短期 Actions artifact。用户要求参照 VaultMesh 的治理方式，把 Aivo 作为 source-available、仅限非商业用途的 Public 项目重新创建，并让版本标签自动生成原生桌面安装包、发布到 Cloudflare R2 和 GitHub Release assets。

## Expected behavior

- 当前及未来由授权方拥有或有权许可的 Aivo 源码按 `PolyForm-Noncommercial-1.0.0` 提供；仓库必须明确这不是 OSI 认可的开源许可证，商业使用需要另行签署书面授权。
- 删除前必须在仓库外创建并验证完整 Git bundle、GitHub 元数据与远端引用备份，并对将公开的当前工作树进行秘密扫描。
- 删除目标必须精确为 `atlantis-mk/Aivo`；同名仓库重新创建为 Public，默认分支为仅包含当前已验证工作树的新根 `main`，不迁移旧 PR、Issue、Action、Release、普通分支或 tag。
- 新仓库必须恢复 description、topics、secret scanning、push protection 和 private vulnerability reporting。
- 推送匹配 `v<package-version>` 的 tag 后，GitHub Actions 必须在原生 macOS Apple Silicon、macOS Intel、Windows x64 和 Linux x64 runner 上验证并打包，将规范化资产和 SHA-256 清单先发布到不可变 R2 版本前缀，再创建或补全同 tag 的 GitHub Release assets。
- R2 同名对象只有摘要元数据完全一致时可以跳过；内容不同或缺少摘要元数据时必须拒绝覆盖。可变 `channels/stable/latest.json` 必须最后发布并回读验证。

## Non-goals

不把 PolyForm Noncommercial 描述为 OSI 开源，不授予商业权利，不迁移旧 GitHub 仓库身份、Actions 日志或隐藏引用，不删除外部 clone/缓存，不新增应用内自动更新协议，不改变 Electron/Core 运行时、数据格式、API/RPC/IPC、Provider、扩展或用户数据，也不把未签名安装包描述为正式签名发行版。

## Impact

影响根许可证、README/贡献和包元数据、Git 提交图、GitHub 仓库身份与安全设置、GitHub Actions、R2 对象和 GitHub Release assets。产品运行时、Renderer、Electron privilege boundary、Go domain/app/persistence/transport、schema、credentials ownership、MCP/LSP/process lifecycle 与用户数据均不变。目标 OS 打包仍受现有签名、notarization 和 smoke gates 约束。

## Implementation constraints

- 许可证正文必须与 PolyForm Noncommercial 1.0.0 标准文本完全一致；package manifests 保持 `private: true`，以防误发包，不改变源码许可。
- 新根历史必须从独立临时 index 生成，不能重置、覆盖或清理当前脏工作树；仓库外恢复 bundle 不得提交或上传。
- GitHub 删除前必须确认备份、秘密扫描、适用测试和工作树快照完成；当前 token 缺少 `delete_repo` scope 时必须暂停，不得绕过。
- R2 凭据仅由 GitHub Actions secrets `R2_ACCESS_KEY_ID` 与 `R2_SECRET_ACCESS_KEY` 提供；公开配置由 variables `R2_ACCOUNT_ID`、`R2_BUCKET`、`R2_PUBLIC_BASE_URL` 提供。日志不得输出凭据。
- 发布必须以原生目标 OS 构建；任何 matrix 构建、测试、上传或回读失败时不得创建完成的 GitHub Release。
- 重复运行必须能够复用内容完全相同的 R2/GitHub assets，并拒绝同版本不同内容。

## Tasks

| Task | Requirement | Verifiable output | Test | Status |
| --- | --- | --- | --- | --- |
| `PUBLIC-01` | `NFR-RELEASE-001` | PolyForm 许可证、source-available/商业授权边界和一致包元数据 | `CT-LICENSE-001` | Complete |
| `PUBLIC-02` | `NFR-RELEASE-001` | 版本标签驱动的四平台打包、R2 与 GitHub Release publication | `CT-RELEASE-001` | In Progress |
| `PUBLIC-03` | `NFR-RELEASE-001` | 仓库外备份、秘密扫描、新根 main 和同名 Public 仓库 | `CT-REPOSITORY-001` | Complete |
| `PUBLIC-04` | N/A | 文档、脚本、构建、远端与发布证据 | `pnpm docs:check` | In Progress |

## Acceptance and evidence

- `node --test scripts/source-license-metadata.test.mjs scripts/release-publication.test.mjs`
- `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, `pnpm build`
- `git bundle verify <external-backup>` 与公开树 secret scan 通过。
- GitHub API 证明新 repository ID 与旧 ID 不同、visibility 为 Public、默认分支为 `main`、普通 heads/tags 只有预期引用、安全能力与元数据已恢复。
- 一次带版本 tag 的 GitHub Actions run 在四个原生 runner 上通过；R2 与 GitHub Release 的规范化资产、大小和 SHA-256 一致。
- 目标 OS 签名/notarization 所需 secret 缺失时必须明确记录，不能把未执行的平台验收写成通过。
- 完成证据写回后进入 `Verified` 并立即执行 `pnpm work:archive -- CHG-2026-043-public-source-release`。

实施中证据（`2026-08-24`）：

- 旧 Private 仓库 ID 为 `1295489589`（`R_kgDOTTeaNQ`）；删除后同名 Public 仓库 ID 为 `1345008131`（`R_kgDOUCsyAw`），创建时间为 `2026-08-24T13:48:03Z`，证明是新的 repository identity。
- `/Users/atlan/Documents/Aivo-history-before-public-20260824.bundle` 包含删除前 28 个本地/远端/工具 refs 并通过 `git bundle verify`，SHA-256 为 `be84b460b302342d6ffab928d51a537b503f6b42f8b8dabb87a021889f425047`。`/Users/atlan/Documents/Aivo-github-backup-20260824/` 保存 repository、branch、tag、release、Action、hook、deployment 和远端 ref 元数据及 SHA-256 清单。
- 新根 bundle 与源码 tar 分别为 `/Users/atlan/Documents/Aivo-public-root-20260824.bundle` 和 `/Users/atlan/Documents/Aivo-public-root-20260824.tar.gz`，SHA-256 分别为 `f99c5c18392209c343eb1423f4fd94f6ef05856a855225f16c861bfdb1e57401` 与 `df7f899e6b8ef2103a8eade38c607cac201ce455a3495cbdced4b89be1090cf0`；两者均已验证且未上传。
- Gitleaks `8.30.1` 对新根提交 `edca100afbfbd8a7169b396dd8d2ee503d959139` 扫描 1,184 个 tracked 文件、约 6.1 MB，无 secret finding。新 Public `main` 初始提交正是该无 parent 根提交。
- 新仓库最终只有 `main`，无 tag、PR、Issue、Release 或 Action；旧 Action run `29030534610` 返回 404，旧 main SHA `75ac8a7d1091e2df5c8ed70d4dc0f24a1247529b` 返回 422。description 和八个 topics 已恢复，secret scanning、push protection、private vulnerability reporting 均已启用。
- `v0.1.0` 标签、两个 package manifest 与同名 Release record 的绑定校验通过；`CT-LICENSE-001` 与 `CT-RELEASE-001` 聚焦测试 9/9 通过；`pnpm scripts:test` 全部通过；`pnpm docs:check` 通过（90 Markdown、45 YAML、21 Requirements、23 Test IDs、21 ADRs、44 Work Packages、25 archived Work Packages）；`pnpm lint` 通过并保留既有 Fast Refresh warnings；`pnpm build` 通过并保留既有 bundle-size warning。
- `pnpm test:core` 首次因 compaction test 读取本机 models.dev cache 而得到 1,050,000 而非内建 400,000 context limit；测试隔离 `AIVO_MODELS_CACHE` 后聚焦测试与全套 `go test ./...` 通过。
- 新仓库已经配置 VaultMesh 相同的 `R2_ACCOUNT_ID`、`R2_BUCKET` 与 `R2_PUBLIC_BASE_URL`，并用 `R2_PREFIX=aivo` 隔离对象。`R2_ACCESS_KEY_ID` 与 `R2_SECRET_ACCESS_KEY` 已于 `2026-08-25` 恢复；macOS signing/notarization secrets 仍未配置，因此 `v0.1.0` 必须明确标注为未签名发行，不能宣称通过签名平台验收。
- GitHub Actions [Release quality 32763350745](https://github.com/atlantis-mk/Aivo/actions/runs/32763350745) 在 Windows x64、macOS Apple Silicon 与 Linux x64 原生 runner 上全部通过安装包生成、内嵌 Core 校验、健康启动 smoke 与 artifact 上传；对应 job 耗时 9分49秒、6分57秒、3分28秒。标签工作流另含 `macos-15-intel` 的 Intel 原生矩阵，但该平台和 R2/GitHub 最终 publication 仍必须由带 Release record 的真实版本 tag 验证。
- `v0.1.0` 首次标签 run `32764995082` 的四个原生平台构建与 smoke 全部通过，R2 七个不可变对象、stable manifest 和回读校验也通过；GitHub 在创建草稿后用 tag API 查询草稿返回 404，导致 GitHub assets 阶段安全失败且未上传 asset。恢复实现改为先取得草稿 Release ID，再使用 ID API，并增加针对既有不可变标签的手动恢复入口，避免移动或重写 `v0.1.0`。
- 首次手动恢复 run `32766778278` 再次通过四平台构建，但在 R2 计划前拒绝了注释标签的多行 `git show` 输出；未比较或修改 R2 对象。恢复路径改为显式解引用 `v0.1.0^{commit}`，并可验证首轮 run 的 tag commit、六个成功前置 job 后下载其原始 artifacts，仅恢复 GitHub 草稿，避免重新生成或覆盖不可变发行包。

## Security and data lifecycle

公开前扫描将进入新根的所有 tracked 内容，不读取或提交本机 credentials。GitHub/R2 secret 只进入 Actions job 环境；publication 脚本只记录文件名、大小、内容类型和 SHA-256。仓库外 bundle 与 GitHub 元数据备份可能包含旧历史，只保存在本机且不上传。删除旧仓库会移除 repository-scoped security/audit context，新仓库必须重新启用安全能力。

## Compatibility and migration

同一 GitHub URL 将指向新的 repository identity 和根历史；旧 clone 必须重新 clone 或显式迁移，旧 SHA、Action/PR/Release 链接不保证有效。公开源码从新根起适用 PolyForm Noncommercial 1.0.0。Release asset 命名和 R2 `releases/v<version>/` 前缀为新契约；本 Work 不增加应用内 updater compatibility。

## Bug root cause (type=bug only)

N/A（`type: governance`）。
