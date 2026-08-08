# Schema 归档清单

* 归档版本：项目 0.4.0 / Schema legacy minimal
* 归档日期：2026-08-04
* 来源文件：`internal/business/service/service.go` 原 `outputSchema` 常量
* 对应迭代：DEV-20260804-004；由 DEV-20260804-005 归档
* 文件校验值：`business-agent-schema.json` = `df5600c37ba65b14c0947ab006b829a948d2417f00a1d7fb0a357099ac491827`
* 主要变更：0.4.1 使用完整 `business-agent-output` 1.0.0 契约替代仅声明顶层必填字段的旧契约。
* 是否兼容旧版本：新 Schema 收紧结构；当前 Mock 输出兼容，未验证第三方模型历史输出。
