<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElDatePicker } from 'element-plus/es/components/date-picker/index'
import 'element-plus/es/components/date-picker/style/css'
import AppShell from '@/components/AppShell.vue'
import { catalogApi } from '@/api/catalog'
import { listImports, uploadImport } from '@/api/imports'
import { configureMMPConnection, listMMPConnections, listMMPSyncRuns, syncMMPConnection } from '@/api/mmp'
import type { MMPProvider } from '@/api/mmp'
import { useAuthStore } from '@/stores/auth'
import type { Game, ImportJob, MMPConnection, MMPSyncRun } from '@/types/catalog'
import { canSyncConnection, connectionHealthLabel, connectionHint, validateSyncRange } from '@/utils/mmp'

const auth = useAuthStore()
const games = ref<Game[]>([])
const jobs = ref<ImportJob[]>([])
const connections = ref<MMPConnection[]>([])
const syncRuns = ref<MMPSyncRun[]>([])
const loading = ref(false)
const uploading = ref(false)
const savingProvider = ref<MMPProvider | null>(null)
const syncingProvider = ref<MMPProvider | null>(null)
const error = ref('')
const success = ref('')
const gameID = ref('')
const importType = ref('ad-metrics')
const source = ref('META')
const file = ref<File | null>(null)
const externalAppIDs = reactive<Record<MMPProvider, string>>({ APPSFLYER: '', ADJUST: '' })
const connectionStatuses = reactive<Record<MMPProvider, 'ACTIVE' | 'DISABLED'>>({ APPSFLYER: 'ACTIVE', ADJUST: 'ACTIVE' })
const syncRange = ref<string[]>([])

const canConfigure = computed(() => auth.hasAnyRole(['ADMIN', 'MANAGER']))
const appsFlyerConnection = computed(() => connectionFor('APPSFLYER'))
const adjustConnection = computed(() => connectionFor('ADJUST'))
const sourceOptions = computed(() => ({ 'ad-metrics': ['META', 'GOOGLE', 'TIKTOK'], 'mmp-metrics': ['APPSFLYER', 'ADJUST'], 'game-revenue': ['GAME', 'INTERNAL'], 'creative-metrics': ['META', 'GOOGLE', 'TIKTOK'] }[importType.value] || []))
const appsFlyerHint = computed(() => connectionHint('APPSFLYER', appsFlyerConnection.value))
const adjustHint = computed(() => connectionHint('ADJUST', adjustConnection.value))

watch(importType, () => { source.value = sourceOptions.value[0] || '' })
watch([gameID, connections], () => {
  for (const provider of ['APPSFLYER', 'ADJUST'] as MMPProvider[]) {
    const connection = connectionFor(provider)
    externalAppIDs[provider] = connection?.external_app_id || ''
    connectionStatuses[provider] = connection?.status || 'ACTIVE'
  }
}, { immediate: true })

