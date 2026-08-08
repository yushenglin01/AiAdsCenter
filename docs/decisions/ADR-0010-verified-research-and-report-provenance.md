# ADR-0010：研究来源先核验，报告保存可验证溯源

* 状态：Accepted
* 日期：2026-08-05
* 决策迭代：DEV-20260805-001

## 背景

Research Agent 需要政策、竞品和市场背景，但让模型自由访问任意网页会引入来源伪造、提示注入、跨租户泄漏和结果不可复现风险。报告若只保存 Markdown，也无法证明它使用了哪些 Agent 输出和研究来源。

## 决策

研究资料先登记为 PENDING，只接受无凭证 HTTPS URL；ADMIN/MANAGER 人工核验后变为 VERIFIED。Research Agent 和 Business Agent 只能按 tenant、game、campaign 与 analysis_date 读取 VERIFIED 来源，不提供任意 HTTP Tool。

报告由确定性 Report Composer 生成，保存生成器名称/版本、来源 ID、workflow/task 标识、生成时间和输入的 SHA-256 摘要。外部研究只作为背景，经营结论必须继续引用确定性指标或分析发现。

## 结果

研究内容可以被审计、按日期复现并在报告中追踪；代价是外部资料需要人工登记和核验，尚不能自动发现最新市场事件。

## 回滚或替代

可通过 000011 down 删除来源库和报告新增字段并回退 0.9.0。未来自动研究连接器必须进入同一 PENDING/VERIFIED 状态机，不能绕过核验或直接写入 Prompt。
