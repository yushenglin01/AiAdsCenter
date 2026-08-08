# ADR-0011：标准批次契约与 Kafka Inbox 消费

* 状态：Accepted
* 日期：2026-08-05
* 决策迭代：DEV-20260805-002

## 背景

平台已有文件上传和 AppsFlyer 直连，但第三方业务系统采集的 AppsFlyer、Adjust、广告平台及自有收入数据缺少稳定接入契约。若为每个来源单独开发导入逻辑，会产生字段语义、幂等和分析触发行为不一致。Kafka 的至少一次交付还可能在消费者崩溃或重平衡时重复写入事实表。

## 决策

定义版本化 `adnova.ingestion.batch` 1.0.0，统一声明生产方、批次 ID、租户、游戏业务编码、数据集、来源、UTC 日期区间、写入模式和记录。HTTP 与 Kafka Adapter 复用同一个导入服务；Kafka 只作为外部数据入口，内部 Agent 任务继续使用 Asynq。

Kafka Worker 按实例固定 tenant 与 producer，关闭自动 Offset 提交。消费前使用事件身份、Kafka 位置和载荷哈希写 MySQL Inbox；事实导入成功后才提交 Offset。永久错误写入 DLQ，临时错误不提交。成功批次合并到带版本号的分析窗口，防抖后执行现有确定性分析流水线。

## 结果

第三方系统可以用同一契约通过同步 HTTP 或 Kafka 接入，重复交付不会产生重复业务效果，契约错误可以审计和重放。代价是部署需要外部 Kafka 集群；一个 Worker 配置只服务一个 tenant/producer；当前分析窗口最终仍触发整个游戏重算，不是日期增量计算。

## 回滚或替代

停用 `GAI_KAFKA_ENABLED` 即可停止流式消费，HTTP 标准批次可独立保留。执行 000012 down 会删除 Inbox、分析窗口和导入任务新增的批次溯源字段，因此生产回滚前必须保留相关审计数据。未来可增加 Schema Registry、OAuth/SCRAM 和范围分析，但不得绕过标准导入服务或直接向事实表写入。
