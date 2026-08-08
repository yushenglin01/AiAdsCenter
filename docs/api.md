# 阶段十二 API（项目 1.2.0）

所有业务 API 使用 `/api/v1` 前缀，响应格式为 `{ code, message, data, request_id }`。

机器可读的当前契约为 [contracts/api-contract.yaml](contracts/api-contract.yaml)，旧版快照位于 `docs/archive/api-contracts/`。

## 接入约定

- 本地默认地址：`http://localhost:8080`。
- 除 `/health`、注册配置、注册、邮箱确认、重发确认、登录和刷新端点外，所有端点都要求 `Authorization: Bearer <access_token>`。
- JSON 请求使用 `Content-Type: application/json`；导入接口也接受 `multipart/form-data`。
- ID 均为 UUID 字符串；日期使用 `YYYY-MM-DD`，时间戳使用带时区的 RFC 3339。
- 服务端从 Access Token 确定 `tenant_id`，客户端不能通过请求参数切换租户。跨租户资源统一按不存在处理。
- 客户端可传 `X-Request-ID` 和 `X-Trace-ID`；未传时由服务端生成。两者会写入响应头，`request_id` 还会出现在 JSON 响应体中。
- 当前列表接口返回 JSON 数组，不提供游标或总数。带 `limit` 的接口若未传、非正数或超过上限，会回退到各接口默认值。

成功响应示例：

```json
{
  "code": 0,
  "message": "ok",
  "data": {},
  "request_id": "b60cbfa9-f435-48a5-a8cb-9fb9d9f93c6a"
}
```

错误响应示例：

```json
{
  "code": 10003,
  "message": "permission denied",
  "data": null,
  "request_id": "b60cbfa9-f435-48a5-a8cb-9fb9d9f93c6a"
}
```

稳定通用错误码为 `10001` 参数错误、`10002` 未认证、`10003` 无权限、`10004` 不存在、`10005` 状态冲突和 `10500` 服务端错误。业务调用仍应优先按 HTTP 状态分支，完整约定见 [错误码契约](contracts/error-codes.md)。

## 认证与快速调用

成员注册状态依次为 `PENDING_EMAIL`、`PENDING_APPROVAL`、`ACTIVE`，管理员也可将申请置为 `REJECTED`，停用账号为 `DISABLED`。邮箱确认不等于授权；只有 `ACTIVE` 成员可以登录。登录接受公司邮箱或用户名，返回 Access Token 与 Refresh Token；`expires_in` 是 Access Token 的有效秒数。Refresh Token 只能提交给刷新端点，不能用于 Bearer 认证。

```bash
# 登录
curl -s http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -H 'X-Request-ID: api-doc-example-login' \
  -d '{"username":"admin","password":"Demo@123456"}'

# 刷新 Token 对
curl -s http://localhost:8080/api/v1/auth/refresh \
  -H 'Content-Type: application/json' \
  -d '{"refresh_token":"<refresh_token>"}'

# 访问受保护端点
curl -s http://localhost:8080/api/v1/auth/me \
  -H 'Authorization: Bearer <access_token>'
```

Token 响应的 `data`：

```json
{
  "access_token": "<jwt>",
  "refresh_token": "<jwt>",
  "token_type": "Bearer",
  "expires_in": 7200
}
```

系统角色包括 `ADMIN`、`MANAGER`、`OPERATOR`、`ANALYST`、`VIEWER`，不同业务写操作的角色组合并非简单的逐级继承。读接口默认允许任意登录用户；具体写权限以接口表为准。审批另有职责分离约束：任务或建议发起者不能审批自己的高风险建议。

## 列表筛选

