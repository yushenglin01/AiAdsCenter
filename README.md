# AdNova · 星曜智投

<p align="center">
  <img src="docs/assets/adnova-hero-v2.png" alt="AdNova 让每一笔投放穿越数据噪声" width="100%" />
</p>

<p align="center"><strong>让分散的买量数据，变成可验证、可解释、可审批的增长决策。</strong></p>

AdNova（星曜智投）是面向海外游戏投放团队的广告经营智能分析平台。它统一接收广告平台、MMP、游戏收入与素材表现数据，由确定性代码计算指标，并由受限 Agent 生成解释、建议和审批单。

当前仓库完成 **阶段十三：真实 MMP 自动拉取**。AppsFlyer 与 Adjust 共用只读连接器、权威区间导入和同步审计边界；配置服务端凭证与游戏映射后，Worker 可按计划自动回拉。

当前项目版本：**1.3.0**；最近迭代：**DEV-20260809-002**。版本历史见 [迭代索引](docs/iterations/README.md) 和 [变更日志](docs/releases/CHANGELOG.md)。

## 核心边界

- 系统只生成建议与审批单，不修改广告平台预算、出价、状态或素材。
- 关键经营指标只能由确定性代码计算，LLM 不负责算数。
- Agent 只能调用注册 Tool，不直接访问数据库、Shell 或任意 HTTP 地址。
- 当前采用 Go 模块化单体，不拆微服务；已接入只读 AppsFlyer Raw Data Pull 与 Adjust Report Service，并支持第三方通过标准 HTTP/Kafka 契约推送 MMP 和广告数据；尚未接入 Meta、Google、TikTok 原生拉取 API 或任何广告平台写入 API。

## 技术栈

- 后端：Go 1.24+、Gin、GORM、MySQL 8、Redis、Asynq、franz-go（可选 Kafka）、JWT、Zap、Viper
- 前端：Vue 3、TypeScript、Vite、Pinia、Vue Router、Element Plus、Axios、ECharts
- 运行：Docker Compose（api、Asynq worker、可选 Kafka ingestion worker、web、mysql、redis）
- 测试：Go testing、testify、Vitest

## 目录

```text
cmd/server/             HTTP API 入口
cmd/worker/             Asynq Worker 与 Outbox dispatcher
cmd/ingestion-worker/   Kafka 标准数据消费与分析窗口 dispatcher
internal/auth/          认证 domain/repository/service/handler
internal/tenant/        租户读取边界
internal/middleware/    JWT、RBAC、CORS、请求上下文
internal/bootstrap/     数据库、Redis、日志、路由、Demo seed
internal/game/          游戏管理
internal/campaign/      渠道与广告计划管理
internal/creative/      素材管理
internal/ingestion/     数据校验、标准化与去重
internal/metrics/       确定性指标计算、聚合与趋势
internal/rules/         可配置规则与经营风险发现
internal/attribution/   渠道、MMP、游戏收入差异分析
internal/analysis/      阶段三同步分析流水线
internal/agent/         AgentSpec、执行契约、任务与受限 Tool Registry
internal/agents/        七类 Agent 的运行时适配器
internal/dataquality/   Data Agent 数据覆盖、新鲜度与导入质量评估
internal/research/      研究来源登记、人工核验、来源保留与 Agent 读取
internal/report/        确定性报告编排、摘要与溯源
internal/openclaw/      OpenClaw 多指令交互服务
internal/workflow/      多 Agent 工作流、步骤状态与持久化编排
internal/notification/  OpenClaw 内部工作流消息与已读状态
internal/mmp/           AppsFlyer/Adjust 连接、只读拉取、同步运行与确定性聚合
internal/llm/           Mock 与 OpenAI-compatible 结构化模型客户端
internal/business/      Business Agent、结果校验、建议与审批请求
internal/taskqueue/     Asynq 任务契约、投递器与消费处理器
internal/approval/      审批状态机、权限策略与事务决策
internal/audit/         Agent Tool 与人工决策审计
internal/recommendation/ 建议中心查询边界
internal/modelusage/    模型用量明细与汇总
internal/dashboard/     任务、审批、用量运营汇总
configs/prompts/        不可变的语义版本 Agent Prompt
configs/schemas/        版本化 Agent JSON Schema
pkg/provider/           CSV、JSON、Mock Provider
examples/generated/     30 天 JSON 与 CSV 示例
migrations/             可审查 SQL 迁移
seed/                   可选手工 Demo seed
web/                    Vue 3 应用
docs/                   架构、API、迭代、发布、ADR 与契约归档
scripts/                一键启动与测试
```

