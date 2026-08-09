# 真实数据接入准备

本文用于把 AdNova 从 Demo 数据切换到单公司真实数据。首次联调建议使用隔离的测试库、只读凭证和少量历史日期，验证通过后再接生产数据源。

## 1. 切换运行模式

复制 `.env.example` 为 `.env`，至少修改以下配置：

```dotenv
GAI_ENVIRONMENT=development
GAI_DEMO_SEED=false
GAI_TENANT_DEFAULT_ID=<新生成的公司 UUID>
JWT_SECRET=<至少 32 字符的随机值>
MYSQL_PASSWORD=<随机值>
MYSQL_ROOT_PASSWORD=<随机值>
```

首次联调可以保留 `GAI_ENVIRONMENT=development`，但必须关闭 Demo Seed 并使用独立数据库卷。共享或正式环境应切换为 `production`；此时服务会拒绝 Demo Seed、演示 JWT、HTTP 注册地址、日志邮件和空邮箱域名白名单。

如果当前 MySQL 卷已经执行过 Demo Seed，不要只修改开关后继续使用。应新建隔离数据库或完成经过审核的数据清理，避免 Demo 游戏、计划和真实数据混在同一租户。

## 2. 初始化公司和首管理员

服务镜像包含一次性 `bootstrap-admin` 命令。密码只从标准输入读取，不出现在参数、Compose 配置或进程列表中；必须为 12–128 位并包含大小写字母、数字和符号。

先启动 MySQL：

```bash
docker compose up -d mysql
```

再从本机安全读取密码并初始化。下面的环境变量只存在于当前 Shell，不要写进仓库：

```bash
read -s ADNOVA_INITIAL_ADMIN_PASSWORD
printf '%s' "$ADNOVA_INITIAL_ADMIN_PASSWORD" | docker compose run --rm -T --entrypoint /app/bootstrap-admin api \
  --tenant-slug your-studio \
  --tenant-name 'Your Studio' \
  --admin-username platform.admin \
  --admin-email admin@example.com \
  --admin-name '平台管理员'
unset ADNOVA_INITIAL_ADMIN_PASSWORD
```

命令会执行迁移，创建公司、标准角色、基线分析规则、不可登录的系统 Agent 身份和首个 ACTIVE 管理员。使用相同公司、用户名和邮箱重试不会重置密码；身份冲突会直接失败。

初始化成功后启动其余服务：

```bash
docker compose up -d --build
```

登录后先在“游戏管理”创建真实游戏、渠道、广告计划和素材映射，再导入指标。所有外部 ID 必须与源系统一致，币种与时区需要在首次导入前确认。

## 3. 选择第一条真实数据链路

### AppsFlyer 只读拉取

适合先验证 MMP 安装与付费事件。所需信息：

- AppsFlyer API Token，仅配置在服务端 `GAI_APPSFLYER_API_TOKEN`；
- 每个游戏的 Android package name 或 iOS `id...` App ID；
- 付费事件名，默认 `af_purchase`，多个事件用逗号配置；
- 用于校对的 AppsFlyer 后台日期、时区和汇总值。

在“数据导入”保存游戏到 App ID 的映射。连接显示 `READY` 后，先同步昨天单日，再扩大到最多 7 天。接口只读，不会调用广告平台写 API。

### Adjust 只读拉取

适合已使用 Adjust 的游戏。服务端需配置：

- `GAI_ADJUST_API_TOKEN`；
- activation、payer、revenue 三个 Report Service 指标 slug；
- 每个游戏的 Adjust App Token（仅作为连接映射保存，不是 API Token）。

在“数据导入”保存 Adjust 映射，显示 `READY` 后先同步昨天单日。手工单次最多 31 天；连接器只调用 Report Service GET 接口，不调用 Adjust 或广告平台写接口。

如需配置后自动拉取，再设置：

```dotenv
GAI_MMP_AUTO_SYNC_ENABLED=true
GAI_MMP_AUTO_SYNC_INTERVAL=1h
GAI_MMP_AUTO_SYNC_LOOKBACK_DAYS=3
```

Asynq Worker 会在启动时和每个间隔扫描所有租户的 ACTIVE/READY AppsFlyer、Adjust 连接。滚动回看会覆盖同来源、游戏和日期范围，适合吸收延迟归因修正；启用前应先完成单日人工核对。

### 标准 HTTP 批次

适合已有数据中台或定时 ETL。向 `POST /api/v1/imports/batches` 提交 `adnova.ingestion.batch` 1.0.0；契约位于 `configs/schemas/standard-ingestion-batch-v1.0.0.json`。

生产方必须稳定生成 `batch_id`，并固定 `producer.system`。同一生产方和批次 ID 重放时内容必须完全一致。建议先各准备一个广告、MMP、游戏收入和素材批次，用页面与源报表核对总行数、花费、安装和收入。

### Kafka 批次

Kafka 与 HTTP 使用同一消息体。配置 `GAI_KAFKA_*` 后用以下方式启动：

```bash
docker compose --profile kafka up -d --build
```

联调前需确认 Broker、TLS/SASL、Topic、Consumer Group、DLQ、固定租户 UUID 和生产方编码。消费端采用 Inbox、手动提交 Offset 与分析窗口租约，但仍需在测试环境验证重放、毒消息和 DLQ 处理流程。

### 真实 LLM

真实经营指标接入不要求立即启用真实模型。建议先保持 `LLM_PROVIDER=mock` 验证确定性指标，再配置 OpenAI-compatible 服务：

```dotenv
LLM_PROVIDER=openai-compatible
LLM_BASE_URL=https://<provider>/v1
LLM_API_KEY=<secret>
LLM_MODEL=<model-id>
```

启用后用一个低风险计划跑经营分析，检查结构化响应、模型用量、超时重试和审批链路。模型只负责解释和建议，不负责计算经营指标。

## 4. 首批验收标准

- Demo Seed 已关闭，数据库中没有 Demo 游戏、计划或指标；
- `/health` 的 MySQL、Redis 均正常，Worker 消费队列正常；
- 游戏、渠道、计划、素材的外部 ID、币种、国家和时区与源系统一致；
- 单日导入的源记录数、标准化行数、跳过行数可解释；
- 花费、展示、点击、安装、付费人数和收入与源报表在约定误差内；
- 相同文件或 `batch_id` 重放不产生重复事实；
- 指标重算结果可复现，异常规则阈值已经按真实业务校准；
- API Token、模型 Key、SMTP 密码、Kafka 密码均未进入 Git、日志或前端响应；
- 建议和审批没有触发任何广告平台写操作。

## 5. 当前连接器边界

目前已有 AppsFlyer Raw Data Pull、Adjust Report Service、标准 HTTP 批次和 Kafka 消费。Meta、Google、TikTok 原生拉取 API 尚未实现；这些来源现阶段应通过标准批次或文件导入接入。系统没有广告平台写入工具。
