<script setup lang="ts">
import { onMounted, ref } from 'vue'
import AppShell from '@/components/AppShell.vue'
import { operationsApi } from '@/api/operations'
import type { ModelUsage, ModelUsageSummary } from '@/types/operations'

const summary=ref<ModelUsageSummary>();const rows=ref<ModelUsage[]>([]);const loading=ref(true);const error=ref('')
const integer=(value:number)=>Number(value||0).toLocaleString();const money=(value:string|number)=>`$${Number(value||0).toFixed(4)}`
async function load(){loading.value=true;error.value='';try{[summary.value,rows.value]=await Promise.all([operationsApi.modelUsageSummary(),operationsApi.modelUsage()])}catch(e:any){error.value=e.response?.data?.message||'模型用量加载失败'}finally{loading.value=false}}
onMounted(load)
</script>

<template><AppShell>
  <div class="page-heading"><div><span class="eyebrow">MODEL GOVERNANCE</span><h2>模型用量</h2><p>按持久化调用记录汇总 Token、延迟和估算成本。</p></div><el-button :loading="loading" @click="load">刷新</el-button></div>
  <el-alert v-if="error" :title="error" type="error" show-icon/>
  <div v-if="loading" class="loading-panel">正在汇总模型调用…</div>
  <template v-else-if="summary"><section class="metric-grid"><article><span>模型调用</span><strong>{{integer(summary.calls)}}</strong><small>成功率 {{(summary.success_rate*100).toFixed(1)}}%</small></article><article><span>输入 Token</span><strong>{{integer(summary.input_tokens)}}</strong><small>结构化 Prompt 输入</small></article><article><span>输出 Token</span><strong>{{integer(summary.output_tokens)}}</strong><small>JSON 结构化结果</small></article><article><span>估算成本</span><strong>{{money(summary.estimated_cost)}}</strong><small>平均延迟 {{summary.average_latency_ms.toFixed(0)}} ms</small></article></section>
    <div class="section-title"><h3>调用明细</h3><span>{{rows.length}} 条</span></div><el-table :data="rows" class="data-table"><el-table-column prop="provider" label="Provider" width="120"/><el-table-column prop="model" label="模型" min-width="170"/><el-table-column prop="prompt_name" label="Prompt" min-width="160"/><el-table-column label="版本" width="110"><template #default="s">{{s.row.prompt_version}} / {{s.row.schema_version}}</template></el-table-column><el-table-column label="Token" width="120"><template #default="s">{{s.row.input_tokens+s.row.output_tokens}}</template></el-table-column><el-table-column prop="latency_ms" label="延迟 ms" width="110"/><el-table-column prop="status" label="状态" width="140"/><el-table-column label="成本" width="110"><template #default="s">{{money(s.row.estimated_cost)}}</template></el-table-column></el-table>
  </template>
</AppShell></template>
