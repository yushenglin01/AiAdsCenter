<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElDatePicker } from 'element-plus/es/components/date-picker/index'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import 'element-plus/es/components/date-picker/style/css'
import { researchApi } from '@/api/research'
import { useAuthStore } from '@/stores/auth'
import type { Campaign } from '@/types/catalog'
import type { ResearchCategory, ResearchSchedule, ResearchScheduleInput, ResearchScheduleRun, ResearchSource, ResearchSourceInput } from '@/types/research'
import { validateResearchSourceInput } from '@/utils/research'

const props = defineProps<{ campaigns: Campaign[] }>()
const auth = useAuthStore()
const rows = ref<ResearchSource[]>([])
const schedules = ref<ResearchSchedule[]>([])
const scheduleRuns = ref<ResearchScheduleRun[]>([])
const loading = ref(true)
const saving = ref(false)
const reviewingID = ref('')
const savingSchedule = ref(false)
const updatingScheduleID = ref('')
const editingScheduleID = ref('')
const error = ref('')
const activePane = ref<'sources' | 'create' | 'schedules' | 'schedule-form'>('sources')
const form = reactive<ResearchSourceInput>({ category: 'MARKET', title: '', summary: '', source_url: '', publisher: '', published_at: new Date().toISOString().slice(0, 10) })
const scheduleForm = reactive<ResearchScheduleInput>({ name: '', category: 'MARKET', query: '', country: '', search_lang: '', freshness: 'pw', result_count: 5, interval_minutes: 1440, enabled: true })

const canCreate = computed(() => auth.hasAnyRole(['ADMIN', 'MANAGER', 'ANALYST']))
const canReview = computed(() => auth.hasAnyRole(['ADMIN', 'MANAGER']))
const canSchedule = computed(() => auth.hasAnyRole(['ADMIN', 'MANAGER']))
const pendingCount = computed(() => rows.value.filter((item) => item.status === 'PENDING').length)
const enabledScheduleCount = computed(() => schedules.value.filter((item) => item.enabled).length)

async function load() {
  loading.value = true
  error.value = ''
  try { [rows.value, schedules.value, scheduleRuns.value] = await Promise.all([researchApi.list(), researchApi.listSchedules(), researchApi.listScheduleRuns()]) }
  catch (cause: any) { error.value = cause.response?.data?.message || '研究来源加载失败' }
  finally { loading.value = false }
}

function schedulePayload(item: ResearchSchedule): ResearchScheduleInput {
  return { name: item.name, game_id: item.game_id, campaign_id: item.campaign_id, category: item.category, query: item.query, country: item.country || '', search_lang: item.search_lang || '', freshness: item.freshness || '', result_count: item.result_count, interval_minutes: item.interval_minutes, enabled: item.enabled }
}

function resetScheduleForm() {
  editingScheduleID.value = ''
  Object.assign(scheduleForm, { name: '', game_id: undefined, campaign_id: undefined, category: 'MARKET' as ResearchCategory, query: '', country: '', search_lang: '', freshness: 'pw', result_count: 5, interval_minutes: 1440, enabled: true })
}

function openScheduleForm(item?: ResearchSchedule) {
  resetScheduleForm()
  if (item) {
    editingScheduleID.value = item.id
    Object.assign(scheduleForm, schedulePayload(item))
  }
  activePane.value = 'schedule-form'
}

async function saveSchedule() {
  if (scheduleForm.name.trim().length < 2 || !scheduleForm.query.trim() || scheduleForm.result_count < 1 || scheduleForm.interval_minutes < 60 || scheduleForm.interval_minutes > 10080) {
    ElMessage.warning('请完整填写任务名称、查询词、结果数和 60–10080 分钟运行间隔')
    return
  }
  savingSchedule.value = true
  try {
    const campaign = props.campaigns.find((item) => item.id === scheduleForm.campaign_id)
    scheduleForm.game_id = campaign?.game_id
    const saved = editingScheduleID.value
      ? await researchApi.updateSchedule(editingScheduleID.value, { ...scheduleForm })
      : await researchApi.createSchedule({ ...scheduleForm })
    const index = schedules.value.findIndex((item) => item.id === saved.id)
    if (index >= 0) schedules.value[index] = saved
    else schedules.value.unshift(saved)
    activePane.value = 'schedules'
    resetScheduleForm()
    ElMessage.success(saved.enabled ? '自动研究任务已保存并进入调度' : '自动研究任务已保存为停用状态')
  } catch (cause: any) { ElMessage.error(cause.response?.data?.message || '自动研究任务保存失败') }
  finally { savingSchedule.value = false }
}

