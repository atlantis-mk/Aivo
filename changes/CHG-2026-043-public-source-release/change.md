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
| `PUBLIC-01` | `NFR-RELEASE-001` | PolyForm 许可证、source-available/商业授权边界和一致包元数据 | `CT-LICENSE-001` | Pending |
| `PUBLIC-02` | `NFR-RELEASE-001` | 版本标签驱动的四平台打包、R2 与 GitHub Release publication | `CT-RELEASE-001` | Pending |
| `PUBLIC-03` | `NFR-RELEASE-001` | 仓库外备份、秘密扫描、新根 main 和同名 Public 仓库 | `CT-REPOSITORY-001` | Pending |
| `PUBLIC-04` | N/A | 文档、脚本、构建、远端与发布证据 | `pnpm docs:check` | Pending |

## Acceptance and evidence

- `node --test scripts/source-license-metadata.test.mjs scripts/release-publication.test.mjs`
- `pnpm docs:check`, `pnpm scripts:test`, `pnpm test:core`, `pnpm lint`, `pnpm build`
- `git bundle verify <external-backup>` 与公开树 secret scan 通过。
- GitHub API 证明新 repository ID 与旧 ID 不同、visibility 为 Public、默认分支为 `main`、普通 heads/tags 只有预期引用、安全能力与元数据已恢复。
- 一次带版本 tag 的 GitHub Actions run 在四个原生 runner 上通过；R2 与 GitHub Release 的规范化资产、大小和 SHA-256 一致。
- 目标 OS 签名/notarization 所需 secret 缺失时必须明确记录，不能把未执行的平台验收写成通过。
- 完成证据写回后进入 `Verified` 并立即执行 `pnpm work:archive -- CHG-2026-043-public-source-release`。

## Security and data lifecycle

公开前扫描将进入新根的所有 tracked 内容，不读取或提交本机 credentials。GitHub/R2 secret 只进入 Actions job 环境；publication 脚本只记录文件名、大小、内容类型和 SHA-256。仓库外 bundle 与 GitHub 元数据备份可能包含旧历史，只保存在本机且不上传。删除旧仓库会移除 repository-scoped security/audit context，新仓库必须重新启用安全能力。

## Compatibility and migration

同一 GitHub URL 将指向新的 repository identity 和根历史；旧 clone 必须重新 clone 或显式迁移，旧 SHA、Action/PR/Release 链接不保证有效。公开源码从新根起适用 PolyForm Noncommercial 1.0.0。Release asset 命名和 R2 `releases/v<version>/` 前缀为新契约；本 Work 不增加应用内 updater compatibility。

## Bug root cause (type=bug only)

N/A（`type: governance`）。
