# Agent 任务状态机

Business Agent 由 API 接收、事务 Outbox 投递并由 Asynq Worker 异步执行。HTTP 202 只表示请求已持久化，不表示分析完成。

```text
PENDING -> RUNNING -> SUCCEEDED
    |          |    -> WAITING_APPROVAL -> SUCCEEDED
    |          |    -> MANUAL_REVIEW
    |          +----> RETRYING -> RUNNING
    |                         -> FAILED
    +--------------------------> CANCELLED
```

| 起始 | 目标 | 条件 | 持久化副作用 |
|---|---|---|---|
| 无 | PENDING | task 与 outbox 在同一事务创建 | `current_step=OUTBOX_PENDING` |
| PENDING | PENDING | 投递成功 | `queued_at`、`current_step=QUEUED` |
| PENDING/RETRYING | RUNNING | Worker 条件认领；或 RUNNING 心跳超过任务超时后恢复 | `started_at`、`last_heartbeat_at`、队列重试次数 |
| RUNNING | SUCCEEDED | 输出校验、派生结果与报告保存成功，且没有待审批建议 | `current_step=COMPLETED`、`finished_at` |
| RUNNING | WAITING_APPROVAL | 报告和建议已生成，至少一个高风险建议需要人工审批 | `current_step=APPROVAL_CREATION`，不写 finished_at |
| WAITING_APPROVAL | SUCCEEDED | 最后一个 PENDING 审批被 ADMIN/MANAGER 原子决策 | `current_step=COMPLETED`、`finished_at` |
| RUNNING | MANUAL_REVIEW | 两次模型输出均未通过确定性校验 | 原始响应和校验错误，`finished_at` |
| RUNNING | RETRYING | 可重试错误且 Asynq 重试未耗尽 | `current_step=RETRY_SCHEDULED`、错误分类 |
| RUNNING/RETRYING | FAILED | Asynq 重试耗尽 | `current_step=RETRY_EXHAUSTED`、`finished_at` |
| PENDING | CANCELLED | 尚未执行的任务被取消（保留状态，当前未暴露 API） | `finished_at` |

状态转换必须使用 `tenant_id + task_id + from_status` 条件更新；受影响行数不是 1 时视为并发或非法转换。幂等键为 `tenant_id + game_id + campaign_id + analysis_date + BUSINESS_ANALYSIS`，Asynq TaskID 使用业务 `task_id`，报告和 Outbox 均以 `task_id` 唯一。

恢复约束：Worker 每 5 秒更新心跳；仅当 RUNNING 的 `last_heartbeat_at` 早于任务超时时间才允许重新认领。每次重试先删除本 task 的未完成派生结果；WAITING_APPROVAL、SUCCEEDED、FAILED、MANUAL_REVIEW、CANCELLED 的重复交付直接成功返回。

审批状态机独立为 `PENDING -> APPROVED | REJECTED | CANCELLED | EXPIRED`。当前 HTTP 仅开放 APPROVED/REJECTED；决策必须对 PENDING 做条件更新，同时更新 recommendation，在同一事务写 audit_logs。任务发起者、approval.requested_by 和 SYSTEM_AGENT 均不得审批；批准不触发广告平台调用。
