<script setup lang="ts">
import { onMounted, ref } from 'vue'
import AppShell from '@/components/AppShell.vue'
import TrendChart from '@/components/TrendChart.vue'
import { metricsApi } from '@/api/metrics'
import { operationsApi } from '@/api/operations'
import type { Overview, TrendPoint } from '@/types/metrics'
import type { OperationsSummary } from '@/types/operations'

const overview = ref<Overview>(); const trends = ref<TrendPoint[]>([]); const operations=ref<OperationsSummary>(); const loading = ref(true); const error = ref('')
const money = (value: string | number | undefined) => `$${Number(value || 0).toLocaleString('en-US', { maximumFractionDigits: 0 })}`
const rate = (value: string | number | undefined) => `${(Number(value || 0) * 100).toFixed(1)}%`
async function load() { loading.value=true; error.value=''; try { [overview.value,trends.value,operations.value]=await Promise.all([metricsApi.overview(),metricsApi.trends(),operationsApi.dashboardOperations()]) } catch(e:any){ error.value=e.response?.data?.message||'经营数据加载失败' } finally { loading.value=false } }
onMounted(load)
</script>

<template><AppShell>
  <div class="page-heading"><div><span class="eyebrow">BUSINESS OVERVIEW</span><h2>经营总览</h2><p>基于导入事实数据的确定性聚合，不使用模型生成数值。</p></div><el-button :loading="loading" @click="load">刷新</el-button></div>
  <el-alert v-if="error" :title="error" type="error" show-icon />
  <div v-if="loading" class="loading-panel">正在计算经营指标…</div>
  <template v-else-if="overview">
    <section class="metric-grid">
      <article><span>广告消耗</span><strong>{{money(overview.spend)}}</strong><small>{{overview.impressions.toLocaleString()}} 次曝光</small></article>
      <article><span>D7 收入</span><strong>{{money(overview.revenue_d7)}}</strong><small>D7 ROAS {{Number(overview.roas_d7).toFixed(2)}}</small></article>
      <article><span>安装成本</span><strong>{{money(overview.cpi)}}</strong><small>{{overview.installs.toLocaleString()}} 次安装</small></article>
      <article><span>付费率</span><strong>{{rate(overview.payer_rate)}}</strong><small>{{overview.payers}} 名付费用户</small></article>
    </section>
    <section class="insight-strip"><div><strong>{{overview.high_risk_campaigns}}</strong><span>高风险计划</span></div><div><strong>{{overview.attribution_anomalies}}</strong><span>归因异常</span></div><div><strong>{{overview.fatigued_creatives}}</strong><span>素材疲劳预警</span></div></section>
    <section v-if="operations" class="operations-strip"><div><span>待审批</span><strong>{{operations.pending_approvals}}</strong></div><div><span>任务成功率</span><strong>{{(operations.task_success_rate*100).toFixed(1)}}%</strong></div><div><span>执行中任务</span><strong>{{operations.active_tasks}}</strong></div><div><span>模型调用</span><strong>{{operations.model_calls}}</strong></div><div><span>估算成本</span><strong>${{Number(operations.model_cost).toFixed(4)}}</strong></div></section>
    <div class="section-title"><h3>经营趋势</h3><span>{{trends.length}} 个观测日</span></div>
    <div class="chart-card"><TrendChart v-if="trends.length" :points="trends"/><el-empty v-else description="导入数据并执行分析后显示趋势"/></div>
  </template>
  <el-empty v-else description="暂无经营指标，请先导入数据并执行分析" />
</AppShell></template>
