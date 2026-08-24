# vX.Y.Z Aivo 发布主题 / Release theme

用一段简短中文介绍说明版本价值、适用用户和主要变化。

Add a concise English introduction describing the release value, audience, and main changes.

## 新增 / Highlights

- 中文用户亮点。
  Matching English highlight.
- 中文用户亮点。
  Matching English highlight.

## 下载 / Download

请根据系统和芯片选择安装包。Choose the package matching your operating system and architecture.

| 系统 System | 芯片 Chip | 格式 Format | 下载 Download |
| --- | --- | --- | --- |
| Windows | x64 | NSIS `.exe` | `[Aivo_X.Y.Z_windows-x86_64-setup.exe](matching tagged asset URL)` |
| macOS | Apple Silicon | `.dmg` | `[Aivo_X.Y.Z_darwin-aarch64.dmg](matching tagged asset URL)` |
| macOS | Intel | `.dmg` | `[Aivo_X.Y.Z_darwin-x86_64.dmg](matching tagged asset URL)` |
| Linux | x64 | `.AppImage` | `[Aivo_X.Y.Z_linux-x86_64.AppImage](matching tagged asset URL)` |

List the two matching macOS ZIP links as alternate downloads.

## 完整性校验 / Integrity

Link `SHA256SUMS`, provide concise verification commands, and state that GitHub and immutable R2 assets have matching SHA-256 digests.

## 安装提示 / Installation notes

State the actual signing and notarization status. Keep operating-system warnings visible and never describe unsigned packages as signed or silently installed.

## 源码与许可 / Source and license

Identify Aivo as source-available and noncommercial rather than OSI open source. Link the canonical license and explain the commercial-authorization boundary.

## 发布记录 / Release record

- Status: Draft
- Release date: YYYY-MM-DD
- Git tag: `vX.Y.Z`
- Desktop/Electron build: TBD
- Go core build: TBD
- Persistence schema: TBD
- Local API version: TBD

### Delivered

Every Work below must already be present in `changes/archive.json`. Creating a Release or tag must not change its status, body, or archive entry.

| Work ID | Requirement/ADR | Type | User-visible outcome |
| --- | --- | --- | --- |
| - | - | CHG/BUG/SEC/DEP/MIG | - |

### Compatibility and migration

Record schema/data, local API/RPC/IPC, minimum OS, settings, provider/credential behavior, upgrade/downgrade, irreversible migration, and compensation. Write “none” when unchanged.

### Verification evidence

| Gate/platform | Result | Evidence |
| --- | --- | --- |
| GATE-1..7 | Pending | - |
| macOS package/sign/notarize | Pending | - |
| Windows package/sign | Pending | - |
| Linux package | Pending | - |

### Known issues and compensation limits

List remaining BUG/SEC IDs, affected versions, workaround, rollback version, and compensation when rollback is impossible. Write “none” when there are no known issues.
