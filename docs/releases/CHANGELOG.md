# 变更日志

## [Unreleased]

### Added

* 企业内部成员页面注册、邮箱确认与管理员角色授权流程。
* 注册域名白名单、确认邮件 SMTP/开发日志适配器及成员申请管理页面。

### Changed

* 登录同时支持公司邮箱和用户名，仅 ACTIVE 成员可获得 Token。

### Fixed

* 无。

### Deprecated

* 无。

### Removed

* 无。

### Security

* 邮箱确认 Token 仅存 SHA-256 摘要，注册重复响应不暴露账号是否存在；生产环境强制 HTTPS、公司域名白名单和 STARTTLS SMTP。

## [1.2.0] - 2026-08-09

### Added

* 新增 `PENDING_EMAIL → PENDING_APPROVAL → ACTIVE` 企业成员状态机、注册/确认/重发 API 与管理员申请审核 API。
* 新增注册页、邮箱确认页、管理员成员申请列表、角色分配和驳回交互。
* 新增 `000014_member_registration` migration、邮件发送适配器与注册全链路测试。

### Changed

* 登录标识兼容用户名和公司邮箱；Demo 用户补齐已确认、已授权状态。
* API、Compose、环境示例和版本归档升级到 1.2.0。

### Security

* 只有 ACTIVE 成员可签发 JWT；生产环境必须配置公司邮箱域名白名单、HTTPS 公网地址和 STARTTLS SMTP。
* 原始确认 Token 不落库，注册/重发响应避免账号枚举，密码策略要求至少 12 位且包含四类字符。

## [1.1.1] - 2026-08-08

### Added

* 新增分析窗口租约与随机 claim token，并通过 `000013_analysis_window_leases` 升级现有数据卷。

### Changed

* 标准批次金额和频次在入库前按实际 MySQL `DECIMAL` 列范围与 6 位小数精度校验。

### Fixed

* 修复 ingestion worker 在分析执行期间崩溃后窗口永久停留在 PROCESSING 的问题。
* 修复旧 Worker 延迟完成可能覆盖重新认领执行状态的问题。
* 修复超范围或超精度数值直到数据库写入时才失败的问题。

### Security

* claim token 仅用于数据库内部 fencing，不通过 API 返回；租约配置不包含凭证或业务载荷。

## [1.1.0] - 2026-08-05

### Added

* 新增 `adnova.ingestion.batch` 1.0.0 标准批次、HTTP 导入端点与 AF/Adjust/广告/自有数据示例。
* 新增 Kafka ingestion worker、MySQL Inbox、DLQ 和带版本认领的分析窗口 dispatcher。
* 新增 Kafka TLS、可选 SASL/PLAIN、手动 Offset 提交与 Compose `kafka` profile。

### Changed

* HTTP 与 Kafka 统一复用导入服务、游戏业务编码解析、事实表校验和批次幂等语义。
* 成功 Kafka 批次在防抖窗口后自动执行现有确定性指标与三类分析。

### Fixed

* 避免 Kafka 至少一次交付、消费者崩溃和重平衡导致重复事实写入或重复分析完成标记。

### Security

* 消费者实例固定 tenant/producer 并与消息交叉校验；TLS 默认开启，日志不记录原始载荷。
* `REPLACE_RANGE` 仅允许 MMP 数据，DLQ 明确要求使用受控 ACL 与加密保留策略。

## [1.0.0] - 2026-08-05

### Added

* 新增 Data Agent 数据覆盖、新鲜度与导入质量报告。
* 新增 Research 来源登记、人工核验、租户隔离 API 与 Agent 工作台管理界面。
* 新增报告生成器版本、研究来源 ID、SHA-256 摘要和 provenance JSON。
* OpenClaw 新增工作流状态、待审批列表、通知列表和通知已读指令。

### Changed

* Research Agent 从固定 SKIPPED 升级为只读取 VERIFIED 来源的可执行 Agent。
* Business Agent/Prompt 升级为 1.1.0，允许读取已核验研究背景，但经营结论仍必须使用确定性指标证据。
* Report Agent 与 OpenClaw Agent 升级为可执行、可健康检查的运行时定义。
* Element Plus 改为组件级样式与路由级低频控件加载，Vite 将 Vue、Element Plus、ECharts 和 zrender 拆为稳定缓存块。