function messageOf(value: any) { return value.response?.data?.message || '请求失败，请稍后重试。' }
function localDate(offsetDays: number) { const date = new Date(); date.setDate(date.getDate() + offsetDays); return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}` }
function choose(event: Event) { file.value = (event.target as HTMLInputElement).files?.[0] || null }
function tagType(status: string) { return status === 'SUCCEEDED' || status === 'READY' ? 'success' : status === 'FAILED' || status === 'NOT_CONFIGURED' ? 'danger' : 'warning' }
function shortDate(value?: string) { return value?.slice(0, 10) || '—' }
function connectionFor(provider: MMPProvider) { return connections.value.find((item) => item.game_id === gameID.value && item.provider === provider) }
function providerLabel(provider: MMPProvider) { return provider === 'ADJUST' ? 'Adjust' : 'AppsFlyer' }
function providerIdentifier(provider: MMPProvider) { return provider === 'ADJUST' ? 'Adjust App Token' : 'AppsFlyer App ID' }
function maxRange(provider: MMPProvider) { return provider === 'ADJUST' ? 31 : 7 }
function canSync(provider: MMPProvider) { return canSyncConnection(connectionFor(provider)) && syncingProvider.value === null }

async function load() {
  loading.value = true
  error.value = ''
  try {
    ;[games.value, jobs.value, connections.value, syncRuns.value] = await Promise.all([catalogApi.listGames(), listImports(), listMMPConnections(), listMMPSyncRuns()])
    if (!gameID.value) gameID.value = games.value[0]?.id || ''
  } catch (e) { error.value = messageOf(e) } finally { loading.value = false }
}

async function submit() {
  if (!gameID.value || !file.value) { error.value = '请选择游戏和 CSV/JSON 文件。'; return }
  uploading.value = true; error.value = ''; success.value = ''
  try {
    const job = await uploadImport(importType.value, gameID.value, source.value, file.value)
    success.value = `${job.file_name} 导入完成：新增 ${job.imported_rows} 行，跳过 ${job.skipped_rows} 行。`
    await load()
  } catch (e) { error.value = messageOf(e); await load() } finally { uploading.value = false }
}

async function saveConnection(provider: MMPProvider) {
  if (!gameID.value || !externalAppIDs[provider].trim()) { error.value = `请选择游戏并填写 ${providerIdentifier(provider)}。`; return }
  savingProvider.value = provider; error.value = ''; success.value = ''
  try {
    const saved = await configureMMPConnection(provider, { game_id: gameID.value, external_app_id: externalAppIDs[provider].trim(), status: connectionStatuses[provider] })
    success.value = `${providerLabel(provider)} 映射已保存；当前状态：${connectionHealthLabel(saved.health)}。`
    await load()
  } catch (e) { error.value = messageOf(e) } finally { savingProvider.value = null }
}

async function startSync(provider: MMPProvider) {
  const rangeError = validateSyncRange(syncRange.value, maxRange(provider))
  if (rangeError) { error.value = rangeError; return }
  const connection = connectionFor(provider)
  if (!connection) { error.value = `请先保存 ${providerLabel(provider)} 连接。`; return }
  syncingProvider.value = provider; error.value = ''; success.value = ''
  try {
    const run = await syncMMPConnection(connection.id, syncRange.value[0], syncRange.value[1])
    success.value = run.status === 'PROCESSING'
      ? `${providerLabel(provider)} 的相同日期同步正在执行，请稍后查看运行记录。`
      : `${providerLabel(provider)} 同步完成：${run.source_rows} 条源记录聚合为 ${run.normalized_rows} 行指标。${run.warning_message || ''}`
    await load()
  } catch (e) { error.value = messageOf(e); await load() } finally { syncingProvider.value = null }
}

onMounted(() => { syncRange.value = [localDate(-2), localDate(-1)]; void load() })
</script>

<template><AppShell>
  <div class="page-heading"><div><span class="eyebrow">DATA INGESTION · PHASE 12</span><h2>数据导入</h2><p>通过 AppsFlyer、Adjust、标准批次或经过字段校验的 CSV / JSON 接入数据。</p></div></div>
  <el-alert v-if="error" :title="error" type="error" show-icon closable @close="error = ''" />
  <el-alert v-if="success" :title="success" type="success" show-icon closable @close="success = ''" />

  <section class="connector-panel" aria-labelledby="appsflyer-heading">
    <div class="connector-heading"><div><span class="eyebrow">READ-ONLY CONNECTOR</span><h3 id="appsflyer-heading">AppsFlyer</h3><p>{{ appsFlyerHint }}</p></div><el-tag :type="tagType(appsFlyerConnection?.health || 'NOT_CONFIGURED')">{{ connectionHealthLabel(appsFlyerConnection?.health || 'NOT_CONFIGURED') }}</el-tag></div>
    <div class="connector-fields">
      <label>游戏<el-select v-model="gameID" aria-label="AppsFlyer 游戏"><el-option v-for="game in games" :key="game.id" :label="game.name" :value="game.id" /></el-select></label>
      <label>AppsFlyer App ID<el-input v-model="externalAppIDs.APPSFLYER" :disabled="!canConfigure" placeholder="com.example.game / id123456789" aria-label="AppsFlyer App ID" /></label>
      <label>连接状态<el-select v-model="connectionStatuses.APPSFLYER" :disabled="!canConfigure" aria-label="AppsFlyer 连接状态"><el-option label="ACTIVE" value="ACTIVE" /><el-option label="DISABLED" value="DISABLED" /></el-select></label>
      <el-button v-if="canConfigure" :loading="savingProvider === 'APPSFLYER'" @click="saveConnection('APPSFLYER')">保存映射</el-button>
    </div>
    <div class="sync-strip"><label>同步日期（最多 7 天）<el-date-picker v-model="syncRange" type="daterange" value-format="YYYY-MM-DD" range-separator="至" start-placeholder="开始日期" end-placeholder="结束日期" aria-label="AppsFlyer 同步日期" /></label><span>Bearer Token 仅由服务端环境变量读取</span><el-button type="primary" :loading="syncingProvider === 'APPSFLYER'" :disabled="!canSync('APPSFLYER')" @click="startSync('APPSFLYER')">同步 AppsFlyer</el-button></div>
  </section>

  <section class="connector-panel" aria-labelledby="adjust-heading">
    <div class="connector-heading"><div><span class="eyebrow">READ-ONLY CONNECTOR</span><h3 id="adjust-heading">Adjust</h3><p>{{ adjustHint }}</p></div><el-tag :type="tagType(adjustConnection?.health || 'NOT_CONFIGURED')">{{ connectionHealthLabel(adjustConnection?.health || 'NOT_CONFIGURED') }}</el-tag></div>
    <div class="connector-fields">
      <label>游戏<el-select v-model="gameID" aria-label="Adjust 游戏"><el-option v-for="game in games" :key="game.id" :label="game.name" :value="game.id" /></el-select></label>
      <label>Adjust App Token<el-input v-model="externalAppIDs.ADJUST" :disabled="!canConfigure" placeholder="Adjust dashboard app token" aria-label="Adjust App Token" /></label>
      <label>连接状态<el-select v-model="connectionStatuses.ADJUST" :disabled="!canConfigure" aria-label="Adjust 连接状态"><el-option label="ACTIVE" value="ACTIVE" /><el-option label="DISABLED" value="DISABLED" /></el-select></label>
      <el-button v-if="canConfigure" :loading="savingProvider === 'ADJUST'" @click="saveConnection('ADJUST')">保存映射</el-button>
    </div>
    <div class="sync-strip"><label>同步日期（最多 31 天）<el-date-picker v-model="syncRange" type="daterange" value-format="YYYY-MM-DD" range-separator="至" start-placeholder="开始日期" end-placeholder="结束日期" aria-label="Adjust 同步日期" /></label><span>API Token 与事件指标映射仅由服务端环境变量读取</span><el-button type="primary" :loading="syncingProvider === 'ADJUST'" :disabled="!canSync('ADJUST')" @click="startSync('ADJUST')">同步 Adjust</el-button></div>
  </section>

  <div class="section-title"><h3>文件导入</h3><span>补录与离线数据</span></div>
  <section class="import-panel">
    <div class="import-fields"><label>游戏<el-select v-model="gameID" aria-label="游戏"><el-option v-for="game in games" :key="game.id" :label="game.name" :value="game.id" /></el-select></label><label>数据类型<el-select v-model="importType" aria-label="数据类型"><el-option label="广告投放数据" value="ad-metrics" /><el-option label="MMP 归因数据" value="mmp-metrics" /><el-option label="游戏收入数据" value="game-revenue" /><el-option label="素材表现数据" value="creative-metrics" /></el-select></label><label>来源<el-select v-model="source" aria-label="来源"><el-option v-for="item in sourceOptions" :key="item" :label="item" :value="item" /></el-select></label></div>
    <label class="file-drop"><input type="file" accept=".csv,.json,text/csv,application/json" @change="choose" /><strong>{{ file?.name || '选择 CSV 或 JSON 文件' }}</strong><span>原始文件不会直接发送给模型</span></label>
    <el-button type="primary" :loading="uploading" :disabled="!file" @click="submit">校验并导入</el-button>
  </section>

  <div class="section-title"><h3>MMP 同步记录</h3><span>{{ syncRuns.length }} 个任务</span></div>
  <div v-if="loading" class="loading-panel">正在加载同步记录…</div><el-empty v-else-if="!syncRuns.length" description="暂无 MMP 同步" /><el-table v-else :data="syncRuns"><el-table-column prop="provider" label="平台" /><el-table-column label="日期范围" min-width="190"><template #default="s">{{ shortDate(s.row.period_start) }} → {{ shortDate(s.row.period_end) }}</template></el-table-column><el-table-column label="源记录 / 指标"><template #default="s">{{ s.row.source_rows }} / {{ s.row.normalized_rows }}</template></el-table-column><el-table-column prop="skipped_rows" label="跳过" /><el-table-column label="状态"><template #default="s"><el-tag :type="tagType(s.row.status)">{{ s.row.status }}</el-tag></template></el-table-column><el-table-column prop="warning_message" label="警告" min-width="220" /><el-table-column prop="error_message" label="错误" min-width="220" /></el-table>

  <div class="section-title"><h3>最近导入</h3><span>{{ jobs.length }} 个任务</span></div>
  <div v-if="loading" class="loading-panel">正在加载导入记录…</div><el-empty v-else-if="!jobs.length" description="暂无导入任务" /><el-table v-else :data="jobs"><el-table-column prop="file_name" label="文件" min-width="180" /><el-table-column prop="import_type" label="类型" /><el-table-column prop="source" label="来源" /><el-table-column label="起始日期"><template #default="s">{{ shortDate(s.row.period_start) }}</template></el-table-column><el-table-column label="结果"><template #default="s">{{ s.row.imported_rows }} 新增 / {{ s.row.skipped_rows }} 跳过</template></el-table-column><el-table-column label="状态"><template #default="s"><el-tag :type="tagType(s.row.status)">{{ s.row.status }}</el-tag></template></el-table-column><el-table-column prop="error_message" label="错误" min-width="220" /></el-table>
</AppShell></template>
