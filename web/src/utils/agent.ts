import type { AgentRuntime, AgentTaskSnapshot } from '@/types/workflow'

export interface AgentPresentation {
  displayName: string
  role: string
  action: string
}

export interface ToolMetadata {
  name: string
  label: string
  description: string
  access: '只读' | '内部写入' | '受控操作'
}

const presentations: Record<string, AgentPresentation> = {
  'openclaw-agent': { displayName: 'OpenClaw 调度 Agent', role: '交互与编排', action: '受理指令并编排工作流' },
  'data-agent': { displayName: '数据 Agent', role: '数据可信度', action: '校验数据并重新计算指标' },
  'attribution-agent': { displayName: '归因 Agent', role: '渠道归因', action: '分析渠道与 MMP 归因差异' },
  'creative-agent': { displayName: '素材 Agent', role: '素材表现', action: '检测素材衰退与疲劳风险' },
  'research-agent': { displayName: '研究 Agent', role: '实时研究', action: '检索并整理可核验市场资料' },
  'business-agent': { displayName: '经营 Agent', role: '经营决策', action: '生成经营建议与审批请求' },
  'report-agent': { displayName: '报告 Agent', role: '结论固化', action: '生成可追溯分析报告' },
}

const toolDescriptions: Record<string, Omit<ToolMetadata, 'name'>> = {
  assess_data_quality: { label: '评估数据质量', description: '检查数据覆盖、新鲜度和导入完整性。', access: '只读' },
  recalculate_metrics: { label: '重新计算指标', description: '使用确定性公式重新计算经营指标。', access: '内部写入' },
  materialize_business_rules: { label: '执行经营规则', description: '运行已配置规则并固化分析发现。', access: '内部写入' },
  validate_imported_data: { label: '校验导入数据', description: '校验已导入广告、归因和收入数据。', access: '只读' },
  calculate_install_gap: { label: '计算安装差异', description: '比较渠道与 MMP 的安装归因差异。', access: '只读' },
  calculate_revenue_gap: { label: '计算收入差异', description: '比较渠道、MMP 与游戏收入差异。', access: '只读' },
  detect_mmp_delay: { label: '检测归因延迟', description: '识别延迟回传和归因修正信号。', access: '只读' },
  persist_attribution_findings: { label: '保存归因发现', description: '把确定性归因发现写入分析记录。', access: '内部写入' },
  calculate_fatigue_score: { label: '计算疲劳分', description: '根据生命周期、频次和转化表现计算素材疲劳度。', access: '只读' },
  detect_ctr_decline: { label: '检测 CTR 衰退', description: '识别点击率持续下降的素材。', access: '只读' },
  detect_high_frequency: { label: '检测高频曝光', description: '识别频次过高带来的疲劳风险。', access: '只读' },
  detect_high_spend_low_conversion: { label: '检测低效消耗', description: '识别高消耗但低转化的素材。', access: '只读' },
  persist_creative_findings: { label: '保存素材发现', description: '把素材风险写入分析记录。', access: '内部写入' },
  get_campaign_metrics: { label: '读取计划指标', description: '读取指定广告计划的确定性经营指标。', access: '只读' },
  get_business_benchmark: { label: '读取经营基准', description: '读取游戏历史经营基准作为比较背景。', access: '只读' },
  get_attribution_anomalies: { label: '读取归因异常', description: '读取指定计划已经计算的归因异常。', access: '只读' },
  get_creative_findings: { label: '读取素材风险', description: '读取指定计划已经计算的素材发现。', access: '只读' },
  get_verified_research_sources: { label: '读取核验研究', description: '读取人工核验且保留来源的市场资料。', access: '只读' },
  forecast_ltv: { label: '计算 LTV 代理', description: '基于当前 D7 观测值返回确定性 LTV 代理。', access: '只读' },
  create_recommendation: { label: '创建经营建议', description: '生成建议记录，但不执行广告平台变更。', access: '内部写入' },
  create_approval_request: { label: '创建审批请求', description: '为高风险建议创建人工审批请求。', access: '内部写入' },
  search_public_web: { label: '实时联网检索', description: '通过受控搜索连接器检索最新公开网页。', access: '受控操作' },
  import_web_result_for_review: { label: '登记联网结果', description: '把选中的搜索结果登记为待核验来源。', access: '内部写入' },
  list_verified_sources: { label: '读取核验来源', description: '只读取已经人工核验的研究资料。', access: '只读' },
  source_contract: { label: '校验来源契约', description: '校验来源 URL、发布方和发布时间。', access: '只读' },
  preserve_provenance: { label: '保留来源链', description: '保留来源、发现方式和内容哈希。', access: '内部写入' },
  compose_analysis_report: { label: '编排分析报告', description: '整合 Agent 结论形成报告快照。', access: '内部写入' },
  get_analysis_report: { label: '读取分析报告', description: '读取已生成的可追溯报告。', access: '只读' },
  verify_source_digest: { label: '验证来源摘要', description: '校验报告引用来源的摘要哈希。', access: '只读' },
  start_analysis_workflow: { label: '启动分析工作流', description: '创建并启动受控的多 Agent 工作流。', access: '受控操作' },
  get_workflow_status: { label: '读取工作流状态', description: '查询工作流和步骤执行状态。', access: '只读' },
  list_pending_approvals: { label: '读取待审批项', description: '查看等待人工决策的经营建议。', access: '只读' },
  list_notifications: { label: '读取内部消息', description: '读取工作流产生的内部通知。', access: '只读' },
  mark_notification_read: { label: '更新消息状态', description: '把内部通知标记为已读。', access: '内部写入' },
}

export function agentPresentation(name: string): AgentPresentation {
  return presentations[name] || { displayName: name, role: '自定义 Agent', action: '执行已注册任务' }
}

export function toolMetadata(name: string): ToolMetadata {
  return { name, ...(toolDescriptions[name] || { label: name.replaceAll('_', ' '), description: '该工具尚未补充中文说明。', access: '受控操作' as const }) }
}

export function runtimeStatusLabel(status: AgentRuntime['runtime_status']) {
  return ({ RUNNING: '处理中', QUEUED: '排队中', FAILED: '最近失败', IDLE: '空闲' })[status]
}

export function taskAction(task: AgentTaskSnapshot) {
  const action = agentPresentation(task.agent_name).action
  if (task.status === 'PENDING') return `等待执行：${action}`
  if (task.status === 'RETRYING') return `正在重试：${action}`
  return action
}

export function elapsedLabel(startedAt: string | undefined, now = Date.now()) {
  if (!startedAt) return '尚未开始'
  const seconds = Math.max(0, Math.floor((now - new Date(startedAt).getTime()) / 1000))
  if (seconds < 60) return `${seconds} 秒`
  if (seconds < 3600) return `${Math.floor(seconds / 60)} 分 ${seconds % 60} 秒`
  return `${Math.floor(seconds / 3600)} 小时 ${Math.floor((seconds % 3600) / 60)} 分`
}
