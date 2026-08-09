<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElDatePicker } from 'element-plus/es/components/date-picker/index'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import 'element-plus/es/components/date-picker/style/css'
import { researchApi } from '@/api/research'
import { useAuthStore } from '@/stores/auth'
import type { Campaign } from '@/types/catalog'
import type { ResearchCategory, ResearchSource, ResearchSourceInput } from '@/types/research'
import { validateResearchSourceInput } from '@/utils/research'

const props = defineProps<{ campaigns: Campaign[] }>()
const auth = useAuthStore()
const rows = ref<ResearchSource[]>([])
const loading = ref(true)
const saving = ref(false)
const reviewingID = ref('')
const error = ref('')
const activePane = ref<'sources' | 'create'>('sources')
const form = reactive<ResearchSourceInput>({ category: 'MARKET', title: '', summary: '', source_url: '', publisher: '', published_at: new Date().toISOString().slice(0, 10) })

const canCreate = computed(() => auth.hasAnyRole(['ADMIN', 'MANAGER', 'ANALYST']))
const canReview = computed(() => auth.hasAnyRole(['ADMIN', 'MANAGER']))
const pendingCount = computed(() => rows.value.filter((item) => item.status === 'PENDING').length)

async function load() {
  loading.value = true
  error.value = ''
  try { rows.value = await researchApi.list() }
  catch (cause: any) { error.value = cause.response?.data?.message || '研究来源加载失败' }
  finally { loading.value = false }
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
            <div class="source-main"><div><el-tag size="small" :type="statusType(item.status)">{{ item.status }}</el-tag><span>{{ item.category }}</span><el-tag v-if="item.discovery_method === 'WEB_SEARCH'" size="small" type="primary" effect="plain">实时联网发现</el-tag></div><a :href="item.source_url" target="_blank" rel="noopener noreferrer">{{ item.title }}</a><p>{{ item.summary }}</p><small>{{ item.publisher }} · {{ item.published_at.slice(0, 10) }} · SHA-256 {{ item.content_hash.slice(0, 12) }}…</small><small v-if="item.discovery_method === 'WEB_SEARCH'">发现来源：{{ item.discovery_provider }} · {{ item.discovered_at?.slice(0, 16).replace('T', ' ') }}</small><small v-if="item.review_comment">审核意见：{{ item.review_comment }}</small></div>
            <div v-if="canReview && item.status === 'PENDING'" class="review-actions"><el-button type="success" plain :loading="reviewingID === item.id" @click="verify(item)">核验通过</el-button><el-button type="danger" plain :disabled="reviewingID === item.id" @click="reject(item)">驳回</el-button></div>
          </article>
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
@media(max-width:760px){.research-list-toolbar,.research-list article{flex-direction:column}.research-actions{width:100%;flex-wrap:wrap}.review-actions{align-self:flex-end}.research-form{grid-template-columns:1fr}.research-form .wide{grid-column:auto}.form-submit{align-items:stretch;gap:12px;flex-direction:column}.form-submit>div{display:flex;justify-content:flex-end}}@media(max-width:480px){.review-actions{align-self:stretch}.review-actions :deep(.el-button){flex:1}.form-submit>div{display:grid;grid-template-columns:1fr 1fr}}
</style>
