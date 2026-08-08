# Contributing

感谢你参与 AdNova。提交改动前，请先通过 Issue 说明较大的功能或架构调整；小型修复可以直接提交 Pull Request。

## 开发流程

1. Fork 仓库并从 `main` 创建功能分支。
2. 不要提交 `.env`、真实凭证、客户数据、数据库导出或私有证书。
3. 保持改动聚焦，并为新增行为补充测试和文档。
4. 运行 `./scripts/test.sh`；若只修改后端或前端，也至少运行对应测试。
5. 提交 Pull Request，说明改动目的、影响和验证方式。

## 提交规范

建议使用简洁的 Conventional Commits 风格，例如 `feat: add connector`、`fix: reject invalid batch`、`docs: clarify deployment`。

参与本项目即表示你同意按照项目的 MIT License 提供贡献。