## 快速启动

要求安装 Docker 与 Docker Compose。首次运行会构建镜像、启动 MySQL/Redis、自动迁移并幂等初始化 Demo 用户。

```bash
cp .env.example .env
# 开发环境可以直接使用示例值；部署前必须更换 JWT_SECRET 和数据库密码
./scripts/start.sh
```

打开 <http://localhost:5173>。API 健康检查：<http://localhost:8080/health>。

本地默认使用 `GAI_REGISTRATION_MAIL_PROVIDER=log`，确认链接只写入 API 日志。共享或生产环境必须配置公司邮箱域名白名单、HTTPS 公网地址和支持 STARTTLS 的 SMTP：

```bash
GAI_REGISTRATION_ALLOWED_EMAIL_DOMAINS=example.com,subsidiary.example.com
GAI_REGISTRATION_PUBLIC_BASE_URL=https://adnova.example.com
GAI_REGISTRATION_MAIL_PROVIDER=smtp
GAI_REGISTRATION_MAIL_SMTP_ADDRESS=smtp.example.com:587
GAI_REGISTRATION_MAIL_SMTP_USERNAME=...
GAI_REGISTRATION_MAIL_SMTP_PASSWORD=...
GAI_REGISTRATION_MAIL_FROM_ADDRESS=no-reply@example.com
```

停止服务：

```bash
docker compose down
```

## Demo 账号

统一密码：`Demo@123456`

这些账号和密码仅用于本地 Demo。任何共享、测试或生产环境都必须关闭 Demo Seed 或更换全部凭证，并使用随机生成的数据库密码与 `JWT_SECRET`。

| 用户名 | 角色 |
|---|---|
| admin | ADMIN |
| manager | MANAGER |
| operator | OPERATOR |

当前按单公司模式运行，登录时无需选择公司。后端通过 `GAI_TENANT_DEFAULT_ID` 固定当前公司，JWT 和数据表继续保留 `tenant_id` 作为安全边界与未来扩展位。

## 本地开发与测试

后端需要本地 MySQL/Redis，或只运行不依赖外部存储的单元测试：

```bash
go test ./...
cd web && npm install && npm run dev
```

完整静态检查、单测和前端构建：

```bash
./scripts/test.sh
```

API 用法见 [docs/api.md](docs/api.md)，架构说明见 [docs/architecture.md](docs/architecture.md)。

## 数据导入、分析与审批

启动 Compose 后可一键生成并导入 30 天 Demo 数据：

```bash
./scripts/import_demo.sh
```

脚本可重复执行，不会新增重复任务或事实数据。导入完成后会同步重新计算指标、执行三类规则分析，并为 Meta 示例提交一次幂等 Mock Business Agent 异步任务。示例包含 Meta 异常样本、Google 正常对照、TikTok 扩量样本、AppsFlyer、游戏收入和素材表现；CSV 示例也位于 `examples/generated/`。

Research Agent 支持通过可选的 Brave Search API 实时检索公开网页；API Key 只保存在服务端，结果默认不持久化。启用结果入库后，用户可把选中结果登记为 PENDING 来源，仍需 ADMIN/MANAGER 人工核验后才会进入分析上下文。OpenClaw 提供内部命令、SSE 状态流、审批 Inbox 和站内消息；外部 IM/邮件/Webhook 仍为 NOT_CONFIGURED，不会伪造投递成功。