| Endpoint | Query | Default / Limit | Notes |
|---|---|---|---|
| `/campaigns` | `game_id` | - | 按游戏筛选 |
| `/creatives` | `campaign_id` | - | 按计划筛选 |
| `/metrics/*`、`/analysis/rules`、`/analysis/attribution`、`/analysis/creative` | `game_id`；趋势额外支持 `campaign_id` | - | UUID，必须属于当前租户 |
| `/research/sources` | `game_id`、`campaign_id`、`category`、`status`、`limit` | 50 / 最大 100 | `category=POLICY|COMPETITOR|MARKET`；`status=PENDING|VERIFIED|REJECTED` |
| `/notifications` | `status` | 固定最多 50 | `UNREAD` 或 `READ` |
| `/recommendations` | `status`、`action`、`risk_level`、`limit` | 50 / 最大 100 | 倒序返回 |
| `/approvals` | `status`、`action`、`limit` | 50 / 最大 100 | 常用状态为 `PENDING|APPROVED|REJECTED` |
| `/model-usage` | `limit` | 100 / 最大 200 | 仅 ADMIN/MANAGER |
| `/audit-logs` | `action`、`resource_type`、`actor_id`、`task_id`、`limit` | 100 / 最大 200 | 仅 ADMIN |

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | No | MySQL 与 Redis 健康检查 |
| GET | `/api/v1/auth/registration-config` | No | 读取注册开关、允许的公司邮箱域名和密码规则 |
| POST | `/api/v1/auth/register` | No | 提交内部成员申请并发送邮箱确认；重复申请返回通用受理结果 |
| POST | `/api/v1/auth/verify-email` | No | 消费一次性限时确认 Token，进入待管理员授权状态 |
| POST | `/api/v1/auth/resend-verification` | No | 按冷却时间重发确认邮件；响应不暴露账号是否存在 |
| POST | `/api/v1/auth/login` | No | 使用公司邮箱或用户名、密码登录；仅 ACTIVE 成员可登录 |
| POST | `/api/v1/auth/refresh` | No | 刷新 Token 对 |
| GET | `/api/v1/auth/me` | Bearer | 当前用户与角色 |
| GET | `/api/v1/tenants/current` | Bearer | JWT 绑定的当前租户 |
| GET | `/api/v1/admin/ping` | ADMIN | RBAC 验证端点 |
| GET | `/api/v1/admin/registration-applications` | ADMIN | 按状态读取成员申请 |
| GET | `/api/v1/admin/roles` | ADMIN | 读取可授予的人类用户角色，不包含 SYSTEM_AGENT |
| POST | `/api/v1/admin/registration-applications/:id/approve` | ADMIN | 为已确认邮箱的申请分配角色并授权登录 |
| POST | `/api/v1/admin/registration-applications/:id/reject` | ADMIN | 驳回待授权申请并记录原因 |
| POST/GET | `/api/v1/games` | ADMIN/MANAGER 写；登录用户读 | 创建或列出游戏 |
| GET/PUT | `/api/v1/games/:id` | ADMIN/MANAGER 写；登录用户读 | 查询或更新游戏 |
| POST/GET | `/api/v1/channels` | ADMIN 写；登录用户读 | 创建或列出渠道 |
| GET/PUT | `/api/v1/channels/:id` | ADMIN 写；登录用户读 | 查询或更新渠道 |
| POST/GET | `/api/v1/campaigns` | ADMIN/MANAGER 写；登录用户读 | 创建或列出广告计划 |
| GET/PUT | `/api/v1/campaigns/:id` | ADMIN/MANAGER 写；登录用户读 | 查询或更新计划 |
| POST/GET | `/api/v1/creatives` | ADMIN/MANAGER/OPERATOR 写；登录用户读 | 创建或列出素材 |
| GET/PUT | `/api/v1/creatives/:id` | 同上 | 查询或更新素材 |
| POST | `/api/v1/imports/ad-metrics` | ADMIN/MANAGER/OPERATOR | 导入广告指标 |
| POST | `/api/v1/imports/mmp-metrics` | ADMIN/MANAGER/OPERATOR | 导入 MMP 指标 |
| POST | `/api/v1/imports/game-revenue` | ADMIN/MANAGER/OPERATOR | 导入游戏收入 |
| POST | `/api/v1/imports/creative-metrics` | ADMIN/MANAGER/OPERATOR | 导入素材指标 |
| POST | `/api/v1/imports/batches` | ADMIN/MANAGER/OPERATOR | 按标准批次契约导入第三方广告、MMP、收入或素材数据 |
| GET | `/api/v1/imports`、`/imports/:id` | 登录用户 | 导入记录 |
| GET | `/api/v1/mmp-connections` | 登录用户 | MMP 连接、凭证配置布尔状态与健康；不返回 Token |
| PUT | `/api/v1/mmp-connections/appsflyer` | ADMIN/MANAGER | 保存游戏到 AppsFlyer App ID 的映射 |
| GET | `/api/v1/mmp-sync-runs` | 登录用户 | AppsFlyer 同步运行、行数、告警和安全错误分类 |
| POST | `/api/v1/mmp-connections/:id/sync` | ADMIN/MANAGER/OPERATOR | 同步最多 7 天安装与指定付费事件 |
| POST | `/api/v1/metrics/recalculate?game_id=...` | ADMIN/MANAGER/OPERATOR/ANALYST | 重算指标并执行三类分析 |
| GET | `/api/v1/metrics/overview` | 登录用户 | 经营总览与去重风险计数 |
| GET | `/api/v1/metrics/campaigns`、`/campaigns/:id` | 登录用户 | 计划聚合指标与详情 |
| GET | `/api/v1/metrics/trends` | 登录用户 | 按日趋势，可传 game_id、campaign_id |
| GET | `/api/v1/analysis/rules` | 登录用户 | 规则配置与经营风险发现 |
| PUT | `/api/v1/analysis/rules/:id` | ADMIN/MANAGER | 修改 threshold、consecutive_days、enabled |
| GET | `/api/v1/analysis/attribution`、`/:id` | 登录用户 | 归因异常列表与详情 |
| GET | `/api/v1/analysis/creative`、`/:id` | 登录用户 | 素材疲劳发现列表与详情 |
| GET | `/api/v1/research/sources` | 登录用户 | 研究来源列表，可按范围、分类与审核状态筛选 |
| POST | `/api/v1/research/sources` | ADMIN/MANAGER/ANALYST | 登记无凭证 HTTPS 研究来源，初始状态 PENDING |
| POST | `/api/v1/research/sources/:id/verify` | ADMIN/MANAGER | 核验来源，使其可进入 Research/Business Agent |
| POST | `/api/v1/research/sources/:id/reject` | ADMIN/MANAGER | 驳回来源，必须填写原因 |
| POST | `/api/v1/analysis/business` | ADMIN/MANAGER/OPERATOR/ANALYST | HTTP 202 接收幂等经营分析任务 |
| GET | `/api/v1/analysis/business/tasks` | 登录用户 | 最近经营分析任务 |
| GET | `/api/v1/analysis/business/tasks/:id` | 登录用户 | 任务、尝试、用量、发现、建议、审批和报告详情 |
| GET | `/api/v1/analysis/business/tasks/:id/events` | 登录用户 | SSE 状态快照；终态关闭，25 秒要求重连 |
| GET | `/api/v1/analysis/business/tasks/:id/report` | 登录用户 | 成功任务的 Markdown 报告快照 |
| GET | `/api/v1/analysis/business/health` | 登录用户 | Agent Provider 健康状态与工具能力 |
| GET | `/api/v1/agents`、`/agents/:name` | 登录用户 | 七类 Agent 目录、执行模式、能力和健康状态 |
| POST | `/api/v1/workflows/analysis` | ADMIN/MANAGER/OPERATOR/ANALYST | 启动完整多 Agent 分析工作流 |
| GET | `/api/v1/workflows`、`/workflows/:id` | 登录用户 | 工作流与每个 Agent 步骤状态 |
| GET | `/api/v1/workflows/:id/events` | 登录用户 | 工作流 SSE；状态变化时推送，25 秒要求重连 |
| POST | `/api/v1/openclaw/commands` | ADMIN/MANAGER/OPERATOR/ANALYST | OpenClaw 内部命令入口，支持工作流、审批 Inbox 与通知指令 |
| GET | `/api/v1/notifications` | 登录用户 | OpenClaw 内部消息；可用 status=UNREAD/READ 筛选 |
| POST | `/api/v1/notifications/:id/read` | 登录用户 | 幂等标记当前租户消息为已读 |
| GET | `/api/v1/recommendations`、`/:id` | 登录用户 | 建议、审批和任务状态；支持筛选 |
| GET | `/api/v1/approvals`、`/:id` | 登录用户 | 审批、建议、任务和报告详情 |
| POST | `/api/v1/approvals/:id/approve` | ADMIN/MANAGER | 批准 PENDING 审批，只更新状态 |
| POST | `/api/v1/approvals/:id/reject` | ADMIN/MANAGER | 驳回 PENDING 审批，必须填写原因 |
| GET | `/api/v1/model-usage`、`/summary` | ADMIN/MANAGER | 模型调用、Token、延迟与成本 |
| GET | `/api/v1/audit-logs` | ADMIN | Agent Tool 与审批审计 |
| GET | `/api/v1/dashboard/operations` | 登录用户 | 审批、任务、模型与审计汇总 |

