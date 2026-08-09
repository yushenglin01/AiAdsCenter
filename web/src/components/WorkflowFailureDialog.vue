<script setup lang="ts">
import { computed } from 'vue'
import type { WorkflowDetails, WorkflowNotification, WorkflowStep } from '@/types/workflow'

const props = defineProps<{
  modelValue: boolean
  notification?: WorkflowNotification
  details?: WorkflowDetails
  campaignName?: string
  loading?: boolean
  error?: string
}>()

const emit = defineEmits<{ (event: 'update:modelValue', value: boolean): void; (event: 'locate'): void }>()

const failedSteps = computed(() => props.details?.steps.filter((step) => ['FAILED', 'MANUAL_REVIEW', 'CANCELLED'].includes(step.status)) || [])
const primaryStep = computed<WorkflowStep | undefined>(() => failedSteps.value.find((step) => step.error_message) || failedSteps.value[0])
const exactReason = computed(() => primaryStep.value?.error_message || props.details?.workflow.error_message || props.notification?.message || '未记录具体失败原因')
const failureTime = computed(() => primaryStep.value?.finished_at || props.details?.workflow.finished_at || props.notification?.created_at)
const guidance = computed(() => {
  const reason = exactReason.value.toLowerCase()
  if (reason.includes('campaign metrics not found')) return '当前广告计划缺少分析所需的经营指标。请先完成数据同步，确认分析日期存在指标记录后再重新启动工作流。'
  if (reason.includes('research') && reason.includes('source')) return 'Research Agent 没有找到可用的已核验来源。请先在“来源治理”中登记并核验资料，再重新运行。'
  if (reason.includes('timeout') || reason.includes('deadline')) return 'Agent 执行超时。请检查依赖服务状态和数据量，确认恢复后重新运行。'
  return '检查失败 Agent 的输入数据与依赖服务，修复上方原始错误后再重新启动工作流。'
})

function statusType(status: string) {
  if (status === 'SUCCEEDED') return 'success'
  if (status === 'FAILED' || status === 'CANCELLED') return 'danger'
  if (status === 'RUNNING') return 'primary'
  return 'info'
}
</script>

<template>
  <el-drawer :model-value="modelValue" class="workflow-failure-drawer" size="760px" direction="rtl" :with-header="false" :close-on-click-modal="false" @update:model-value="emit('update:modelValue', $event)">
    <div class="failure-drawer-shell">
      <header class="failure-dialog-heading">
        <div><span class="eyebrow">INCIDENT REVIEW</span><h3>多 Agent 工作流失败详情</h3><p>查看真实失败步骤、原始错误和建议处理方式。</p></div>
        <div class="failure-heading-actions"><span class="failure-severity"><i></i> ACTION REQUIRED</span><button type="button" aria-label="关闭失败详情" @click="emit('update:modelValue', false)">×</button></div>
      </header>

      <main class="failure-drawer-body">
        <div v-if="loading" class="failure-dialog-loading">正在读取工作流失败信息…</div>
        <el-alert v-else-if="error" :title="error" type="error" show-icon />
        <div v-else-if="details" class="failure-dialog-content">
          <section class="failure-summary">
            <div><span>广告计划</span><strong>{{ campaignName || details.workflow.campaign_id }}</strong></div>
            <div><span>失败 Agent</span><strong>{{ primaryStep?.agent_name || details.workflow.current_step }}</strong></div>
            <div><span>发生时间</span><strong>{{ failureTime?.slice(0, 16).replace('T', ' ') || '-' }}</strong></div>
            <div><span>工作流状态</span><strong class="danger">{{ details.workflow.status }}</strong></div>
          </section>

          <section class="failure-reason" aria-label="具体失败原因">
            <span>具体失败原因</span>
            <code>{{ exactReason }}</code>
            <p>{{ guidance }}</p>
          </section>

          <section class="failure-step-section">
            <header><div><strong>Agent 执行链</strong><p>失败步骤已突出显示，上游成功步骤无需重复排查。</p></div><small>{{ details.steps.length }} 个步骤</small></header>
            <ol class="failure-step-list">
              <li v-for="step in details.steps" :key="step.step_id" :class="{ failed: step.status === 'FAILED' }">
                <span class="step-sequence">{{ step.sequence_number }}</span>
                <div><strong>{{ step.agent_name }}</strong><small>{{ step.execution_mode }}</small><code v-if="step.error_message">{{ step.error_message }}</code></div>
                <el-tag :type="statusType(step.status)" size="small">{{ step.status }}</el-tag>
              </li>
            </ol>
          </section>

          <div class="failure-identifiers"><span>WORKFLOW</span><code>{{ details.workflow.workflow_id }}</code><span v-if="details.workflow.trace_id">TRACE</span><code v-if="details.workflow.trace_id">{{ details.workflow.trace_id }}</code></div>
        </div>
        <el-empty v-else :image-size="52" description="没有可展示的工作流详情" />
      </main>

      <footer class="failure-drawer-footer"><el-button @click="emit('update:modelValue', false)">关闭</el-button><el-button v-if="details" type="primary" @click="emit('locate')">定位到工作流</el-button></footer>
    </div>
  </el-drawer>
