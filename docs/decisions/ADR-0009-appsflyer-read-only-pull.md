# ADR-0009：AppsFlyer 采用只读 Raw Data Pull 与确定性聚合

* 状态：Accepted
* 日期：2026-08-04
* 决策迭代：DEV-20260804-010

## 决策

首个真实 MMP 连接器使用 AppsFlyer Raw Data Pull API v5 的非自然量 installs 和 in-app events GET 接口。API Token 只由服务端环境变量提供，数据库保存租户/游戏到 AppsFlyer App ID 的映射和同步运行，不保存密钥或完整原始报告。

安装按首次打开记录聚合为 installs 与 activations；指定付费事件按日期、campaign_id、country_code 聚合，AppsFlyer ID 去重为 payers，收入以 USD 请求并由确定性代码求和。结果转换为现有 MMP 导入契约，再执行计划归属和租户边界校验；随后在一个事务中替换同一来源、游戏和日期范围的 MMP 事实，以吸收延迟事件回补。文件上传仍使用原有追加去重语义。

连接器只允许官方 `https://hq1.appsflyer.com`，单次最多 7 天；429/AppsFlyer CallLimit/5xx 有限重试。达到 20 万行或 50MB 上限、返回负收入、无效日期或本地计划未映射时拒绝导入；如果全部源记录都缺少 campaign_id，则不替换现有指标。成功同步后禁止直接更换 App ID，避免同一游戏混入两个应用的数据。

官方契约：[Raw Data Pull API overview](https://dev.appsflyer.com/hc/reference/raw_data_pull_api_tokenv2-overview)、[Installs v5](https://dev.appsflyer.com/hc/reference/get_app-id-installs-report-v5)、[In-app events v5](https://dev.appsflyer.com/hc/reference/get_app-id-in-app-events-report-v5)。

## 结果

系统获得首个真实只读数据入口，并保持指标由确定性代码计算、凭证不泄漏和广告平台无写入能力。代价是首版不处理自然量、退款/负收入和超大报告；这些场景必须通过后续明确的数据模型扩展解决，不能静默截断或改写。
