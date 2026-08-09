<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { researchApi } from '@/api/research'
import { useAuthStore } from '@/stores/auth'
import type { Campaign } from '@/types/catalog'
import type { ResearchCategory, ResearchSource, WebSearchCapability, WebSearchResult } from '@/types/research'
import type { AgentCatalogItem, AgentRuntime } from '@/types/workflow'
import { agentPresentation, elapsedLabel, runtimeStatusLabel, taskAction, toolMetadata } from '@/utils/agent'

const props = defineProps<{ agents: AgentCatalogItem[]; runtimes: AgentRuntime[]; campaigns: Campaign[] }>()
const emit = defineEmits<{ imported: [source: ResearchSource]; refreshRuntime: [] }>()
const auth = useAuthStore()
const filter = ref<'ALL' | 'RUNNING' | 'IDLE' | 'FAILED' | 'WEB'>('ALL')
const selectedName = ref('')
const drawerOpen = ref(false)
const drawerTab = ref('overview')
const now = ref(Date.now())
const capability = ref<WebSearchCapability>({ configured: false, provider: 'disabled', import_enabled: false, max_results: 8, details: [] })
const searching = ref(false)
const importingURL = ref('')
const searchError = ref('')
const searchResults = ref<WebSearchResult[]>([])
const searchForm = reactive<{ query: string; campaign_id: string; category: ResearchCategory; count: number; freshness: '' | 'pd' | 'pw' | 'pm' | 'py' }>({ query: '', campaign_id: '', category: 'MARKET', count: 6, freshness: 'pw' })
let clock: number | undefined
const filters: Array<{ key: 'ALL' | 'RUNNING' | 'IDLE' | 'FAILED' | 'WEB'; label: string }> = [{ key: 'ALL', label: '全部' }, { key: 'RUNNING', label: '处理中' }, { key: 'IDLE', label: '空闲' }, { key: 'FAILED', label: '异常' }, { key: 'WEB', label: '可联网' }]

const runtimeMap = computed(() => new Map(props.runtimes.map((item) => [item.agent_name, item])))
const selectedAgent = computed(() => props.agents.find((item) => item.definition.spec.name === selectedName.value))
const selectedRuntime = computed(() => selectedName.value ? runtimeMap.value.get(selectedName.value) : undefined)
const runningCount = computed(() => props.runtimes.filter((item) => item.runtime_status === 'RUNNING').length)
const queuedCount = computed(() => props.runtimes.filter((item) => item.runtime_status === 'QUEUED').length)
const failedCount = computed(() => props.runtimes.filter((item) => item.runtime_status === 'FAILED').length)
const canResearch = computed(() => auth.hasAnyRole(['ADMIN', 'MANAGER', 'ANALYST']))
const filteredAgents = computed(() => props.agents.filter((item) => {
  const name = item.definition.spec.name
  const runtime = runtimeMap.value.get(name)?.runtime_status || 'IDLE'
  if (filter.value === 'ALL') return true
  if (filter.value === 'WEB') return name === 'research-agent'
  return runtime === filter.value
}))
const selectedCampaign = computed(() => props.campaigns.find((item) => item.id === searchForm.campaign_id))

function runtimeFor(name: string): AgentRuntime {
  return runtimeMap.value.get(name) || { agent_name: name, runtime_status: 'IDLE', active_tasks: [] }
}

function statusType(status: AgentRuntime['runtime_status']): '' | 'success' | 'warning' | 'danger' | 'info' {
  if (status === 'RUNNING') return 'success'
  if (status === 'QUEUED') return 'warning'
  if (status === 'FAILED') return 'danger'
  return 'info'
}

function healthType(status: string): '' | 'success' | 'warning' | 'danger' | 'info' {
  if (status === 'UP' || status === 'READY') return 'success'
  if (status === 'DOWN') return 'danger'
  return 'warning'
}

function connectivity(name: string) {
  if (name === 'research-agent') return capability.value.configured ? `实时联网 · ${capability.value.provider}` : '实时联网待配置'
  if (name === 'attribution-agent') return 'AppsFlyer / 导入 MMP'
  if (name === 'openclaw-agent') return '内部 API'
  return '内部数据'
}

function openAgent(name: string, tab = 'overview') {
  selectedName.value = name
  drawerOpen.value = true
  drawerTab.value = tab
  if (name === 'research-agent' && tab === 'web') void loadCapability()
}

async function loadCapability() {
  try { capability.value = await researchApi.webCapability() }
  catch { capability.value = { configured: false, provider: 'disabled', import_enabled: false, max_results: 8, details: ['联网能力状态加载失败'] } }
}

