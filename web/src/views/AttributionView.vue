<script setup lang="ts">
import { onMounted, ref } from 'vue'
import AppShell from '@/components/AppShell.vue'
import { metricsApi } from '@/api/metrics'
import type { AttributionFinding } from '@/types/metrics'
const rows=ref<AttributionFinding[]>([]);const loading=ref(true);const error=ref('')
const severity=(v:string)=>v==='HIGH'?'danger':v==='MEDIUM'?'warning':'info'
async function load(){loading.value=true;try{rows.value=await metricsApi.attribution()}catch(e:any){error.value=e.response?.data?.message||'归因分析加载失败'}finally{loading.value=false}}
onMounted(load)
</script>
<template><AppShell><div class="page-heading"><div><span class="eyebrow">ATTRIBUTION AGENT</span><h2>归因异常分析</h2><p>对比渠道、MMP 与游戏内收入，按统一差异率识别异常。</p></div><el-button @click="load">刷新</el-button></div><el-alert v-if="error" :title="error" type="error" show-icon/><div v-if="loading" class="loading-panel">正在加载归因证据…</div><el-empty v-else-if="!rows.length" description="当前未发现归因异常"/><el-table v-else :data="rows"><el-table-column label="等级" width="90"><template #default="s"><el-tag :type="severity(s.row.severity)">{{s.row.severity}}</el-tag></template></el-table-column><el-table-column prop="campaign_name" label="计划" min-width="170"/><el-table-column prop="title" label="异常" min-width="180"/><el-table-column label="偏差率" width="110"><template #default="s">{{(Number(s.row.difference_rate)*100).toFixed(1)}}%</template></el-table-column><el-table-column prop="description" label="证据说明" min-width="360"/></el-table></AppShell></template>