经营分析请求：

```json
{
  "game_id": "30000000-0000-4000-8000-000000000001",
  "campaign_id": "50000000-0000-4000-8000-000000000001",
  "analysis_date": "2026-07-30"
}
```

OpenClaw 命令请求：

```json
{
  "intent": "RUN_FULL_ANALYSIS",
  "input": {
    "game_id": "30000000-0000-4000-8000-000000000001",
    "campaign_id": "50000000-0000-4000-8000-000000000001",
    "analysis_date": "2026-08-04"
  }
}
```

RUN_FULL_ANALYSIS 返回 HTTP 202；其他 OpenClaw 查询/内部状态指令返回 HTTP 200。支持的 intent 为 `RUN_FULL_ANALYSIS`、`GET_WORKFLOW_STATUS`、`LIST_PENDING_APPROVALS`、`LIST_NOTIFICATIONS`、`MARK_NOTIFICATION_READ`。同一租户、工作流类型、游戏、计划和分析日期组成工作流幂等键；重复命令返回原 workflow_id。Research 没有已核验来源时以 `NO_VERIFIED_SOURCES` 成功完成且不生成外部事实；Business Agent 提交后工作流等待异步任务，高风险建议生成后为 WAITING_APPROVAL，全部审批完成后在下一次详情读取时收口为 COMPLETED。

