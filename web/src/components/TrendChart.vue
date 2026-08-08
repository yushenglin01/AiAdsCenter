<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { init, use, type ECharts } from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { TrendPoint } from '@/types/metrics'

use([LineChart, GridComponent, LegendComponent, TooltipComponent, CanvasRenderer])
const props = defineProps<{ points: TrendPoint[] }>()
const root = ref<HTMLElement>()
let chart: ECharts | undefined
const resize = () => chart?.resize()
function render() {
  if (!root.value) return
  chart ||= init(root.value)
  chart.setOption({
    color: ['#216147', '#d98a43', '#7a8f85'],
    tooltip: { trigger: 'axis' }, legend: { top: 0, right: 8 },
    grid: { left: 54, right: 30, top: 44, bottom: 38 },
    xAxis: { type: 'category', data: props.points.map((p) => p.date.slice(5, 10)), boundaryGap: false },
    yAxis: [{ type: 'value', name: '金额' }, { type: 'value', name: 'ROAS' }],
    series: [
      { name: '消耗', type: 'line', smooth: true, symbol: 'none', data: props.points.map((p) => Number(p.spend)) },
      { name: 'D7 收入', type: 'line', smooth: true, symbol: 'none', data: props.points.map((p) => Number(p.revenue_d7)) },
      { name: 'D7 ROAS', type: 'line', yAxisIndex: 1, smooth: true, symbol: 'none', data: props.points.map((p) => Number(p.roas_d7)) },
    ],
  })
}
watch(() => props.points, render, { deep: true })
onMounted(() => { render(); window.addEventListener('resize', resize) })
onBeforeUnmount(() => { window.removeEventListener('resize', resize); chart?.dispose() })
</script>

<template><div ref="root" class="trend-chart" role="img" aria-label="消耗、收入和 ROAS 趋势图" /></template>
