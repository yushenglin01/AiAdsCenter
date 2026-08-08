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
</script>

<template>
  <section class="research-panel" aria-labelledby="research-source-title">
    <header>
      <div><span class="eyebrow">VERIFIED RESEARCH SOURCES</span><h3 id="research-source-title">Research Agent 来源库</h3><p>只允许 HTTPS 来源；人工核验前不会进入分析上下文。</p></div>
      <div class="research-actions"><el-tag v-if="pendingCount" type="warning">{{ pendingCount }} 条待核验</el-tag><el-button :loading="loading" @click="load">刷新</el-button></div>
    </header>
    <el-alert v-if="error" :title="error" type="error" show-icon />

    <div v-if="canCreate" class="research-form">
      <label>分类<el-select v-model="form.category" aria-label="研究分类"><el-option label="市场" value="MARKET"/><el-option label="政策" value="POLICY"/><el-option label="竞品" value="COMPETITOR"/></el-select></label>
      <label>关联计划（可选）<el-select v-model="form.campaign_id" clearable aria-label="关联计划"><el-option v-for="campaign in campaigns" :key="campaign.id" :label="campaign.name" :value="campaign.id"/></el-select></label>
      <label>发布日期<el-date-picker v-model="form.published_at" type="date" value-format="YYYY-MM-DD" aria-label="发布日期"/></label>
      <label>发布方<el-input v-model="form.publisher" maxlength="200" aria-label="发布方"/></label>
      <label class="wide">标题<el-input v-model="form.title" maxlength="300" aria-label="来源标题"/></label>
      <label class="wide">HTTPS 来源<el-input v-model="form.source_url" placeholder="https://…" aria-label="HTTPS 来源地址"/></label>
      <label class="wide">摘要<el-input v-model="form.summary" type="textarea" :rows="3" maxlength="2000" show-word-limit aria-label="来源摘要"/></label>
      <div class="wide form-submit"><span>登记后状态为 PENDING，ADMIN/MANAGER 核验后生效。</span><el-button type="primary" :loading="saving" @click="create">登记来源</el-button></div>
    </div>

    <div v-if="loading" class="research-loading">正在加载来源…</div>
    <el-empty v-else-if="!rows.length" :image-size="60" description="暂无研究来源；可先登记公开且可核验的资料"/>
    <div v-else class="research-list">
      <article v-for="item in rows" :key="item.id">
        <div class="source-main"><div><el-tag size="small" :type="statusType(item.status)">{{ item.status }}</el-tag><span>{{ item.category }}</span></div><a :href="item.source_url" target="_blank" rel="noopener noreferrer">{{ item.title }}</a><p>{{ item.summary }}</p><small>{{ item.publisher }} · {{ item.published_at.slice(0, 10) }} · SHA-256 {{ item.content_hash.slice(0, 12) }}…</small><small v-if="item.review_comment">审核意见：{{ item.review_comment }}</small></div>
        <div v-if="canReview && item.status === 'PENDING'" class="review-actions"><el-button type="success" plain :loading="reviewingID === item.id" @click="verify(item)">核验通过</el-button><el-button type="danger" plain :disabled="reviewingID === item.id" @click="reject(item)">驳回</el-button></div>
      </article>
    </div>
  </section>
</template>

<style scoped>
.research-panel{padding:24px;border:1px solid var(--line);border-radius:20px;background:white;display:grid;gap:18px}.research-panel>header{display:flex;justify-content:space-between;gap:20px;align-items:flex-start}.research-panel h3{margin:4px 0 6px}.research-panel p{margin:0;color:var(--ink-500)}.research-actions{display:flex;gap:10px;align-items:center}.research-form{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:14px;padding:18px;border-radius:16px;background:#f6f7fb}.research-form label{display:grid;gap:7px;font-size:12px;color:var(--ink-500)}.research-form .wide{grid-column:1/-1}.form-submit{display:flex;justify-content:space-between;align-items:center;color:var(--ink-500);font-size:12px}.research-list{display:grid;gap:10px}.research-list article{display:flex;justify-content:space-between;gap:20px;padding:16px;border:1px solid var(--line);border-radius:14px}.source-main{display:grid;gap:7px;min-width:0}.source-main>div{display:flex;gap:8px;align-items:center;color:var(--ink-500);font-size:12px}.source-main a{color:var(--ink-800);font-weight:700;text-decoration:none}.source-main a:hover{text-decoration:underline}.source-main small{color:var(--ink-500)}.review-actions{display:flex;align-items:center;flex-shrink:0}.research-loading{padding:28px;text-align:center;color:var(--ink-500)}
@media(max-width:900px){.research-form{grid-template-columns:1fr 1fr}.research-panel>header,.research-list article{flex-direction:column}.review-actions{align-self:flex-end}}@media(max-width:560px){.research-form{grid-template-columns:1fr}.research-form .wide{grid-column:auto}.form-submit{align-items:stretch;gap:12px}.review-actions{align-self:stretch}.review-actions :deep(.el-button){flex:1}}
</style>