AppsFlyer 使用服务端 `GAI_APPSFLYER_API_TOKEN`；Adjust 使用 `GAI_ADJUST_API_TOKEN` 及激活、付费人数、收入三个事件指标 slug。前端和数据库只保存“凭证是否已配置”与游戏到 App ID/App Token 的映射，不保存或返回 API Token。单次手工同步默认分别最多 7 天和 31 天；开启 `GAI_MMP_AUTO_SYNC_ENABLED` 后，Asynq Worker 会遍历所有租户的就绪连接，按回看窗口每日吸收延迟归因修正。

从空库开始接真实数据时，先关闭 Demo Seed，并使用镜像内的 `bootstrap-admin` 命令初始化公司、标准角色和首管理员。完整切换步骤、凭证清单与验收标准见 [真实数据接入准备](docs/real-data-onboarding.md)。生产模式会拒绝 Demo Seed 和演示 JWT，避免真实数据误写入演示租户。

第三方接入使用 [标准批次 JSON Schema](configs/schemas/standard-ingestion-batch-v1.0.0.json)。同一份消息可调用 `POST /api/v1/imports/batches`，也可发送到配置的 Kafka Topic；示例见 [standard_mmp_batch.json](examples/generated/standard_mmp_batch.json)。接入现有 Kafka 集群时，在 `.env` 配置 `GAI_KAFKA_*`，然后启动可选 profile：

```bash
docker compose --profile kafka up -d --build
```

Compose 不内置 Kafka Broker。消费端使用 MySQL Inbox、手动 Offset 提交和 DLQ 实现至少一次传输下的业务幂等；成功批次会合并分析窗口，在防抖后执行现有确定性分析流水线。详细 Topic、租户/生产方边界和重放语义见 [API 文档](docs/api.md#标准批次与-kafka-接入)。

## Mock LLM 与真实 LLM

默认 `LLM_PROVIDER=mock`，无需外部模型即可完整演示。切换真实 OpenAI-compatible 服务时，必须同时设置 `LLM_PROVIDER=openai-compatible`、`LLM_BASE_URL`、`LLM_API_KEY` 与 `LLM_MODEL`。客户端支持结构化 JSON、超时、有限重试、Token 用量归一化和安全错误分类；不会记录 API Key 或完整敏感 Prompt。

## OpenClaw / Hermes

OpenClaw 内部命令入口为 `POST /api/v1/openclaw/commands`，可启动完整分析工作流。Hermes/外部 OpenClaw HTTP 与消息推送仍保持适配边界；当前没有对应环境，因此 Agent 目录会明确标记为未配置。

## 阶段十三完成情况

已完成：阶段一至十二全部能力；provider-neutral MMP 拉取边界；Adjust Report Service 只读连接器；AppsFlyer/Adjust 独立映射与手工同步入口；跨租户自动回拉、Provider 级日期上限、权威区间替换与同步审计。

仍未完成：企业 SSO/SCIM、邀请制注册、找回密码、管理员强制下线；真实 Kafka 集群与 MMP 凭证联调、Meta/Google/TikTok 原生只读连接器、自动化 Research 定时任务、外部消息投递、生产级不可篡改审计存储、跨进程 OpenTelemetry 与真实模型联调。系统仍没有广告平台执行工具；APPROVED 仅表示人工认可建议，不表示执行。

## 开源与安全

项目采用 [MIT License](LICENSE)。欢迎通过 Issue 或 Pull Request 参与贡献，提交前请阅读 [CONTRIBUTING.md](CONTRIBUTING.md)。

请勿把 `.env`、真实 API Token、数据库备份、客户数据或私有证书提交到仓库。安全漏洞请按 [SECURITY.md](SECURITY.md) 说明进行私密报告，不要在公开 Issue 中披露利用细节。