async function searchWeb() {
  if (!searchForm.query.trim() || !capability.value.configured) return
  searching.value = true
  searchError.value = ''
  searchResults.value = []
  try {
    const response = await researchApi.searchWeb({
      query: searchForm.query.trim(), category: searchForm.category, count: searchForm.count, freshness: searchForm.freshness,
      game_id: selectedCampaign.value?.game_id, campaign_id: selectedCampaign.value?.id, country: selectedCampaign.value?.country,
    })
    capability.value = response.capability
    searchResults.value = response.results
    if (!response.results.length) ElMessage.info('没有找到符合条件的公开网页，可调整关键词或时间范围')
  } catch (cause: any) {
    searchError.value = cause.response?.data?.message || '实时联网查询失败'
  } finally {
    searching.value = false
  }
}

async function importResult(result: WebSearchResult) {
  if (!capability.value.import_enabled) return
  importingURL.value = result.url
  try {
    const source = await researchApi.importWebResult({ query: searchForm.query.trim(), category: searchForm.category, game_id: selectedCampaign.value?.game_id, campaign_id: selectedCampaign.value?.id, result })
    emit('imported', source)
    ElMessage.success('已加入待核验来源，核验通过后才能进入 Agent 分析')
  } catch (cause: any) {
    ElMessage.error(cause.response?.data?.message || '联网结果登记失败')
  } finally {
    importingURL.value = ''
  }
}

onMounted(() => {
  void loadCapability()
  clock = window.setInterval(() => { now.value = Date.now() }, 1000)
})
onUnmounted(() => { if (clock) window.clearInterval(clock) })
</script>

