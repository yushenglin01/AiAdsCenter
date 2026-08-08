<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import AppShell from '@/components/AppShell.vue'
import TrendChart from '@/components/TrendChart.vue'
import { catalogApi } from '@/api/catalog'
import { metricsApi } from '@/api/metrics'
import type { Game } from '@/types/catalog'
import type { CampaignMetric, TrendPoint } from '@/types/metrics'

const games=ref<Game[]>([]); const gameID=ref(''); const campaigns=ref<CampaignMetric[]>([]); const trends=ref<TrendPoint[]>([]); const campaignID=ref(''); const loading=ref(false); const analyzing=ref(false); const error=ref('')
const selectedName=computed(()=>campaigns.value.find((x)=>x.campaign_id===campaignID.value)?.campaign_name||'全部计划')
const money=(v:string|number)=>`$${Number(v).toLocaleString('en-US',{maximumFractionDigits:2})}`
const percent=(v:string|number)=>`${(Number(v)*100).toFixed(1)}%`
async function load(){loading.value=true;error.value='';try{if(!games.value.length){games.value=await catalogApi.listGames();gameID.value ||= games.value[0]?.id||''};[campaigns.value,trends.value]=await Promise.all([metricsApi.campaigns(gameID.value),metricsApi.trends(gameID.value,campaignID.value)])}catch(e:any){error.value=e.response?.data?.message||'指标加载失败'}finally{loading.value=false}}
async function changeCampaign(id:string){campaignID.value=id;trends.value=await metricsApi.trends(gameID.value,id)}
async function recalculate(){if(!gameID.value)return;analyzing.value=true;try{const result=await metricsApi.recalculate(gameID.value);ElMessage.success(`完成 ${result.calculated_rows} 行指标计算，发现 ${result.business_findings+result.attribution_findings+result.creative_findings} 个风险信号`);await load()}catch(e:any){error.value=e.response?.data?.message||'分析执行失败'}finally{analyzing.value=false}}
onMounted(load)
</script>

<template><AppShell>
  <div class="page-heading"><div><span class="eyebrow">PERFORMANCE METRICS</span><h2>计划经营指标</h2><p>统一口径计算 CTR、CPI、付费率、ROAS 与预算消耗。</p></div><div class="heading-actions"><el-select v-model="gameID" aria-label="游戏" @change="load"><el-option v-for="game in games" :key="game.id" :label="game.name" :value="game.id"/></el-select><el-button type="primary" :loading="analyzing" @click="recalculate">重新计算与分析</el-button></div></div>
  <el-alert v-if="error" :title="error" type="error" show-icon />
  <div v-if="loading" class="loading-panel">正在加载计划指标…</div>
  <template v-else>
    <el-table v-if="campaigns.length" :data="campaigns" highlight-current-row @current-change="(row:CampaignMetric)=>changeCampaign(row?.campaign_id||'')"><el-table-column prop="campaign_name" label="计划" min-width="190"/><el-table-column prop="country" label="地区" width="70"/><el-table-column label="消耗"><template #default="s">{{money(s.row.spend)}}</template></el-table-column><el-table-column label="CPI"><template #default="s">{{money(s.row.cpi)}}</template></el-table-column><el-table-column label="付费率"><template #default="s">{{percent(s.row.payer_rate)}}</template></el-table-column><el-table-column label="D1 ROAS"><template #default="s">{{Number(s.row.roas_d1).toFixed(2)}}</template></el-table-column><el-table-column label="D7 ROAS"><template #default="s"><strong :class="{'risk-text':Number(s.row.roas_d7)<1.3}">{{Number(s.row.roas_d7).toFixed(2)}}</strong></template></el-table-column><el-table-column label="预算消耗"><template #default="s">{{percent(s.row.budget_consumption_rate)}}</template></el-table-column></el-table>
    <el-empty v-else description="暂无指标，请先完成数据导入"/>
    <div class="section-title"><h3>{{selectedName}}趋势</h3><span>点击计划可聚焦查看</span></div><div class="chart-card"><TrendChart v-if="trends.length" :points="trends"/><el-empty v-else description="暂无趋势数据"/></div>
  </template>
</AppShell></template>
