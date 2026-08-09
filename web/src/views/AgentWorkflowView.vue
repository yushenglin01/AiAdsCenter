<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ElBadge } from 'element-plus/es/components/badge/index'
import { ElDatePicker } from 'element-plus/es/components/date-picker/index'
import { ElMessage } from 'element-plus/es/components/message/index'
import 'element-plus/es/components/badge/style/css'
import 'element-plus/es/components/date-picker/style/css'
import AppShell from '@/components/AppShell.vue'
import AgentOperationsPanel from '@/components/AgentOperationsPanel.vue'
import ResearchSourceManager from '@/components/ResearchSourceManager.vue'
import WorkflowFailureDialog from '@/components/WorkflowFailureDialog.vue'
import { catalogApi } from '@/api/catalog'
import { workflowApi } from '@/api/workflows'
import type { Campaign } from '@/types/catalog'
import type { AgentCatalogItem, AgentRuntime, WorkflowDetails, WorkflowNotification, WorkflowRun } from '@/types/workflow'
import { unreadNotificationCount, workflowStatusType, workflowTerminal } from '@/utils/workflow'

const agents = ref<AgentCatalogItem[]>([])
const agentRuntimes = ref<AgentRuntime[]>([])
const campaigns = ref<Campaign[]>([])
const workflows = ref<WorkflowRun[]>([])
const notifications = ref<WorkflowNotification[]>([])
const selected = ref<WorkflowDetails>()
const campaignID = ref('')
const analysisDate = ref(new Date().toISOString().slice(0, 10))
const activeSection = ref<'workflows' | 'notifications' | 'agents' | 'sources'>('workflows')
const launchDialog = ref(false)
const launchError = ref('')
const loading = ref(true)
const starting = ref(false)
const error = ref('')
const failureDialog = ref(false)
const failureLoading = ref(false)
const failureError = ref('')
const failureNotification = ref<WorkflowNotification>()
const failureDetails = ref<WorkflowDetails>()
const streamState = ref<'IDLE' | 'CONNECTING' | 'LIVE' | 'RECONNECTING'>('IDLE')
let eventController: AbortController | undefined
let reconnectTimer: number | undefined
let runtimeTimer: number | undefined
const researchManager = ref<{ refresh: () => Promise<void> }>()

const selectedCampaign = computed(() => campaigns.value.find((item) => item.id === campaignID.value))
const canStart = computed(() => Boolean(selectedCampaign.value && analysisDate.value))
const unreadCount = computed(() => unreadNotificationCount(notifications.value))

function campaignName(id: string) {
  return campaigns.value.find((item) => item.id === id)?.name || id
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    ;[agents.value, agentRuntimes.value, campaigns.value, workflows.value, notifications.value] = await Promise.all([workflowApi.listAgents(), workflowApi.listAgentRuntime(), catalogApi.listCampaigns(), workflowApi.list(), workflowApi.listNotifications()])
    campaignID.value ||= campaigns.value[0]?.id || ''
    if (workflows.value.length) await selectWorkflow(workflows.value[0])
  } catch (cause: any) {
    error.value = cause.response?.data?.message || 'Agent 工作台加载失败'
  } finally {
    loading.value = false
  }
}

async function start() {
  if (!canStart.value || !selectedCampaign.value) return
  starting.value = true
  launchError.value = ''
  try {
    selected.value = await workflowApi.openClawCommand({ game_id: selectedCampaign.value.game_id, campaign_id: selectedCampaign.value.id, analysis_date: analysisDate.value })
    workflows.value = await workflowApi.list()
    launchDialog.value = false
    activeSection.value = 'workflows'
    ElMessage.success('OpenClaw 已受理，多个 Agent 正在协作分析')
    watchWorkflow(selected.value.workflow.workflow_id)
  } catch (cause: any) {
    launchError.value = cause.response?.data?.message || '工作流启动失败'
  } finally {
    starting.value = false
  }
}

function openLaunchDialog() {
  launchError.value = ''
  launchDialog.value = true
}

async function selectWorkflow(run: WorkflowRun) {
  stopWatching()
  try {
    selected.value = await workflowApi.get(run.workflow_id)
    watchWorkflow(run.workflow_id)
  } catch (cause: any) {
    error.value = cause.response?.data?.message || '工作流详情加载失败'
  }
}

function applyWorkflow(details: WorkflowDetails) {
  selected.value = details
  const index = workflows.value.findIndex((item) => item.workflow_id === details.workflow.workflow_id)
  if (index >= 0) workflows.value[index] = details.workflow
  else workflows.value.unshift(details.workflow)
}

