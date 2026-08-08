# ADR-0008：幂等工作流与内部通知

* 状态：Accepted
* 日期：2026-08-04
* 对应版本：0.8.0

## 背景

OpenClaw 命令可能因用户双击、网络超时或客户端重连而重复提交。仅在服务层先查后写不能关闭并发窗口。工作流状态此前依赖详情轮询，且完成或待审批后没有可持久化的用户消息。

## 决策

完整分析使用 `tenant_id + workflow_type + game_id + campaign_id + analysis_date` 形成幂等键，并由数据库唯一索引最终裁决。竞争请求读取已存在工作流。

状态浏览使用租户隔离的 SSE 快照流。WAITING_APPROVAL 和终态生成一条 INTERNAL 通知；`tenant_id + workflow_id + event_type` 保证通知只生成一次，已读操作本身幂等。

外部 Slack、邮件、Webhook 或 OpenClaw/Hermes 投递不在本决策范围。通知 metadata 必须保存 `external_delivery=NOT_CONFIGURED`，不能把站内记录当作外部投递成功。

## 结果

API 可以安全重试，页面刷新后能够恢复状态和消息；代价是分析日期相同的主动重跑需要未来增加显式 force/new_run 语义。外部通知接入时应使用 Outbox 和适配器，不修改当前幂等含义。