async function toggleSchedule(item: ResearchSchedule, enabled: boolean) {
  updatingScheduleID.value = item.id
  try {
    const updated = await researchApi.updateSchedule(item.id, { ...schedulePayload(item), enabled })
    const index = schedules.value.findIndex((candidate) => candidate.id === item.id)
    if (index >= 0) schedules.value[index] = updated
    ElMessage.success(enabled ? '自动研究任务已启用' : '自动研究任务已停用')
  } catch (cause: any) { ElMessage.error(cause.response?.data?.message || '任务状态更新失败') }
  finally { updatingScheduleID.value = '' }
}

function latestRun(scheduleID: string) {
  return scheduleRuns.value.find((run) => run.schedule_id === scheduleID)
}

function intervalLabel(minutes: number) {
  if (minutes % 1440 === 0) return `${minutes / 1440} 天`
  if (minutes % 60 === 0) return `${minutes / 60} 小时`
  return `${minutes} 分钟`
}

function localTime(value?: string) {
  return value ? value.slice(0, 16).replace('T', ' ') : '—'
}

async function create() {
  if (!validateResearchSourceInput(form)) {
    ElMessage.warning('请完整填写标题、摘要、发布方、无凭证 HTTPS 来源和发布日期')
    return
  }
  saving.value = true
  try {
    const campaign = props.campaigns.find((item) => item.id === form.campaign_id)
    form.game_id = campaign?.game_id
    const created = await researchApi.create({ ...form })
    rows.value = [created, ...rows.value.filter((item) => item.id !== created.id)]
    Object.assign(form, { category: 'MARKET' as ResearchCategory, title: '', summary: '', source_url: '', publisher: '', published_at: new Date().toISOString().slice(0, 10), game_id: undefined, campaign_id: undefined })
    activePane.value = 'sources'
    ElMessage.success('来源已登记，核验通过后才会进入 Research Agent')
  } catch (cause: any) { ElMessage.error(cause.response?.data?.message || '来源登记失败') }
  finally { saving.value = false }
}

function replaceRow(updated: ResearchSource) {
  const index = rows.value.findIndex((item) => item.id === updated.id)
  if (index >= 0) rows.value[index] = updated
}

async function verify(item: ResearchSource) {
  reviewingID.value = item.id
  try { replaceRow(await researchApi.verify(item.id)); ElMessage.success('来源已核验，可供 Research Agent 使用') }
  catch (cause: any) { ElMessage.error(cause.response?.data?.message || '来源核验失败') }
  finally { reviewingID.value = '' }
}

async function reject(item: ResearchSource) {
  try {
    const { value } = await ElMessageBox.prompt('请填写驳回原因', '驳回研究来源', { inputValidator: (text) => Boolean(text?.trim()) || '驳回原因不能为空', confirmButtonText: '确认驳回', cancelButtonText: '取消' })
    reviewingID.value = item.id
    replaceRow(await researchApi.reject(item.id, value.trim()))
    ElMessage.success('来源已驳回')
  } catch (cause: any) {
    if (cause !== 'cancel' && cause !== 'close') ElMessage.error(cause.response?.data?.message || '来源驳回失败')
  } finally { reviewingID.value = '' }
}

function statusType(status: ResearchSource['status']) {
  if (status === 'VERIFIED') return 'success'
  if (status === 'REJECTED') return 'danger'
  return 'warning'
}

onMounted(load)
defineExpose({ load })
</script>

