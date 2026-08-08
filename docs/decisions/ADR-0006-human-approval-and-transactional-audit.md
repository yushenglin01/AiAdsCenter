# ADR-0006：人工审批与事务审计

* 状态：Accepted
* 日期：2026-08-04
* 决策迭代：DEV-20260804-007

## 背景

阶段四已能生成高风险审批单，但没有决策入口，也无法证明批准没有触发外部广告平台。审批与建议状态若分开写入，还会产生相互矛盾的结果；决策成功但审计失败同样不可接受。

## 决策

审批由独立 approval 模块执行。只有 ADMIN/MANAGER 角色可以发起决策，且 task.created_by、approval.requested_by 和 SYSTEM_AGENT 均被服务层与事务层拒绝。PENDING 审批、PROPOSED 建议、最后一张审批后的 WAITING_APPROVAL 任务和成功审计日志在一个 MySQL 事务内更新。审计 `metadata.advertising_platform_called` 固定为 false；系统不注册任何预算执行 Tool。

Agent Tool 调用写入同一 audit_logs 体系，仅记录工具名、状态、actor、task 和安全错误摘要，不保存隐藏推理或完整 Prompt/输出。被拒绝和冲突的审批尝试也单独记录。

## 结果

审批可以证明“谁、何时、对什么、做了什么决定”，并保持建议/任务一致。代价是审批决策依赖审计表可用；审计写入失败会使决策事务回滚。audit_logs 当前为追加式应用约束，生产不可篡改存储和归档策略留待后续基础设施加固。
