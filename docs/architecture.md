# 阶段十三架构（项目 1.3.0）

当前采用模块化单体。`cmd/server` 负责 HTTP、数据库 migration 与队列投递，`cmd/worker` 运行 Asynq 消费者和 Outbox dispatcher，`cmd/ingestion-worker` 可选消费外部 Kafka 数据并触发确定性分析。业务调用方向为 Handler → Service → Repository；Handler 不访问 GORM，Repository 查询显式接收 `tenant_id`。

认证使用短期 Access Token 与长期 Refresh Token。当前按单公司模式运行，登录接口不接收公司参数，账号查询只能使用服务端 `GAI_TENANT_DEFAULT_ID`。签名 Claims 仍包含 `user_id`、`tenant_id` 和角色，为数据安全边界及后续多公司扩展保留稳定结构。RBAC 在服务端中间件执行，前端守卫只用于改善体验。

1.3.0 将 AppsFlyer 专用同步服务抽象为 provider-neutral MMP Fetcher，并增加 Adjust Report Service 只读实现。连接配置仍按 tenant/game/provider 隔离；全局 API Token 与事件指标映射只存在于服务端环境，连接查询仅返回 `credential_configured`。常驻 Worker 启动时及固定间隔扫描所有租户连接，按 Provider 最大范围执行滚动回看；SYSTEM_AGENT 发起的运行复用同一幂等、审计和权威区间导入路径。

MySQL 保存业务数据、Agent 任务、事务 Outbox 和报告快照；Redis 保存 Asynq 待执行任务与重试元数据。API 启动时按文件名顺序执行 `migrations/*.up.sql`，在 `schema_migrations` 保存 SHA-256；Worker 等待 API 健康后仅打开数据库，避免并发迁移。已发布 migration 校验值变化时拒绝启动。

导入边界通过统一 `DataProvider` 接口隔离，提供 CSV、JSON 和 Mock 实现。导入流程依次执行文件限制、格式解码、字段与类型校验、租户内关联解析、日期/国家/币种标准化，再在事务中写入事实表。文件级幂等键和事实表业务唯一索引共同去重；原始内容不会直接进入模型 Prompt。AppsFlyer 与 Adjust 的规范化结果通过 MMP 权威区间事务替换吸收延迟事件回补，文件上传仍保持追加去重语义。

1.1.0 增加 `adnova.ingestion.batch` 1.0.0 标准输入。HTTP 与 Kafka Adapter 只负责认证、传输元数据和错误映射，二者复用同一个 `ingestion.Service.ImportBatch`，因此 AF、Adjust、广告平台和业务自有数据遵循相同字段校验、游戏编码解析、事实表写入和批次幂等语义。契约强制 UTC、闭区间日期、版本、生产方与稳定 `batch_id`；单批最多 10,000 行、10MB。`REPLACE_RANGE` 只对 MMP 开放，避免广告和收入事实被第三方批次意外清空。

Kafka 是可选的外部数据入口，不替换内部 Asynq。每个 `ingestion-worker` 固定一个 tenant 与 producer，订阅配置的 Topic；TLS 默认启用，可选 SASL/PLAIN。消费端关闭自动提交，先用 `tenant + producer + batch_id` 和 `topic + partition + offset` 写入 MySQL Inbox，完成标准导入后才提交 Offset。已成功事件重放直接返回；同事件 ID 对应不同载荷哈希时进入永久冲突。契约/业务校验错误写入 DLQ 后提交，数据库等临时错误保留 Offset 并重试。日志只记录 Topic、Partition、Offset、Hash 和错误分类，不打印原始数据。

每次 Kafka 成功导入都会 upsert `analysis_windows`。窗口以 tenant、game、period 唯一，合并收到的数据集并延后执行时间，从而在默认 60 秒内吸收 AF/AD/收入等相邻批次。Dispatcher 使用版本号、租约和随机 claim token 条件认领，运行已有指标、规则、归因和素材确定性流水线；处理中到达的新批次会提高版本，使旧执行结果不能把新窗口错误标记完成。Worker 崩溃后窗口会在租约到期后重新进入可认领集合，旧 Worker 即使延迟返回也因 token 失效而不能覆盖新执行状态。当前分析实现仍按整个游戏重算，日期窗口用于防抖、可恢复调度和审计，后续可在不改变接入契约的前提下演进为范围计算。