function watchWorkflow(id: string) {
  stopWatching()
  if (selected.value && workflowTerminal(selected.value.workflow.status)) return
  const controller = new AbortController()
  eventController = controller
  streamState.value = 'CONNECTING'
  workflowApi.watch(id, (details) => {
    streamState.value = 'LIVE'
    applyWorkflow(details)
    void reloadAgentRuntime()
    if (details.workflow.status === 'WAITING_APPROVAL' || workflowTerminal(details.workflow.status)) void reloadNotifications()
  }, controller.signal).then(() => {
    if (controller.signal.aborted || workflowTerminal(selected.value?.workflow.status || '')) return
    scheduleReconnect(id)
  }).catch(() => {
    if (!controller.signal.aborted) scheduleReconnect(id)
  })
}

function scheduleReconnect(id: string) {
  streamState.value = 'RECONNECTING'
  reconnectTimer = window.setTimeout(() => watchWorkflow(id), 5000)
}

function stopWatching() {
  eventController?.abort()
  eventController = undefined
  if (reconnectTimer) window.clearTimeout(reconnectTimer)
  reconnectTimer = undefined
  streamState.value = 'IDLE'
}

async function reloadNotifications() {
  try { notifications.value = await workflowApi.listNotifications() } catch { /* keep the last successful snapshot */ }
}

async function reloadAgentRuntime() {
  try { agentRuntimes.value = await workflowApi.listAgentRuntime() } catch { /* retain the last runtime snapshot */ }
}

function handleResearchImported() {
  void researchManager.value?.refresh()
}

async function markRead(item: WorkflowNotification) {
  try {
    const updated = await workflowApi.markNotificationRead(item.id)
    const index = notifications.value.findIndex((candidate) => candidate.id === item.id)
    if (index >= 0) notifications.value[index] = updated
  } catch (cause: any) {
    ElMessage.error(cause.response?.data?.message || '消息状态更新失败')
  }
}

async function openNotification(item: WorkflowNotification) {
  failureNotification.value = item
  failureDetails.value = undefined
  failureError.value = ''
  failureDialog.value = true
  if (item.status === 'UNREAD') void markRead(item)
  if (!item.workflow_id) {
    failureError.value = '这条消息没有关联工作流，无法读取步骤详情。'
    return
  }
  failureLoading.value = true
  try { failureDetails.value = await workflowApi.get(item.workflow_id) }
  catch (cause: any) { failureError.value = cause.response?.data?.message || '工作流失败详情加载失败' }
  finally { failureLoading.value = false }
}

function locateFailedWorkflow() {
  if (!failureDetails.value) return
  applyWorkflow(failureDetails.value)
  activeSection.value = 'workflows'
  failureDialog.value = false
  watchWorkflow(failureDetails.value.workflow.workflow_id)
}

onMounted(() => {
  void load()
  runtimeTimer = window.setInterval(reloadAgentRuntime, 5000)
})
onUnmounted(() => {
  stopWatching()
  if (runtimeTimer) window.clearInterval(runtimeTimer)
})
</script>

