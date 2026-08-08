<script setup lang="ts">
import { onMounted, ref } from 'vue'
import AppShell from '@/components/AppShell.vue'
import { operationsApi } from '@/api/operations'
import type { RecommendationDetail } from '@/types/operations'

const rows=ref<RecommendationDetail[]>([]);const loading=ref(true);const error=ref('');const status=ref('');const risk=ref('')
const tagType=(value:string)=>value==='APPROVED'?'success':value==='REJECTED'?'danger':value==='PROPOSED'?'warning':'info'
async function load(){loading.value=true;error.value='';try{rows.value=await operationsApi.recommendations({...status.value?{status:status.value}:{},...risk.value?{risk_level:risk.value}:{}})}catch(e:any){error.value=e.response?.data?.message||'建议列表加载失败'}finally{loading.value=false}}
onMounted(load)
</script>

<template><AppShell>
  <div class="page-heading"><div><span class="eyebrow">RECOMMENDATION CENTER</span><h2>建议中心</h2><p>所有动作都是经营建议，不代表广告平台已经执行。</p></div><el-button :loading="loading" @click="load">刷新</el-button></div>
  <el-alert v-if="error" :title="error" type="error" show-icon/>
  <section class="filter-bar" aria-label="建议筛选"><label>状态<el-select v-model="status" clearable placeholder="全部状态" @change="load"><el-option label="待决策" value="PROPOSED"/><el-option label="已批准" value="APPROVED"/><el-option label="已驳回" value="REJECTED"/></el-select></label><label>风险<el-select v-model="risk" clearable placeholder="全部风险" @change="load"><el-option label="高风险" value="HIGH"/><el-option label="中风险" value="MEDIUM"/><el-option label="低风险" value="LOW"/></el-select></label><span>{{rows.length}} 条建议</span></section>
  <div v-if="loading" class="loading-panel">正在加载建议…</div>
  <el-table v-else-if="rows.length" :data="rows" class="data-table">
    <el-table-column prop="campaign_name" label="广告计划" min-width="150"/>
    <el-table-column label="建议动作" min-width="170"><template #default="s"><strong>{{s.row.recommendation.action}}</strong><small class="table-subtitle">{{s.row.recommendation.suggested_value||'无数值变更'}}</small></template></el-table-column>
    <el-table-column label="业务说明" min-width="320"><template #default="s">{{s.row.recommendation.description}}</template></el-table-column>
    <el-table-column label="风险" width="90"><template #default="s"><el-tag :type="s.row.recommendation.risk_level==='HIGH'?'danger':'warning'">{{s.row.recommendation.risk_level}}</el-tag></template></el-table-column>
    <el-table-column label="审批边界" width="130"><template #default="s">{{s.row.recommendation.requires_approval?'需要人工审批':'仅观察建议'}}</template></el-table-column>
    <el-table-column label="状态" width="110"><template #default="s"><el-tag :type="tagType(s.row.recommendation.status)">{{s.row.recommendation.status}}</el-tag></template></el-table-column>
  </el-table>
  <el-empty v-else description="当前筛选条件下没有建议"/>
</AppShell></template>