### Fixed

* 修复 Data Agent 无论数据是否缺失都报告 VALIDATED 的错误语义。
* 修复历史分析可能混入分析日期之后研究资料的可复现性问题。
* 修复日期选择器、徽标、数值输入和开关未注册导致页面控件无法渲染，以及前端仍显示 0.9.0/阶段九的问题。

### Security

* Research 来源仅接受无凭证 HTTPS URL，登记与核验角色分离；未核验来源不会进入 Prompt。
* OpenClaw 不包含自动审批、广告预算、出价、状态或素材写入指令。

## [0.9.0] - 2026-08-04

### Added

* 新增 AppsFlyer Raw Data Pull API v5 只读连接器、mmp_connections、mmp_sync_runs 与 000010 migration。
* 新增连接配置/健康、手工同步/记录 API，以及数据导入页 AppsFlyer 入口。

### Changed

* AppsFlyer 安装和付费事件由确定性代码聚合后复用现有 MMP 导入、指标和 Agent 工作流数据边界。

### Fixed

* 真实上游达到行数或响应体上限时明确失败，避免把截断报告当成完整数据。

### Security

* API Token 只从服务端环境变量读取；官方 HTTPS Host 白名单、租户隔离、RBAC 和审计覆盖配置与同步。
* AppsFlyer Client 只调用 GET 报告接口，不包含广告或 MMP 写入能力。

## [0.8.0] - 2026-08-04

### Added

* 新增工作流 SSE、OpenClaw 内部消息列表和幂等已读接口。
* 新增 workflow_notifications 与 000009 migration、前端内部消息中心。

### Changed

* 完整分析按租户、类型、游戏、计划、日期建立数据库级幂等键；并发重试回读胜出的工作流。
* Agent 工作台从轮询切换为带清理与退避重连的 SSE。

### Security

* 通知查询和已读操作按 tenant_id 隔离；channel 固定 INTERNAL，外部投递明确 NOT_CONFIGURED。

## [0.7.0] - 2026-08-04

### Added

* 新增 Agent Registry、七 Agent 工作流、workflow_runs/workflow_steps 和 000008 migration。
* 新增 Agent 目录、工作流查询/启动、OpenClaw 命令 API 与 Vue Agent 工作台。

### Changed

* AgentExecutor 使用通用 AgentInput/AgentResult；Business 任务增加 workflow、parent 和 agent 标识。
* Data Agent 在指标重算后物化经营规则，Business Agent 继续复用安全异步队列与审批链路。

### Security

* Research 不伪造外部信息；OpenClaw 仅启动和查询工作流，批准仍不触发广告平台。

## [0.6.0] - 2026-08-04

### Added

* 新增建议中心、审批中心、审计日志、模型用量和 Dashboard 运营汇总 API/页面。
* 新增 WAITING_APPROVAL 工作流、审批策略单测、阶段六角色/审计 E2E。
* 新增 audit_logs、审批上下文字段、系统 Agent 身份和 ADR-0006。

### Changed

* 高风险 Agent 任务生成报告后进入 WAITING_APPROVAL，最后一个审批完成后进入 SUCCEEDED。
* 审批 requested_by 统一为不可登录的 SYSTEM_AGENT；旧审批由 000007 回填动作、风险和报告关联。

### Fixed

* 修复审批单只能展示、无法形成决策闭环的问题。
* 修复建议、审批、任务和审计可能分步成功导致状态不一致的问题。

### Deprecated

* 无。

### Removed

* 无广告平台执行能力；批准接口不包含任何外部执行调用。

### Security

* OPERATOR、ANALYST、VIEWER、SYSTEM_AGENT 和任务发起者均不得自审批；成功、拒绝和冲突尝试全部审计。
* Agent Tool 调用只审计安全元数据，不记录隐藏推理、完整 Prompt 或密钥。

## [0.5.0] - 2026-08-04

### Added

* 新增 MySQL 事务 Outbox、Asynq v0.26.0 队列投递与独立 Worker 消费。
* 新增任务心跳租约、有限重试、失败终态、SSE 状态快照和确定性 Markdown 报告。
* 新增 `000006_async_workflow_and_reports` migration、队列契约单测与 ADR-0005。