<template>
  <section class="research-dialog-panel" aria-label="Research Agent 来源管理">
    <el-alert v-if="error" :title="error" type="error" show-icon />
    <el-tabs v-model="activePane" class="research-management-tabs">
      <el-tab-pane name="sources">
        <template #label><span class="research-tab-label">来源列表 <i>{{ rows.length }}</i></span></template>
        <div class="research-list-toolbar">
          <div><strong>已登记资料</strong><p>只有核验通过的来源才能进入 Research Agent 分析上下文。</p></div>
          <div class="research-actions"><el-tag v-if="pendingCount" type="warning">{{ pendingCount }} 条待核验</el-tag><el-button :loading="loading" @click="load">刷新</el-button><el-button v-if="canCreate" type="primary" @click="activePane = 'create'">登记新来源</el-button></div>
        </div>
        <div v-if="loading" class="research-loading">正在加载来源…</div>
        <el-empty v-else-if="!rows.length" :image-size="60" description="暂无研究来源；登记公开且可核验的资料后，会显示在这里"><el-button v-if="canCreate" type="primary" @click="activePane = 'create'">登记第一个来源</el-button></el-empty>
        <div v-else class="research-list">
          <article v-for="item in rows" :key="item.id">
            <div class="source-main"><div><el-tag size="small" :type="statusType(item.status)">{{ item.status }}</el-tag><span>{{ item.category }}</span><el-tag v-if="item.discovery_method !== 'MANUAL'" size="small" type="primary" effect="plain">{{ item.discovery_method === 'SCHEDULED_WEB_SEARCH' ? '自动联网发现' : '实时联网发现' }}</el-tag></div><a :href="item.source_url" target="_blank" rel="noopener noreferrer">{{ item.title }}</a><p>{{ item.summary }}</p><small>{{ item.publisher }} · {{ item.published_at.slice(0, 10) }} · SHA-256 {{ item.content_hash.slice(0, 12) }}…</small><small v-if="item.discovery_method !== 'MANUAL'">发现来源：{{ item.discovery_provider }} · {{ item.discovered_at?.slice(0, 16).replace('T', ' ') }}<template v-if="item.discovery_schedule_id"> · 任务 {{ item.discovery_schedule_id.slice(0, 8) }}</template></small><small v-if="item.review_comment">审核意见：{{ item.review_comment }}</small></div>
            <div v-if="canReview && item.status === 'PENDING'" class="review-actions"><el-button type="success" plain :loading="reviewingID === item.id" @click="verify(item)">核验通过</el-button><el-button type="danger" plain :disabled="reviewingID === item.id" @click="reject(item)">驳回</el-button></div>
          </article>
        </div>
      </el-tab-pane>

      <el-tab-pane name="schedules">
        <template #label><span class="research-tab-label">自动发现 <i>{{ enabledScheduleCount }}/{{ schedules.length }}</i></span></template>
        <div class="research-list-toolbar">
          <div><strong>Research 定时任务</strong><p>Worker 按数据库租约执行；新来源只进入待核验区，不会自动影响分析。</p></div>
          <div class="research-actions"><el-button :loading="loading" @click="load">刷新</el-button><el-button v-if="canSchedule" type="primary" @click="openScheduleForm()">新建任务</el-button></div>
        </div>
        <el-empty v-if="!loading && !schedules.length" :image-size="60" description="暂无自动发现任务"><el-button v-if="canSchedule" type="primary" @click="openScheduleForm()">新建第一个任务</el-button></el-empty>
        <div v-else class="schedule-list">
          <article v-for="item in schedules" :key="item.id">
            <header><div><el-tag :type="item.enabled ? 'success' : 'info'" size="small">{{ item.enabled ? 'ENABLED' : 'DISABLED' }}</el-tag><span>{{ item.category }}</span><strong>{{ item.name }}</strong></div><el-switch v-if="canSchedule" :model-value="item.enabled" :loading="updatingScheduleID === item.id" aria-label="启停自动研究任务" @change="toggleSchedule(item, Boolean($event))" /></header>
            <p>{{ item.query }}</p>
            <div class="schedule-meta"><span>每 {{ intervalLabel(item.interval_minutes) }}</span><span>单次 {{ item.result_count }} 条</span><span>{{ item.country || '全球' }} / {{ item.search_lang || '自动语言' }}</span><span>下次 {{ localTime(item.next_run_at) }}</span></div>
            <div v-if="latestRun(item.id)" class="schedule-run" :class="latestRun(item.id)?.status.toLowerCase()"><strong>{{ latestRun(item.id)?.status }}</strong><span>{{ latestRun(item.id)?.provider }} · 发现 {{ latestRun(item.id)?.result_count }} · 新增 {{ latestRun(item.id)?.imported_count }} · 去重 {{ latestRun(item.id)?.duplicate_count }} · 跳过 {{ latestRun(item.id)?.skipped_count }}</span><small v-if="latestRun(item.id)?.error_message">{{ latestRun(item.id)?.error_code }} · {{ latestRun(item.id)?.error_message }}</small></div>
            <footer><small>查询哈希 {{ item.query_hash.slice(0, 12) }}… · 最近运行 {{ localTime(item.last_run_at) }}</small><el-button v-if="canSchedule" link type="primary" @click="openScheduleForm(item)">编辑配置</el-button></footer>
          </article>
        </div>
      </el-tab-pane>

      <el-tab-pane v-if="canSchedule" label="配置自动发现" name="schedule-form">
        <div class="research-form-intro"><span>SCHEDULED WEB DISCOVERY</span><strong>{{ editingScheduleID ? '编辑自动研究任务' : '创建自动研究任务' }}</strong><p>启用后由 Worker 周期运行；结果保留来源和查询哈希，并始终等待人工核验。</p></div>
        <div class="research-form schedule-form">
          <label>任务名称<el-input v-model="scheduleForm.name" maxlength="120" aria-label="自动研究任务名称"/></label>
          <label>关联计划（可选）<el-select v-model="scheduleForm.campaign_id" clearable aria-label="自动研究关联计划"><el-option v-for="campaign in campaigns" :key="campaign.id" :label="campaign.name" :value="campaign.id"/></el-select></label>
          <label>分类<el-select v-model="scheduleForm.category" aria-label="自动研究分类"><el-option label="市场" value="MARKET"/><el-option label="政策" value="POLICY"/><el-option label="竞品" value="COMPETITOR"/></el-select></label>
          <label>运行间隔<el-select v-model="scheduleForm.interval_minutes" aria-label="自动研究运行间隔"><el-option label="每小时" :value="60"/><el-option label="每 6 小时" :value="360"/><el-option label="每天" :value="1440"/><el-option label="每 3 天" :value="4320"/><el-option label="每周" :value="10080"/></el-select></label>
          <label>国家/地区<el-input v-model="scheduleForm.country" maxlength="2" placeholder="US（可选）" aria-label="自动研究国家地区"/></label>
          <label>搜索语言<el-input v-model="scheduleForm.search_lang" maxlength="12" placeholder="en（可选）" aria-label="自动研究搜索语言"/></label>
          <label>时效范围<el-select v-model="scheduleForm.freshness" aria-label="自动研究时效范围"><el-option label="不限" value=""/><el-option label="最近一天" value="pd"/><el-option label="最近一周" value="pw"/><el-option label="最近一月" value="pm"/><el-option label="最近一年" value="py"/></el-select></label>
          <label>单次结果数<el-input-number v-model="scheduleForm.result_count" :min="1" :max="20" controls-position="right" aria-label="自动研究单次结果数"/></label>
          <label class="wide">搜索查询<el-input v-model="scheduleForm.query" type="textarea" :rows="3" maxlength="400" show-word-limit aria-label="自动研究搜索查询"/></label>
          <label class="schedule-enabled wide"><span><strong>保存后启用</strong><small>启用任务需要服务端配置联网搜索和结果存储权。</small></span><el-switch v-model="scheduleForm.enabled" aria-label="保存后启用自动研究任务"/></label>
          <div class="wide form-submit"><span>每家公司最多启用 20 个任务；运行失败会记录安全错误，并推进到下一周期。</span><div><el-button :disabled="savingSchedule" @click="activePane = 'schedules'">返回任务</el-button><el-button type="primary" :loading="savingSchedule" @click="saveSchedule">保存任务</el-button></div></div>
        </div>
      </el-tab-pane>

      <el-tab-pane v-if="canCreate" label="登记新来源" name="create">
        <div class="research-form-intro"><span>NEW VERIFIED SOURCE</span><strong>登记公开研究资料</strong><p>填写来源与摘要后先进入待核验区，不会直接影响 Agent 分析。</p></div>
        <div class="research-form">
          <label>分类<el-select v-model="form.category" aria-label="研究分类"><el-option label="市场" value="MARKET"/><el-option label="政策" value="POLICY"/><el-option label="竞品" value="COMPETITOR"/></el-select></label>
          <label>关联计划（可选）<el-select v-model="form.campaign_id" clearable aria-label="关联计划"><el-option v-for="campaign in campaigns" :key="campaign.id" :label="campaign.name" :value="campaign.id"/></el-select></label>
          <label>发布日期<el-date-picker v-model="form.published_at" type="date" value-format="YYYY-MM-DD" aria-label="发布日期"/></label>
          <label>发布方<el-input v-model="form.publisher" maxlength="200" aria-label="发布方"/></label>
          <label class="wide">标题<el-input v-model="form.title" maxlength="300" aria-label="来源标题"/></label>
          <label class="wide">HTTPS 来源<el-input v-model="form.source_url" placeholder="https://…" aria-label="HTTPS 来源地址"/></label>
          <label class="wide">摘要<el-input v-model="form.summary" type="textarea" :rows="4" maxlength="2000" show-word-limit aria-label="来源摘要"/></label>
          <div class="wide form-submit"><span>登记后状态为 PENDING，ADMIN/MANAGER 核验后生效。</span><div><el-button :disabled="saving" @click="activePane = 'sources'">返回列表</el-button><el-button type="primary" :loading="saving" @click="create">登记来源</el-button></div></div>
        </div>
      </el-tab-pane>
    </el-tabs>
  </section>
