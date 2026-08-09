<script setup lang="ts">
import { nextTick, ref } from 'vue'
import ResearchSourcePanel from '@/components/ResearchSourcePanel.vue'
import type { Campaign } from '@/types/catalog'

defineProps<{ campaigns: Campaign[] }>()

const dialogOpen = ref(false)
const panel = ref<{ load: () => Promise<void> }>()
const needsRefresh = ref(false)

async function open() {
  dialogOpen.value = true
  if (!needsRefresh.value) return
  await nextTick()
  await panel.value?.load()
  needsRefresh.value = false
}

async function refresh() {
  if (!dialogOpen.value) {
    needsRefresh.value = true
    return
  }
  await nextTick()
  await panel.value?.load()
}

defineExpose({ open, refresh })
</script>

<template>
  <section class="research-source-entry" aria-labelledby="research-source-entry-title">
    <div class="source-entry-mark" aria-hidden="true"><span></span><i></i><i></i></div>
    <div class="source-entry-copy">
      <span class="eyebrow">SOURCE GOVERNANCE</span>
      <h3 id="research-source-entry-title">Research Agent 来源治理</h3>
      <p>登记、核验和追踪外部研究资料；完整管理流程已从 Agent 工作台拆分。</p>
    </div>
    <div class="source-entry-rules" aria-label="来源准入规则"><span>HTTPS ONLY</span><span>人工核验</span><span>全程溯源</span></div>
    <el-button type="primary" @click="open">打开来源库</el-button>
  </section>

  <el-dialog v-model="dialogOpen" class="research-source-dialog" width="1120px" align-center :close-on-click-modal="false">
    <template #header>
      <div class="research-dialog-heading"><div><span class="eyebrow">VERIFIED RESEARCH SOURCES</span><h3>Research Agent 来源库</h3><p>独立管理资料登记与审核，不占用 Agent 运行中心的主操作空间。</p></div><span class="research-dialog-boundary"><i></i> HUMAN VERIFIED</span></div>
    </template>
    <ResearchSourcePanel ref="panel" :campaigns="campaigns" />
  </el-dialog>
</template>

<style scoped>
.research-source-entry{position:relative;display:grid;grid-template-columns:auto minmax(260px,1fr) auto auto;gap:22px;align-items:center;padding:20px 22px;border:1px solid var(--line);border-radius:18px;background:#fff;overflow:hidden}.research-source-entry::after{content:"";position:absolute;inset:0 auto 0 0;width:3px;background:linear-gradient(#6b5cff,#9b92ff)}.source-entry-mark{position:relative;width:48px;height:48px;border:1px solid #dfe2f3;border-radius:14px;background:#f6f7ff}.source-entry-mark span{position:absolute;inset:11px;border:1px solid #7064ef;border-radius:8px}.source-entry-mark i{position:absolute;width:5px;height:5px;border-radius:50%;background:#7064ef}.source-entry-mark i:nth-child(2){top:8px;right:8px}.source-entry-mark i:nth-child(3){bottom:8px;left:8px}.source-entry-copy h3{margin:4px 0 5px;font-size:16px}.source-entry-copy p{margin:0;color:var(--ink-500);font-size:12px}.source-entry-rules{display:flex;gap:7px;flex-wrap:wrap;justify-content:flex-end}.source-entry-rules span{padding:6px 9px;border:1px solid #e2e4f1;border-radius:999px;color:#6d7489;background:#fafaff;font-size:9px;font-weight:800;letter-spacing:.06em}.research-dialog-heading{display:flex;justify-content:space-between;gap:20px;align-items:flex-start;padding-right:28px}.research-dialog-heading h3{margin:5px 0 5px;font-size:20px}.research-dialog-heading p{margin:0;color:var(--ink-500);font-size:12px}.research-dialog-boundary{display:inline-flex;align-items:center;gap:7px;margin-top:5px;padding:7px 10px;border-radius:999px;color:#527348;background:#f0f8ed;font-size:9px;font-weight:800;letter-spacing:.08em;white-space:nowrap}.research-dialog-boundary i{width:6px;height:6px;border-radius:50%;background:#68ad54;box-shadow:0 0 0 4px #dff0da}
:global(.research-source-dialog.el-dialog){display:flex;flex-direction:column;max-width:calc(100vw - 32px);max-height:calc(100vh - 32px);margin:0;border-radius:20px;overflow:hidden}:global(.research-source-dialog .el-dialog__header){flex:0 0 auto;margin:0;padding:22px 24px 17px;border-bottom:1px solid var(--line)}:global(.research-source-dialog .el-dialog__body){min-height:0;padding:16px 24px 24px;background:#fbfcff;overflow:auto}
@media(max-width:900px){.research-source-entry{grid-template-columns:auto 1fr auto}.source-entry-rules{grid-column:2/4;justify-content:flex-start}}@media(max-width:620px){.research-source-entry{grid-template-columns:auto 1fr}.research-source-entry>.el-button{grid-column:1/-1;width:100%}.source-entry-rules{grid-column:1/-1;justify-content:flex-start}.research-dialog-heading{flex-direction:column}.research-dialog-boundary{margin-top:0}}
</style>
