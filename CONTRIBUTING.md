# Contributing

感谢你参与 AdNova。提交改动前，请先通过 Issue 说明较大的功能或架构调整；小型修复可以直接提交 Pull Request。

## 开发流程

1. Fork 仓库并从 `develop` 创建短期工作分支。
2. 不要提交 `.env`、真实凭证、客户数据、数据库导出或私有证书。
3. 保持改动聚焦，并为新增行为补充测试和文档。
4. 运行 `./scripts/test.sh`；若只修改后端或前端，也至少运行对应测试。
5. 向 `develop` 提交 Pull Request，说明改动目的、影响和验证方式。

## 分支模型

仓库采用精简的 Git Flow。只有 `main` 和 `develop` 是长期分支，其他分支完成合并后应删除。

| 分支 | 起点 | 合并目标 | 用途 |
|---|---|---|---|
| `main` | — | — | 已发布、可部署的稳定代码；版本提交使用 `vX.Y.Z` Tag |
| `develop` | `main` | `main`（经 release） | 下一版本的日常集成分支 |
| `feature/<issue>-<name>` | `develop` | `develop` | 新功能，例如 `feature/42-kafka-dashboard` |
| `fix/<issue>-<name>` | `develop` | `develop` | 非生产紧急缺陷修复 |
| `docs/<name>` / `chore/<name>` | `develop` | `develop` | 文档或工程维护 |
| `release/vX.Y.Z` | `develop` | `main`，随后同步回 `develop` | 发布候选，只接受版本、文档和发布阻塞修复 |
| `hotfix/vX.Y.Z` | `main` | `main`，随后同步回 `develop` | 线上紧急修复 |

### 常规功能

```bash
git switch develop
git pull --ff-only
git switch -c feature/42-kafka-dashboard
# 开发、测试、提交并推送，然后向 develop 创建 PR
```

### 发布

从 `develop` 创建 `release/vX.Y.Z`，完成版本号、变更日志和发布验证后向 `main` 创建 PR。合并后在 `main` 创建同名版本 Tag，并把 `main` 同步回 `develop`，避免发布修复丢失。

### 紧急修复

从 `main` 创建 `hotfix/vX.Y.Z`，向 `main` 提交 PR；发布并打 Tag 后，再把 `main` 同步回 `develop`。

## 合并规则

- 禁止直接推送或强制推送 `main`、`develop`，所有改动通过 Pull Request 合并。
- 合并前解决全部 Review 对话，并运行与改动相关的测试。
- 使用 Squash Merge 或 Rebase Merge 保持线性历史；推荐 Squash Merge。
- 一个 Pull Request 只处理一个明确主题，合并后删除源分支。
- 不在 `main` 上开发；`main` 只接收 release、hotfix 或必要的仓库治理变更。

## 提交规范

建议使用简洁的 Conventional Commits 风格，例如 `feat: add connector`、`fix: reject invalid batch`、`docs: clarify deployment`。

参与本项目即表示你同意按照项目的 MIT License 提供贡献。