</template>

<style scoped>
.failure-drawer-shell{height:100%;display:flex;flex-direction:column;background:#fbfcff}.failure-dialog-heading{flex:0 0 auto;display:flex;justify-content:space-between;gap:20px;align-items:flex-start;padding:22px 24px 17px;border-bottom:1px solid var(--line);background:#fff}.failure-dialog-heading h3{margin:5px 0;font-size:20px}.failure-dialog-heading p{margin:0;color:var(--ink-500);font-size:12px}.failure-heading-actions{display:flex;align-items:center;gap:12px}.failure-heading-actions>button{width:30px;height:30px;border:0;border-radius:50%;color:#8a90a0;background:#f2f3f7;font-size:20px;line-height:1;cursor:pointer}.failure-heading-actions>button:hover{color:#3c4252;background:#e8eaf1}.failure-heading-actions>button:focus-visible{outline:2px solid var(--orbit);outline-offset:2px}.failure-severity{display:inline-flex;align-items:center;gap:7px;padding:7px 10px;border-radius:999px;color:#a74444;background:#fff0f0;font-size:9px;font-weight:800;letter-spacing:.08em;white-space:nowrap}.failure-severity i{width:6px;height:6px;border-radius:50%;background:#d84e4e;box-shadow:0 0 0 4px #f7d9d9}.failure-drawer-body{min-height:0;flex:1;padding:20px 24px;overflow:auto}.failure-drawer-footer{flex:0 0 auto;padding:14px 24px;border-top:1px solid var(--line);background:#fff;text-align:right}.failure-dialog-loading{padding:90px 20px;text-align:center;color:var(--ink-500)}.failure-dialog-content{display:grid;gap:16px}.failure-summary{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:1px;border:1px solid var(--line);border-radius:14px;background:var(--line);overflow:hidden}.failure-summary>div{display:grid;gap:6px;padding:14px;background:#fff}.failure-summary span{color:var(--ink-500);font-size:9px;font-weight:800;letter-spacing:.08em}.failure-summary strong{font-size:12px;overflow-wrap:anywhere}.failure-summary .danger{color:#c04646}.failure-reason{position:relative;display:grid;gap:9px;padding:18px 20px 18px 24px;border:1px solid #f1cece;border-radius:15px;background:#fff8f8;overflow:hidden}.failure-reason::before{content:"";position:absolute;inset:0 auto 0 0;width:4px;background:#d84e4e}.failure-reason>span{color:#a74444;font-size:9px;font-weight:800;letter-spacing:.12em}.failure-reason code{color:#722f2f;font-size:13px;font-weight:700;white-space:pre-wrap;overflow-wrap:anywhere}.failure-reason p{margin:0;color:#7e5b5b;font-size:12px;line-height:1.65}.failure-step-section{border:1px solid var(--line);border-radius:15px;overflow:hidden}.failure-step-section>header{display:flex;justify-content:space-between;gap:18px;align-items:flex-start;padding:14px 16px;background:#f7f8fc}.failure-step-section header strong{font-size:13px}.failure-step-section header p{margin:4px 0 0;color:var(--ink-500);font-size:10px}.failure-step-section header small{color:var(--ink-500)}.failure-step-list{display:grid;margin:0;padding:0;list-style:none}.failure-step-list li{display:grid;grid-template-columns:auto 1fr auto;gap:12px;align-items:center;padding:11px 14px;border-top:1px solid var(--line)}.failure-step-list li.failed{background:#fff8f8}.step-sequence{display:grid;width:24px;height:24px;place-items:center;border-radius:50%;color:#6e7487;background:#eff1f6;font-size:10px;font-weight:800}.failed .step-sequence{color:white;background:#d84e4e}.failure-step-list li>div{display:grid;gap:3px}.failure-step-list li strong{font-size:11px}.failure-step-list li small{color:var(--ink-500);font-size:9px}.failure-step-list li code{margin-top:3px;color:#b13f3f;font-size:10px;overflow-wrap:anywhere}.failure-identifiers{display:grid;grid-template-columns:auto 1fr;gap:6px 10px;padding:2px 4px;color:var(--ink-500);font-size:9px}.failure-identifiers span{font-weight:800;letter-spacing:.08em}.failure-identifiers code{overflow-wrap:anywhere}
:global(.workflow-failure-drawer.el-drawer){max-width:100vw;border-radius:20px 0 0 20px;overflow:hidden;box-shadow:-22px 0 54px rgba(13,20,40,.18)}:global(.workflow-failure-drawer .el-drawer__body){padding:0;overflow:hidden}
@media(max-width:680px){.failure-dialog-heading{flex-direction:column}.failure-heading-actions{width:100%;justify-content:space-between}.failure-summary{grid-template-columns:1fr 1fr}}@media(max-width:440px){.failure-summary{grid-template-columns:1fr}.failure-step-list li{grid-template-columns:auto 1fr}.failure-step-list li>.el-tag{grid-column:2}}
</style>