AppsFlyer Token 只从 `GAI_APPSFLYER_API_TOKEN` 读取；Adjust Token 与 activation/payer/revenue 指标 slug 只从 `GAI_ADJUST_*` 读取。`mmp_connections` 只保存 tenant/game/provider/App ID 或 App Token 映射，API 只返回 `credential_configured` 布尔状态。连接器固定访问官方 HTTPS Host：AppsFlyer 并行拉取 Raw Data Pull API v5 installs 与指定 in-app purchase events；Adjust 拉取 Report Service 的 day/campaign/country/currency 聚合报表。两者统一转换为 installs、activations、payers、revenue 后复用 MMP Import Service。

`mmp_sync_runs` 将上游拉取状态与导入任务分离，保存日期范围、源行数、聚合行数、跳过数、import_job_id 和安全错误分类。连接+日期范围构成幂等键；已有成功/处理中运行直接返回，失败运行允许在同一逻辑 ID 上重试。单次最多 7 天，达到 20 万行或 50MB 时拒绝导入，防止截断数据进入经营指标。Token、完整原始报告和上游错误正文不持久化。

所有 HTTP 请求生成或透传 `X-Request-ID` 与 `X-Trace-ID`，Zap 记录结构化请求日志。OpenTelemetry SDK 与跨进程 Trace 将在任务链路落地阶段接入。

阶段三分析流水线依次执行指标重算、经营规则、归因差异和素材疲劳分析。指标使用十进制定点数与安全除法，分母为零时返回零，统一保留 8 位精度。规则阈值存储在 `analysis_rules`，ADMIN/MANAGER 可修改阈值、连续天数和启停状态。每条发现保存生成时的证据 JSON，前端只展示结果，不重新计算。

归因差异率定义为 `abs(A-B)/max(abs(A),abs(B))`；素材疲劳评分由近 7 日 CTR 降幅（60%）和最新频次归一值（40%）组成。两者都是确定性分析器，当前不调用 LLM。指标重算流水线仍同步；Business Agent 已迁移为可重试的异步任务，两者保持明确边界。

Business Agent 定义为 `AgentSpec + Prompt + Tools + Permissions + Model + OutputSchema`。它只能调用八个注册工具：读取计划指标、历史基准、归因异常、素材发现、已核验研究来源、确定性 LTV 代理，以及创建建议和审批请求。预算修改、暂停计划、Shell、任意 HTTP 和 Raw SQL 均未注册，因此无法从 Agent 路径调用。

模型边界由统一 `LLMClient.GenerateStructured` 隔离。Mock Client 根据结构化示例输入稳定生成结果；OpenAI-compatible Client 封装认证、JSON 请求映射、超时、429/5xx 有限重试、响应解析和安全错误分类。Provider 原始对象不会泄漏到业务层，API Key 不进入日志或数据库。

`BusinessResultValidator` 在持久化建议前校验 JSON、置信度、证据实际值与目标值、允许动作，以及高风险人工审批要求。首次失败会携带校验错误重新生成一次；第二次失败进入 `MANUAL_REVIEW`，保留原始响应和校验错误，但不创建建议。任务、模型尝试、用量、经营发现、建议与审批请求分别存储，方便后续异步恢复和审计。任务保存 Schema 名称/版本，用量保存 Prompt 名称/版本和 Schema 版本；1.0.0 运行时加载不可变 Prompt 1.1.0 与输出 Schema 1.0.0。

异步提交在同一 MySQL 事务中写入 `agent_tasks` 与 `task_outbox`。API 尝试即时投递，Worker 的 dispatcher 周期补偿未发布记录；Asynq 使用业务任务 UUID 作为 TaskID，重复投递按幂等成功处理。Worker 通过带来源状态条件的更新认领任务，并以心跳和超时租约恢复崩溃后残留的 RUNNING。失败在未耗尽时转为 RETRYING，耗尽后转 FAILED；每次重跑先删除该任务未完成的派生结果，终态重复交付直接返回。

