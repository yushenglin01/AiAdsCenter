# 错误码契约

当前 HTTP API 使用统一响应 `{ code, message, data, request_id }`。`code=0` 表示成功；错误同时以 HTTP 状态和稳定语义标识表达。当前实现尚未为所有业务错误分配独立整数码，客户端必须优先处理 HTTP 状态，不能解析中文 `message` 做流程判断。

| HTTP | 语义标识 | 场景 | 是否可重试 |
|---|---|---|---|
| 400 | BAD_REQUEST | JSON、日期或路径参数格式错误 | 修正请求后 |
| 401 | UNAUTHORIZED | Token 缺失、过期、签名无效或登录失败 | 刷新或重新登录 |
| 403 | FORBIDDEN | RBAC 不允许、SYSTEM_AGENT 决策、OPERATOR 决策或审批自己的建议 | 否 |
| 404 | NOT_FOUND | 资源不存在或不属于当前公司 | 否 |
| 409 | CONFLICT | 状态转换冲突、审批已决策或并发冲突 | 读取最新状态后 |
| 422 | VALIDATION_FAILED | 导入、业务参数或 Agent 输出校验失败 | 修正数据后 |
| 429 | PROVIDER_RATE_LIMIT | 外部模型或 AppsFlyer 限流 | 是，有限退避 |
| 502 | UPSTREAM_INVALID_OR_UNAVAILABLE | AppsFlyer 不可用、响应无法验证或达到安全上限 | 缩小范围或稍后重试 |
| 500 | INTERNAL_ERROR | 未分类服务端错误 | 谨慎重试 |
| 503 | DEPENDENCY_UNAVAILABLE | MySQL、Redis 或外部模型不可用 | 是 |

Agent 内部错误分类：`INVALID_REQUEST`、`CONFIGURATION`、`AUTHENTICATION`、`RATE_LIMIT`、`UNAVAILABLE`、`TIMEOUT`、`MALFORMED_RESPONSE`。不得把失败响应或校验失败写成 `SUCCEEDED`。
