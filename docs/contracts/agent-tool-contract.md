# Agent 工具契约

Business Agent 只允许使用注册表中的八个工具。工具调用必须携带服务端构造的 `tenant_id`、`task_id` 与 `user_id` 上下文，模型不能覆盖这些身份字段。

| 工具 | 权限 | 副作用 | 边界 |
|---|---|---|---|
| `get_campaign_metrics` | READ_METRICS | 无 | 只读当前公司指定计划指标 |
| `get_business_benchmark` | READ_ANALYSIS | 无 | 只读当前公司基准 |
| `get_attribution_anomalies` | READ_ANALYSIS | 无 | 只读确定性归因发现 |
| `get_creative_findings` | READ_ANALYSIS | 无 | 只读确定性素材发现 |
| `get_verified_research_sources` | READ_VERIFIED_RESEARCH | 无 | 只读当前公司、分析日期以前且经人工核验的 HTTPS 来源 |
| `forecast_ltv` | READ_METRICS | 无 | 由确定性代码生成 LTV 代理，不让 LLM 算数 |
| `create_recommendation` | CREATE_RECOMMENDATION | 写建议 | 状态固定为 `PROPOSED`，不执行广告操作 |
| `create_approval_request` | CREATE_APPROVAL_REQUEST | 写审批请求 | 状态固定为 `PENDING`，必须关联建议 |

明确禁止注册：Raw SQL、Shell、任意 HTTP、预算修改、出价修改、暂停计划、上传或启用素材。高风险建议必须 `requires_approval=true`；审批单的创建不等于审批通过，更不等于外部动作完成。

Research Agent 不注册任意网页抓取工具。资料必须先由 `/research/sources` 登记为 PENDING，经 ADMIN/MANAGER 核验为 VERIFIED 后才可读取。OpenClaw 只注册工作流、审批列表和内部消息工具，不注册自动审批或广告平台执行工具。