各 intent 的 `input`：

| intent | 必填字段 | 可选字段 | HTTP |
|---|---|---|---|
| `RUN_FULL_ANALYSIS` | `game_id`、`campaign_id` | `analysis_date`，默认服务端当前日期 | 202 |
| `GET_WORKFLOW_STATUS` | `workflow_id` | - | 200 |
| `LIST_PENDING_APPROVALS` | 无 | - | 200 |
| `LIST_NOTIFICATIONS` | 无 | `notification_status=UNREAD|READ` | 200 |
| `MARK_NOTIFICATION_READ` | `notification_id` | - | 200 |

命令成功时，`data` 为 `{ intent, status, data }`；运行工作流时 `status=ACCEPTED`，其他命令完成时 `status=SUCCEEDED`。当前入口是受信任的站内适配边界，仍使用用户 Bearer Token 与相同 RBAC，不是匿名 Webhook。

研究来源登记示例：

```json
{
  "game_id": "30000000-0000-4000-8000-000000000001",
  "campaign_id": "50000000-0000-4000-8000-000000000001",
  "category": "MARKET",
  "title": "公开市场报告",
  "summary": "用于解释市场背景的摘要，不能替代内部经营指标。",
  "source_url": "https://example.com/reports/market",
  "publisher": "Example Research",
  "published_at": "2026-08-04"
}
```

来源只接受绝对 HTTPS URL，禁止 URL 用户名/密码；登记后为 PENDING。核验请求体为 `{ "comment": "来源与摘要已核对" }`，驳回 comment 必填。Research Agent 只读取分析日期当天及以前的 VERIFIED 来源。

创建来源按当前租户内 `source_url + title + summary` 的规范化内容做幂等去重。若传 `campaign_id`，必须同时传 `game_id`，且二者必须属于同一租户范围。`published_at` 不能是未来日期。

## SSE 状态订阅

经营分析任务和工作流分别通过以下端点订阅：

- `GET /api/v1/analysis/business/tasks/:id/events`
- `GET /api/v1/workflows/:id/events`

请求需要 Bearer Token，并应发送 `Accept: text/event-stream`。服务端只在状态版本变化时推送快照；进入终态后关闭连接。非终态连接约 25 秒后发送 `reconnect` 事件并关闭，客户端应重新建立连接；当前不支持 `Last-Event-ID` 续传。

```text
event: workflow
data: {"run":{"id":"...","status":"RUNNING","current_step":"RESEARCH"},"steps":[]}

event: reconnect
data: {"workflow_id":"..."}
```

工作流的流式终止状态为 `WAITING_APPROVAL`、`COMPLETED`、`FAILED`、`MANUAL_REVIEW`、`CANCELLED`。任务流在任务自身终态后关闭；收到 `reconnect` 或连接中断后，客户端也可以先调用详情端点校准最终状态。

内部消息在 COMPLETED、WAITING_APPROVAL、FAILED、MANUAL_REVIEW 或 CANCELLED 时生成一次。当前 channel=INTERNAL，metadata 中的 external_delivery=NOT_CONFIGURED；接口不表示 Slack、邮件或 Webhook 已发送。

幂等键为 `tenant_id + game_id + campaign_id + analysis_date + BUSINESS_ANALYSIS`。高风险建议只创建 `PENDING` 审批请求，不执行预算或素材变更。

提交端点始终返回 HTTP 202：新任务先以 `PENDING` 持久化，再由 Outbox 投递；幂等命中会返回已有任务，但不会重复生成结果。没有高风险审批时任务进入 SUCCEEDED；存在高风险建议时进入 WAITING_APPROVAL，报告已经可读，最后一个 PENDING 审批决策后进入 SUCCEEDED。