<template>
  <AppShell>
    <div class="page-heading">
      <div><span class="eyebrow">MULTI-AGENT WORKFLOW</span><h2>Agent 工作台</h2><p>OpenClaw 负责交互，确定性 Agent 计算指标，Business Agent 生成建议，Report Agent 固化报告。</p></div>
      <div class="heading-actions"><el-button :loading="loading" @click="load">刷新</el-button><el-button type="primary" @click="openLaunchDialog">启动多 Agent 工作流</el-button></div>
    </div>
    <el-alert v-if="error" :title="error" type="error" show-icon />
    <el-alert title="Research Agent 支持受控实时联网；搜索结果必须登记并经人工核验后，才会进入经营分析上下文。" type="info" show-icon />

    <el-tabs v-model="activeSection" class="workflow-hub-tabs">
      <el-tab-pane name="workflows">
        <template #label><span class="workflow-tab-label">工作流<span>{{ workflows.length }}</span></span></template>
        <div v-if="loading" class="loading-panel">正在加载 Agent 与工作流…</div>
        <div v-else class="workflow-layout">
          <aside class="workflow-list">
            <div class="section-title"><h3>工作流记录</h3><span>{{ workflows.length }}</span></div>
            <el-empty v-if="!workflows.length" description="暂无工作流" />
            <button v-for="run in workflows" :key="run.workflow_id" :class="{ active: run.workflow_id === selected?.workflow.workflow_id }" @click="selectWorkflow(run)">
              <span>{{ campaignName(run.campaign_id) }}</span><small>{{ run.created_at.slice(0, 16).replace('T', ' ') }}</small><el-tag :type="workflowStatusType(run.status)" size="small">{{ run.status }}</el-tag>
            </button>
          </aside>
          <main class="workflow-detail">
            <el-empty v-if="!selected" description="启动一个新工作流，或从左侧选择历史记录"><el-button type="primary" @click="openLaunchDialog">启动工作流</el-button></el-empty>
            <template v-else>
              <header class="result-header"><div><span class="eyebrow">{{ selected.workflow.workflow_type }}</span><h3>{{ campaignName(selected.workflow.campaign_id) }}</h3><p>当前步骤：{{ selected.workflow.current_step }} · 实时连接：{{ streamState }}</p></div><el-tag :type="workflowStatusType(selected.workflow.status)" size="large">{{ selected.workflow.status }}</el-tag></header>
              <el-alert v-if="selected.workflow.error_message" :title="selected.workflow.error_message" type="error" show-icon />
              <ol class="workflow-steps" aria-label="Agent 执行步骤">
                <li v-for="step in selected.steps" :key="step.step_id" :class="step.status.toLowerCase()">
                  <div class="step-index">{{ step.sequence_number }}</div>
                  <div><strong>{{ step.agent_name }}</strong><span>{{ step.execution_mode }}</span><p v-if="step.error_message">{{ step.error_message }}</p></div>
                  <el-tag :type="workflowStatusType(step.status)">{{ step.status }}</el-tag>
                </li>
              </ol>
              <div v-if="selected.workflow.business_task_id" class="report-snapshot"><span>BUSINESS TASK</span><strong>{{ selected.workflow.business_task_id }}</strong><p>经营建议和审批仍由现有安全边界控制，不会直接修改广告平台。</p></div>
            </template>
          </main>
        </div>
      </el-tab-pane>

      <el-tab-pane name="notifications">
        <template #label><span class="workflow-tab-label">内部消息<span v-if="unreadCount">{{ unreadCount }}</span></span></template>
        <section class="notification-center" aria-label="OpenClaw 内部消息">
          <header><div><span class="eyebrow">OPENCLAW INBOX</span><h3>内部消息</h3></div><el-badge :value="unreadCount" :hidden="!unreadCount"><el-button @click="reloadNotifications">刷新消息</el-button></el-badge></header>
          <el-empty v-if="!notifications.length" :image-size="54" description="暂无工作流消息" />
          <div v-else class="notification-list">
            <article v-for="item in notifications.slice(0, 6)" :key="item.id" :class="{ unread: item.status === 'UNREAD', failed: item.event_type === 'WORKFLOW_FAILED' }" role="button" tabindex="0" @click="openNotification(item)" @keydown.enter="openNotification(item)">
              <span class="notification-dot" aria-hidden="true"></span>
              <div><strong>{{ item.title }}</strong><p>{{ item.message }}</p><small>{{ item.created_at.slice(0, 16).replace('T', ' ') }} · {{ item.channel }}</small></div>
              <span class="notification-action" :class="{ danger: item.event_type === 'WORKFLOW_FAILED' }">{{ item.event_type === 'WORKFLOW_FAILED' ? '查看失败原因 →' : '查看详情 →' }}</span>
            </article>
          </div>
        </section>
      </el-tab-pane>

      <el-tab-pane name="agents">
        <template #label><span class="workflow-tab-label">Agent 运行<span>{{ agents.length }}</span></span></template>
        <AgentOperationsPanel :agents="agents" :runtimes="agentRuntimes" :campaigns="campaigns" @refresh-runtime="reloadAgentRuntime" @imported="handleResearchImported" />
      </el-tab-pane>

      <el-tab-pane name="sources">
        <template #label><span class="workflow-tab-label">来源治理<span>HTTPS</span></span></template>
        <ResearchSourceManager ref="researchManager" :campaigns="campaigns" />
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="launchDialog" class="workflow-start-dialog" title="启动多 Agent 工作流" width="560px" :close-on-click-modal="!starting" :close-on-press-escape="!starting">
      <div class="workflow-dialog-intro"><span class="eyebrow">OPENCLAW COMMAND</span><strong>启动完整经营分析</strong><p>多个 Agent 将按固定顺序协作；经营建议仍需人工审批，不会直接修改广告平台。</p></div>
      <el-alert v-if="launchError" :title="launchError" type="error" show-icon />
      <div class="workflow-dialog-form">
        <label>广告计划<el-select v-model="campaignID" aria-label="广告计划"><el-option v-for="campaign in campaigns" :key="campaign.id" :label="campaign.name" :value="campaign.id" /></el-select></label>
        <label>分析日期<el-date-picker v-model="analysisDate" aria-label="分析日期" type="date" value-format="YYYY-MM-DD" /></label>
      </div>
      <template #footer><el-button :disabled="starting" @click="launchDialog = false">取消</el-button><el-button type="primary" :loading="starting" :disabled="!canStart" @click="start">启动多 Agent 工作流</el-button></template>
    </el-dialog>

    <WorkflowFailureDialog v-model="failureDialog" :notification="failureNotification" :details="failureDetails" :campaign-name="failureDetails ? campaignName(failureDetails.workflow.campaign_id) : ''" :loading="failureLoading" :error="failureError" @locate="locateFailedWorkflow" />
  </AppShell>
</template>