<template>
  <section class="agent-operations" aria-labelledby="agent-operations-title">
    <header class="agent-operations-heading">
      <div><span class="eyebrow">AGENT OPERATIONS</span><h3 id="agent-operations-title">Agent 运行中心</h3><p>查看每个 Agent 正在处理的任务、工具权限和外部连接状态。</p></div>
      <el-button @click="emit('refreshRuntime')">刷新运行状态</el-button>
    </header>

    <div class="agent-signal-board" aria-label="Agent 运行概览">
      <div><span>已注册</span><strong>{{ agents.length }}</strong><small>个 Agent</small></div>
      <div class="running"><span>处理中</span><strong>{{ runningCount }}</strong><small>实时任务</small></div>
      <div><span>排队中</span><strong>{{ queuedCount }}</strong><small>等待步骤</small></div>
      <div :class="{ danger: failedCount }"><span>最近异常</span><strong>{{ failedCount }}</strong><small>需要关注</small></div>
      <div class="connectivity"><span>联网研究</span><strong>{{ capability.configured ? 'READY' : 'SETUP' }}</strong><small>{{ capability.configured ? capability.provider : '等待 API Key' }}</small></div>
    </div>

    <div class="agent-filters" aria-label="筛选 Agent">
      <button v-for="item in filters" :key="item.key" :class="{ active: filter === item.key }" @click="filter = item.key">{{ item.label }}</button>
    </div>

    <div class="agent-operations-grid">
      <article v-for="item in filteredAgents" :key="item.definition.spec.name" class="agent-operation-card" :class="[`is-${runtimeFor(item.definition.spec.name).runtime_status.toLowerCase()}`]" tabindex="0" @click="openAgent(item.definition.spec.name)" @keydown.enter="openAgent(item.definition.spec.name)">
        <div class="agent-card-signal" aria-hidden="true"><i></i></div>
        <header>
          <div><span>{{ agentPresentation(item.definition.spec.name).role }}</span><small>{{ item.definition.spec.execution_mode }}</small></div>
          <el-tag :type="statusType(runtimeFor(item.definition.spec.name).runtime_status)" effect="light">{{ runtimeStatusLabel(runtimeFor(item.definition.spec.name).runtime_status) }}</el-tag>
        </header>
        <div class="agent-card-title"><h4>{{ agentPresentation(item.definition.spec.name).displayName }}</h4><code>{{ item.definition.spec.name }}</code></div>
        <p>{{ item.definition.spec.description }}</p>

        <div v-if="runtimeFor(item.definition.spec.name).active_tasks[0]" class="agent-current-task">
          <span>正在处理</span>
          <strong>{{ runtimeFor(item.definition.spec.name).active_tasks[0].campaign_name }}</strong>
          <p>{{ taskAction(runtimeFor(item.definition.spec.name).active_tasks[0]) }}</p>
          <small>{{ runtimeFor(item.definition.spec.name).active_tasks[0].status }} · {{ elapsedLabel(runtimeFor(item.definition.spec.name).active_tasks[0].started_at, now) }}</small>
        </div>
        <div v-else class="agent-idle-state"><span>当前无任务</span><small v-if="runtimeFor(item.definition.spec.name).last_task">最近运行：{{ runtimeFor(item.definition.spec.name).last_task?.updated_at.slice(0, 16).replace('T', ' ') }}</small><small v-else>等待工作流调度</small></div>

        <div class="agent-tool-preview"><span v-for="tool in item.definition.spec.tools.slice(0, 3)" :key="tool">{{ toolMetadata(tool).label }}</span><span v-if="item.definition.spec.tools.length > 3">+{{ item.definition.spec.tools.length - 3 }}</span></div>
        <footer><span><i></i>{{ connectivity(item.definition.spec.name) }}</span><button type="button" @click.stop="openAgent(item.definition.spec.name)">查看详情 →</button></footer>
      </article>
    </div>
    <el-empty v-if="!filteredAgents.length" :image-size="56" description="当前筛选条件下没有 Agent" />

    <el-drawer v-model="drawerOpen" class="agent-detail-drawer" size="640px" direction="rtl" :with-header="false">
      <template v-if="selectedAgent">
        <header class="agent-drawer-heading">
          <div><span class="eyebrow">{{ agentPresentation(selectedName).role }}</span><h3>{{ agentPresentation(selectedName).displayName }}</h3><code>{{ selectedName }} · v{{ selectedAgent.definition.spec.version }}</code></div>
          <div><el-tag :type="statusType(selectedRuntime?.runtime_status || 'IDLE')">{{ runtimeStatusLabel(selectedRuntime?.runtime_status || 'IDLE') }}</el-tag><el-tag :type="healthType(selectedAgent.health?.status || selectedAgent.definition.availability)" effect="plain">服务 {{ selectedAgent.health?.status || selectedAgent.definition.availability }}</el-tag></div>
        </header>

        <el-tabs v-model="drawerTab" class="agent-drawer-tabs">
          <el-tab-pane label="概览" name="overview">
            <section class="agent-drawer-section agent-intro"><h4>这个 Agent 做什么</h4><p>{{ selectedAgent.definition.spec.description }}</p><div class="agent-facts"><div><span>执行方式</span><strong>{{ selectedAgent.definition.spec.execution_mode }}</strong></div><div><span>Provider</span><strong>{{ selectedAgent.health?.provider || 'internal' }}</strong></div><div><span>最大步骤</span><strong>{{ selectedAgent.definition.spec.max_steps }}</strong></div><div><span>超时</span><strong>{{ Math.round(selectedAgent.definition.spec.timeout / 1_000_000_000) }} 秒</strong></div></div></section>
            <section class="agent-drawer-section"><h4>连接与安全边界</h4><div class="connection-status"><div><span>数据连接</span><strong>{{ connectivity(selectedName) }}</strong></div><div><span>外部写入</span><strong>{{ selectedName === 'openclaw-agent' ? '禁止广告平台执行' : '仅内部受控写入' }}</strong></div><div v-if="selectedName === 'research-agent'"><span>来源准入</span><strong>联网结果需人工核验</strong></div></div></section>
            <section v-if="selectedAgent.health?.details?.length" class="agent-drawer-section"><h4>健康检查</h4><div class="detail-chips"><span v-for="detail in selectedAgent.health.details" :key="detail">{{ detail }}</span></div></section>
          </el-tab-pane>

          <el-tab-pane :label="`任务 ${selectedRuntime?.active_tasks.length || 0}`" name="tasks">
            <section class="agent-drawer-section"><h4>当前任务</h4><div v-if="selectedRuntime?.active_tasks.length" class="agent-task-list"><article v-for="task in selectedRuntime.active_tasks" :key="task.workflow_id"><header><strong>{{ task.campaign_name }}</strong><el-tag :type="statusType(selectedRuntime.runtime_status)" size="small">{{ task.status }}</el-tag></header><p>{{ taskAction(task) }}</p><small>工作流 {{ task.workflow_id }} · {{ elapsedLabel(task.started_at, now) }}</small></article></div><el-empty v-else :image-size="54" description="当前没有正在处理的任务" /></section>
            <section v-if="selectedRuntime?.last_task" class="agent-drawer-section"><h4>最近运行</h4><div class="last-agent-task"><span>{{ selectedRuntime.last_task.campaign_name }}</span><strong>{{ selectedRuntime.last_task.status }}</strong><small>{{ selectedRuntime.last_task.updated_at.slice(0, 16).replace('T', ' ') }}</small></div></section>
          </el-tab-pane>

          <el-tab-pane :label="`工具 ${selectedAgent.definition.spec.tools.length}`" name="tools">
            <section class="agent-drawer-section"><h4>受限工具</h4><div class="agent-tool-list"><article v-for="name in selectedAgent.definition.spec.tools" :key="name"><div><strong>{{ toolMetadata(name).label }}</strong><code>{{ name }}</code></div><p>{{ toolMetadata(name).description }}</p><el-tag size="small" effect="plain">{{ toolMetadata(name).access }}</el-tag></article></div></section>
            <section class="agent-drawer-section"><h4>权限</h4><div class="detail-chips"><span v-for="permission in selectedAgent.definition.spec.permissions" :key="permission">{{ permission }}</span></div></section>
          </el-tab-pane>

          <el-tab-pane v-if="selectedName === 'research-agent'" label="联网研究" name="web">
            <section class="agent-drawer-section web-research-console">
              <div class="web-research-status"><div><span class="web-pulse" :class="{ online: capability.configured }"></span><div><strong>{{ capability.configured ? '实时联网已就绪' : '实时联网待配置' }}</strong><p>{{ capability.configured ? `由 ${capability.provider} 提供最新公开网页结果` : '配置服务端 API Key 后即可发起实时查询' }}</p></div></div><el-tag :type="capability.configured ? 'success' : 'warning'">{{ capability.configured ? 'LIVE WEB' : 'NOT CONFIGURED' }}</el-tag></div>
              <el-alert v-if="!capability.configured" title="需要在服务端配置 GAI_WEB_SEARCH_API_KEY；密钥不会发送到浏览器。" type="warning" show-icon />
              <el-alert v-else-if="!capability.import_enabled" title="实时搜索可用，但结果入库未启用。确认供应商计划包含存储权后，可开启 GAI_WEB_SEARCH_IMPORT_ENABLED。" type="info" show-icon />
              <el-alert v-if="searchError" :title="searchError" type="error" show-icon />
              <div class="web-search-form">
                <label class="wide">查询内容<el-input v-model="searchForm.query" type="textarea" :rows="3" maxlength="400" show-word-limit placeholder="例如：日本手游广告市场 2026 最新政策与平台变化" :disabled="!capability.configured || !canResearch" @keydown.meta.enter="searchWeb" /></label>
                <label>关联计划<el-select v-model="searchForm.campaign_id" clearable :disabled="!capability.configured"><el-option v-for="campaign in campaigns" :key="campaign.id" :label="campaign.name" :value="campaign.id" /></el-select></label>
                <label>资料类型<el-select v-model="searchForm.category" :disabled="!capability.configured"><el-option label="市场" value="MARKET"/><el-option label="政策" value="POLICY"/><el-option label="竞品" value="COMPETITOR"/></el-select></label>
                <label>时间范围<el-select v-model="searchForm.freshness" :disabled="!capability.configured"><el-option label="不限" value=""/><el-option label="24 小时" value="pd"/><el-option label="7 天" value="pw"/><el-option label="31 天" value="pm"/><el-option label="1 年" value="py"/></el-select></label>
                <div class="web-search-submit"><span>搜索结果不会自动进入经营分析。</span><el-button type="primary" :loading="searching" :disabled="!capability.configured || !canResearch || !searchForm.query.trim()" @click="searchWeb">实时联网查询</el-button></div>
              </div>
            </section>

            <section v-if="searchResults.length" class="agent-drawer-section"><div class="web-result-heading"><h4>公开网页结果</h4><span>{{ searchResults.length }} 条 · 需人工核验</span></div><div class="web-result-list"><article v-for="result in searchResults" :key="result.url"><div><a :href="result.url" target="_blank" rel="noopener noreferrer">{{ result.title }}</a><p>{{ result.description }}</p><small>{{ result.publisher }}<template v-if="result.published_at"> · {{ result.published_at }}</template><template v-if="result.language"> · {{ result.language }}</template></small></div><el-button type="primary" plain :loading="importingURL === result.url" :disabled="!capability.import_enabled" @click="importResult(result)">加入待核验来源</el-button></article></div></section>
          </el-tab-pane>
        </el-tabs>
      </template>
    </el-drawer>
  </section>
</template>