审批请求体为 `{ "comment": "..." }`。驳回意见必填；批准意见可选。任务发起者不能审批自己的高风险建议，OPERATOR/ANALYST/VIEWER/SYSTEM_AGENT 均返回 403；重复决策返回 409。每次成功、被拒或冲突的决策都有审计记录，成功审计明确 `advertising_platform_called=false`。

0.4.1 起，任务对象保存 `schema_name`、`schema_version`；用量对象保存 `prompt_name`、`prompt_version`、`schema_version`，用于追踪每次结构化结果的契约来源。这是向后兼容的响应字段新增。

导入端点同时接受 multipart 表单（`game_id`、`source`、`file`）或 JSON：

```json
{
  "game_id": "30000000-0000-4000-8000-000000000001",
  "source": "META",
  "file_name": "meta_ad_metrics.json",
  "records": []
}
```

广告 CSV 字段：`date,campaign_external_id,country,currency,spend,impressions,clicks,installs`。其他格式参考 `examples/generated/`。

## 标准批次与 Kafka 接入

第三方业务系统应优先使用 `adnova.ingestion.batch` 1.0.0 契约。完整字段约束位于 [标准批次 JSON Schema](../configs/schemas/standard-ingestion-batch-v1.0.0.json)，可直接联调的 AppsFlyer 示例位于 [standard_mmp_batch.json](../examples/generated/standard_mmp_batch.json)。同一份 JSON 既可提交到 `POST /api/v1/imports/batches`，也可作为 Kafka 消息 value。

批次公共字段包括：

- `batch_id`：生产方稳定生成的批次事件 ID；相同 `producer.system + batch_id` 重放时内容必须一致。
- `dataset_type`：`AD_METRICS`、`MMP_METRICS`、`GAME_REVENUE` 或 `CREATIVE_METRICS`。
- `source`：数据来源，例如 `APPSFLYER`、`ADJUST`、`META`、`GOOGLE`、`TIKTOK` 或 `INTERNAL`。
- `game_code`：当前租户内的游戏业务编码，不使用数据库 UUID 作为外部系统耦合键。
- `timezone`：首版固定 `UTC`；`period` 为闭区间，所有记录日期必须落在该区间。
- `write_mode`：默认 `APPEND`；只有 MMP 延迟回补允许 `REPLACE_RANGE`。

HTTP 接入从 JWT 确定租户；请求中的 `tenant_id` 若存在，必须与 JWT 一致。Kafka Worker 采用“一个消费者实例固定一个租户和生产方”的部署边界，消息中的 `tenant_id`、`producer.system` 必须分别匹配 `GAI_KAFKA_TENANT_ID`、`GAI_KAFKA_PRODUCER_SYSTEM`。单批最多 10,000 行，消息体和 HTTP 请求体上限为 10MB。

默认建议按数据集使用四个 Topic：

```text
adnova.ingestion.ad-metrics.v1
adnova.ingestion.mmp-metrics.v1
adnova.ingestion.game-revenue.v1
adnova.ingestion.creative-metrics.v1
```

Kafka 消费使用手动提交 Offset。Worker 先写 MySQL Inbox 并完成事实导入，再提交 Offset；消费者崩溃时消息可能被再次交付，但相同事件只产生一次业务效果。永久契约错误在写入 `adnova.ingestion.dlq.v1` 后提交原 Offset；临时数据库错误不提交并由后续重试恢复。DLQ 保存来源位置、错误分类、载荷 SHA-256 与 Base64 原始载荷，便于修复后受控重放。

成功导入会合并相同租户、游戏和日期范围的分析窗口。默认等待 60 秒吸收相邻 AD/MMP/收入/素材批次，然后执行现有确定性指标、经营规则、归因和素材分析流水线。窗口认领默认持有 15 分钟租约并使用随机 claim token 隔离旧 Worker；进程崩溃后可重新认领，延迟返回的旧执行不能覆盖新状态。当前流水线仍按整个游戏重算，窗口用于防抖与安全重放，并不表示已经实现按日期增量计算。

AppsFlyer 连接配置示例：

```json
{
  "game_id": "30000000-0000-4000-8000-000000000001",
  "external_app_id": "com.example.game",
  "status": "ACTIVE"
}
```

同步请求为 `{ "from": "2026-08-01", "to": "2026-08-03" }`。连接器并行读取官方 Raw Data Pull API v5 的 `installs_report` 和 `in_app_events_report`，以 UTC 日期和 USD 聚合；campaign_id 必须能匹配当前租户的广告计划。相同连接和日期范围重复提交返回已有成功运行。未配置 `GAI_APPSFLYER_API_TOKEN`、达到 20 万行/50MB、负收入或计划未映射时明确失败。
