# 贡献指南

感谢你参与 Aivo。这个项目能够执行本地代码并访问用户选择的工作区，因此权限边界、数据安全和可验证性优先于功能数量。

## 开始之前

1. 阅读 [`AGENTS.md`](AGENTS.md) 和 [`docs/00-spec-index.md`](docs/00-spec-index.md)。它们定义当前产品范围、架构所有权、Work 流程和完成门禁。
2. 对安全边界、公共契约、依赖、许可、迁移或发布产生影响的改动，先创建或复用 Work Package；局部且可回滚的既有行为修复可以按 `AGENTS.md` 的直接修改规则处理。
3. 新功能在对应 Work 进入 `Accepted` 前不得修改产品代码。不要自行提升 Future 或 Out of scope 能力。
4. Issue、测试、日志和截图中都不得包含 API key、refresh token、credential store、本地数据库、用户 prompt、private repository、auth session、签名材料或其他敏感数据。

## 本地验证

```bash
pnpm install --frozen-lockfile
pnpm docs:check
pnpm scripts:test
pnpm test:core
pnpm lint
pnpm build
```

开发迭代可以先运行聚焦测试，但 Pull Request 必须说明最终运行了哪些命令、哪些目标平台验收不适用或仍待完成。不得删除测试、降低断言或修改 fixture 来隐藏问题。

## Pull Request

- 保持改动聚焦，说明问题、行为、非目标、风险和验证证据。
- 关联适用的 Work ID、Requirement ID 和 Test ID。
- 涉及 UI 时提供无秘密的截图；涉及平台能力时说明 OS、架构和安装形态。
- 失败、取消、重复、超时、回滚和兼容路径必须实现或明确说明为何不适用。
- `Verified` Work 必须在同一任务封存；合并本身不代表已验证或已发布。

## 贡献许可

提交贡献即表示你有权提交该内容，并同意该贡献按照仓库当前的 `PolyForm-Noncommercial-1.0.0` 许可证提供。第三方代码或资产必须保留来源、版权和兼容许可证信息；不要提交来源或授权不清晰的材料。

公开 Pull Request 不会自动把贡献的商业再许可权或版权转让给 `atlantis-mk`。若项目计划在商业授权版本中包含非平凡的第三方贡献，维护者必须在合并前取得贡献者另行签署的书面贡献协议；没有该协议时，不得宣称项目商业许可覆盖该贡献。