### Changed

* `POST /api/v1/analysis/business` 从同步 200 改为异步 202；幂等语义保持不变。
* 前端经营分析页改为订阅任务状态，刷新后恢复监听，并支持报告下载。
* Go 最低版本提升到 1.24，以使用当前 Asynq 依赖链。

### Fixed

* 消除数据库任务已创建但 Redis 投递失败时的不可恢复双写窗口。
* 重试前清理任务的部分派生结果，终态重复交付直接短路。

### Deprecated

* 无。

### Removed

* 移除 Business Agent 的 HTTP 同步执行路径和 Worker 启动占位逻辑。

### Security

* 队列载荷在执行前与数据库中的 tenant、game、campaign、creator 身份逐项匹配；SSE 与报告读取继续执行租户隔离。

## [0.4.1] - 2026-08-04

### Added

* 新增迭代索引、正式发布记录、ADR、API/Schema/Prompt/Migration 归档与自动完整性检查。
* 新增完整 Business Agent JSON Schema 1.0.0 和版本化 Prompt 1.0.0。
* 新增 `000000_schema_migrations` 引导 migration 与 `000005_version_agent_contracts` migration。
* Stage 4 E2E 支持通过 `PLAYWRIGHT_CHROMIUM_EXECUTABLE` 使用已安装的 Chromium。

### Changed

* 用带 SHA-256 历史校验的顺序 SQL migration runner 替代 GORM AutoMigrate。
* Agent 任务保存 schema_name/schema_version，模型用量保存 prompt_name/prompt_version/schema_version。
* 本地测试脚本默认使用 `/tmp/gai-go-cache`，兼容受限开发环境。

### Fixed

* 修复 Prompt、Schema 与模型用量无法形成完整版本追踪链的问题。
* 修复 Web 容器页面可用但 `localhost` 健康探针连接拒绝导致的伪不健康状态。

### Deprecated

* 非版本化 `configs/prompts/*.txt` 仅作为 0.4.0 历史来源保留，运行时不再加载。

### Removed

* 移除运行时 GORM AutoMigrate。

### Security

* 已发布 migration 校验值变化时拒绝启动，避免历史数据库契约被静默覆盖。

## [0.4.0] - 2026-08-04

### Added

* 新增受限 Business Agent、7 个白名单工具、Mock LLM 与可选 OpenAI-compatible Client。
* 新增结构化输出校验、纠错重试、任务/尝试/用量、建议和 PENDING 审批请求。
* 新增经营分析 API 与前端页面。

### Changed

* Demo 导入完成后为 Meta 示例触发一次幂等 Mock 经营分析。

### Fixed

* 无。

### Deprecated / Removed / Security

* 无。Mock 功能明确标记为 Mock，未声称真实模型或广告平台已接通。

## [0.3.0] - 2026-08-04

### Added

* 新增确定性经营指标、规则引擎、归因差异和素材疲劳分析。
* 新增指标与分析 API、趋势与分析页面。

### Changed

* Demo 数据扩展为异常、正常与扩量对照。

### Fixed

* 安全除法避免零分母产生无效指标。

### Deprecated / Removed / Security

* 无。

## [0.2.0] - 2026-08-04

### Added

* 新增游戏、渠道、广告计划、素材目录。
* 新增广告/MMP/游戏收入/素材指标 JSON 与 CSV 导入、校验、标准化和幂等去重。

### Changed

* 应用壳增加目录与导入入口。

### Fixed

* 无。

### Deprecated / Removed / Security

* 无。外部数据 Provider 是 CSV/JSON/Mock，不是真实广告平台连接。

## [0.1.0] - 2026-08-04

### Added

* 初始化 Go/Vue/Docker Compose 项目结构。
* 增加 JWT 登录/刷新、服务端 RBAC、请求上下文和单公司默认边界。

### Changed

* 无。

### Fixed

* 无。

### Deprecated / Removed

* 无。

### Security

* JWT Claims 与 Repository 查询保留 tenant_id 数据边界，但产品当前只提供单公司模式。