SSE 端点每 500ms 读取租户范围内的任务快照，只在状态版本变化时发送，终态立即关闭，25 秒后提示客户端重连。成功任务同时写入一条按 task_id 唯一的确定性 Markdown 报告；报告只引用已持久化模型结果与确定性指标，不让 LLM 重新计算数值。

阶段六增加 approval、recommendation、audit、modelusage 和 dashboard 模块。含高风险建议的 Agent 任务在报告与审批单生成后进入 WAITING_APPROVAL；SSE 发送该可展示状态后关闭。每个审批都由不可登录的 SYSTEM_AGENT 身份请求，人工决策执行角色与自审批双重校验。

批准/驳回使用 MySQL 行锁和 PENDING 条件更新，在同一事务中更新 approval_requests、recommendations、必要时 agent_tasks，并追加 audit_logs。审批入口没有广告 SDK 或执行 Tool，审计元数据显式记录未调用广告平台。事务外的权限拒绝与状态冲突也追加安全审计。Agent Tool 审计只保存工具名、状态和关联 task，不保存模型隐藏推理。

Dashboard 运营汇总、模型用量和审计查询都以 tenant_id 为首要过滤条件。模型成本来自 model_usage_records 的 decimal 汇总；默认 Mock 成本为零，不伪造真实费用。

0.7.0 增加统一 Agent Registry 和持久化 Workflow Orchestrator。完整分析按 OpenClaw、Data、Attribution、Creative、Research、Business、Report 顺序记录七个步骤；Data 先重算指标并物化经营规则，归因和素材 Agent 继续执行确定性分析，Business Agent 通过原有 Outbox/Asynq 异步运行，Report Agent 读取不可变报告快照。工作流读取时会对齐 Business 任务状态。

1.0.0 增加 research_sources 审核状态机。来源登记必须提供无凭证 HTTPS URL、发布方、发布日期与摘要，初始为 PENDING；ADMIN/MANAGER 核验后变为 VERIFIED。Research Agent 只读取当前租户、游戏/计划范围和 analysis_date 之前的 VERIFIED 来源，没有来源时返回 NO_VERIFIED_SOURCES，不生成市场事实。Business Prompt 1.1.0 明确研究来源只能解释背景，不能替代确定性指标证据。

Data Agent 在重算前读取四类事实表和导入任务，输出行数、计划覆盖、最新日期、缺失/陈旧状态与失败导入数。它不会隐式触发外部同步；数据获取由显式导入、手工 MMP 同步或独立 Worker 定时任务负责，避免分析请求产生不可预期的外部副作用。

报告由 `internal/report` 的确定性 Composer 生成，保存 report-agent 版本、研究来源 ID、SHA-256 source digest、workflow/task 标识和生成时间。Report Agent 返回这些溯源字段；报告中的外部链接只来自 VERIFIED 来源。

OpenClaw Agent 现在有真实 Executor 和五类内部指令：启动分析、读取工作流、列出待审批、列出通知和标记通知已读。它不提供自动审批决策；批准/驳回仍必须走人工 RBAC 与审计接口。外部消息推送仍为 NOT_CONFIGURED，且没有广告平台执行权限。

0.8.0 为完整分析增加数据库级幂等键 `tenant + workflow_type + game + campaign + analysis_date`。服务先读取已有工作流，数据库唯一索引负责关闭并发窗口；竞争失败的请求回读胜出的持久化工作流，因此 API 重试不会重复创建步骤或 Business 任务。

Workflow SSE 每 500ms 发送状态版本变化，WAITING_APPROVAL 和终态关闭，25 秒无终态时要求客户端重连。Vue 工作台使用 AbortController 清理订阅，并以 5 秒退避恢复连接；浏览器刷新后仍以数据库状态为准。

终态和 WAITING_APPROVAL 会写入 `workflow_notifications`。`tenant_id + workflow_id + event_type` 唯一约束保证通知幂等，用户可重复执行“标为已读”。当前 channel 固定为 INTERNAL；metadata 明确 `external_delivery=NOT_CONFIGURED`，不把站内记录描述为外部投递。外部通知后续应通过 Outbox 适配器扩展，不能直接耦合 Workflow Service 与第三方 SDK。
