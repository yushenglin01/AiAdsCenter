<script setup lang="ts">
import { onMounted, ref } from 'vue'
import AppShell from '@/components/AppShell.vue'
import { metricsApi } from '@/api/metrics'
import type { CreativeFinding } from '@/types/metrics'
const rows=ref<CreativeFinding[]>([]);const loading=ref(true);const error=ref('')
const severity=(v:string)=>v==='HIGH'?'danger':v==='MEDIUM'?'warning':'info'
async function load(){loading.value=true;try{rows.value=await metricsApi.creative()}catch(e:any){error.value=e.response?.data?.message||'素材分析加载失败'}finally{loading.value=false}}
onMounted(load)
</script>
<template><AppShell><div class="page-heading"><div><span class="eyebrow">CREATIVE AGENT</span><h2>素材疲劳分析</h2><p>综合近 7 日 CTR 降幅与曝光频次，生成可复核的疲劳评分。</p></div><el-button @click="load">刷新</el-button></div><el-alert v-if="error" :title="error" type="error" show-icon/><div v-if="loading" class="loading-panel">正在加载素材信号…</div><el-empty v-else-if="!rows.length" description="当前未发现素材疲劳风险"/><el-table v-else :data="rows"><el-table-column label="等级" width="90"><template #default="s"><el-tag :type="severity(s.row.severity)">{{s.row.severity}}</el-tag></template></el-table-column><el-table-column prop="creative_name" label="素材" min-width="210"/><el-table-column prop="title" label="风险" min-width="180"/><el-table-column label="疲劳评分" width="110"><template #default="s">{{Number(s.row.fatigue_score).toFixed(2)}}</template></el-table-column><el-table-column label="CTR 降幅" width="110"><template #default="s">{{(Number(s.row.ctr_change_7d)*100).toFixed(1)}}%</template></el-table-column><el-table-column label="最新频次" width="100"><template #default="s">{{Number(s.row.frequency).toFixed(2)}}</template></el-table-column><el-table-column prop="description" label="证据说明" min-width="340"/></el-table></AppShell></template>