</template>

<style scoped>
.research-dialog-panel{display:grid;gap:14px;min-height:460px}.research-management-tabs{min-width:0}.research-tab-label{display:inline-flex;align-items:center;gap:7px}.research-tab-label i{min-width:20px;padding:1px 6px;border-radius:999px;background:#eef0ff;color:#6559ec;font-size:10px;font-style:normal;text-align:center}.research-list-toolbar{display:flex;justify-content:space-between;gap:20px;align-items:flex-start;padding:8px 2px 16px}.research-list-toolbar strong{font-size:15px}.research-list-toolbar p{margin:5px 0 0;color:var(--ink-500);font-size:12px}.research-actions{display:flex;gap:10px;align-items:center}.research-form-intro{padding:18px 20px;color:white;border-radius:16px 16px 0 0;background:linear-gradient(120deg,#151d38,#20284c)}.research-form-intro span{display:block;margin-bottom:8px;color:#8f99ff;font-size:9px;font-weight:800;letter-spacing:.16em}.research-form-intro strong{font-size:17px}.research-form-intro p{margin:6px 0 0;color:#aeb6cf;font-size:12px}.research-form{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px;padding:20px;border:1px solid var(--line);border-top:0;border-radius:0 0 16px 16px;background:#f6f7fb}.research-form label{display:grid;gap:7px;font-size:12px;color:var(--ink-500)}.research-form .wide{grid-column:1/-1}.form-submit{display:flex;justify-content:space-between;align-items:center;color:var(--ink-500);font-size:12px}.research-list{display:grid;gap:10px;max-height:52vh;overflow:auto;padding-right:4px}.research-list article{display:flex;justify-content:space-between;gap:20px;padding:16px;border:1px solid var(--line);border-radius:14px;background:#fff}.source-main{display:grid;gap:7px;min-width:0}.source-main>div{display:flex;gap:8px;align-items:center;color:var(--ink-500);font-size:12px}.source-main a{color:var(--ink-800);font-weight:700;text-decoration:none}.source-main a:hover{text-decoration:underline}.source-main p{margin:0;color:var(--ink-500);font-size:12px;line-height:1.65}.source-main small{color:var(--ink-500)}.review-actions{display:flex;align-items:center;flex-shrink:0}.research-loading{padding:60px 28px;text-align:center;color:var(--ink-500)}
.schedule-list{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px;max-height:52vh;overflow:auto;padding-right:4px}.schedule-list article{display:grid;gap:12px;padding:16px;border:1px solid var(--line);border-radius:14px;background:#fff}.schedule-list header,.schedule-list footer{display:flex;justify-content:space-between;gap:14px;align-items:center}.schedule-list header>div{display:flex;gap:8px;align-items:center;min-width:0}.schedule-list header span{color:var(--ink-500);font-size:11px}.schedule-list header strong{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.schedule-list>article>p{min-height:38px;margin:0;color:var(--ink-700);font-size:12px;line-height:1.6}.schedule-meta{display:flex;gap:6px;flex-wrap:wrap}.schedule-meta span{padding:5px 8px;border-radius:999px;background:#f2f3f8;color:var(--ink-500);font-size:10px}.schedule-run{display:grid;gap:4px;padding:10px 12px;border-left:3px solid #7a70ef;border-radius:8px;background:#f6f6ff;font-size:11px}.schedule-run.succeeded{border-left-color:#63a951;background:#f2f8ef}.schedule-run.failed{border-left-color:#df6767;background:#fff3f3}.schedule-run span,.schedule-run small,.schedule-list footer small{color:var(--ink-500)}.schedule-list footer{padding-top:3px;border-top:1px solid #eef0f5}.schedule-enabled{display:flex!important;grid-template-columns:1fr auto;align-items:center;padding:13px 14px;border:1px solid var(--line);border-radius:12px;background:#fff}.schedule-enabled span{display:grid;gap:4px}.schedule-enabled small{color:var(--ink-500)}
@media(max-width:900px){.schedule-list{grid-template-columns:1fr}}@media(max-width:760px){.research-list-toolbar,.research-list article{flex-direction:column}.research-actions{width:100%;flex-wrap:wrap}.review-actions{align-self:flex-end}.research-form{grid-template-columns:1fr}.research-form .wide{grid-column:auto}.form-submit{align-items:stretch;gap:12px;flex-direction:column}.form-submit>div{display:flex;justify-content:flex-end}}@media(max-width:480px){.review-actions{align-self:stretch}.review-actions :deep(.el-button){flex:1}.form-submit>div{display:grid;grid-template-columns:1fr 1fr}.schedule-list header>div{align-items:flex-start;flex-wrap:wrap}.schedule-list footer{align-items:flex-start;flex-direction:column}}
</style>
