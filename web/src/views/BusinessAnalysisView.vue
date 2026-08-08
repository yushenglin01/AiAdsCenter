<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ElDatePicker } from 'element-plus/es/components/date-picker/index'
import { ElMessage } from 'element-plus/es/components/message/index'
import 'element-plus/es/components/date-picker/style/css'
import AppShell from '@/components/AppShell.vue'
import { businessApi } from '@/api/business'
import { catalogApi } from '@/api/catalog'
import type { Campaign } from '@/types/catalog'
import type { AgentTask, TaskDetails } from '@/types/business'

const campaigns=ref<Campaign[]>([]);const tasks=ref<AgentTask[]>([]);const details=ref<TaskDetails>();const campaignID=ref('');const analysisDate=ref(new Date().toISOString().slice(0,10));const loading=ref(true);const running=ref(false);const error=ref('')
const campaignName=computed(()=>campaigns.value.find((row)=>row.id===details.value?.task.campaign_id)?.name||'经营分析')
const canAnalyze=computed(()=>Boolean(campaignID.value&&analysisDate.value))
const statusType=(value:string)=>value==='SUCCEEDED'?'success':value==='FAILED'||value==='MANUAL_REVIEW'?'danger':value==='RUNNING'||value==='RETRYING'||value==='WAITING_APPROVAL'?'warning':'info'
const riskType=(value:string)=>value==='HIGH'||value==='CRITICAL'?'danger':value==='MEDIUM'?'warning':'info'
const terminal=(value:string)=>['WAITING_APPROVAL','SUCCEEDED','FAILED','MANUAL_REVIEW','CANCELLED'].includes(value)
let eventController:AbortController|undefined
let reconnectTimer:number|undefined
async function load(){loading.value=true;error.value='';try{[campaigns.value,tasks.value]=await Promise.all([catalogApi.listCampaigns(),businessApi.listTasks()]);campaignID.value ||= campaigns.value[0]?.id||'';if(tasks.value.length){details.value=await businessApi.getTask(tasks.value[0].task_id);if(!terminal(details.value.task.status))void watchTask(details.value.task.task_id)}}catch(e:any){error.value=e.response?.data?.message||'经营分析加载失败'}finally{loading.value=false}}
function applyEvent(value:TaskDetails){details.value=value;const index=tasks.value.findIndex((task)=>task.task_id===value.task.task_id);if(index>=0)tasks.value[index]=value.task;else tasks.value.unshift(value.task);if(terminal(value.task.status))ElMessage.success(value.task.status==='SUCCEEDED'?'异步经营分析完成':value.task.status==='WAITING_APPROVAL'?'分析完成，等待人工审批':'异步任务已结束')}
async function watchTask(taskID:string){eventController?.abort();if(reconnectTimer)window.clearTimeout(reconnectTimer);eventController=new AbortController();try{await businessApi.watchTask(taskID,applyEvent,eventController.signal)}catch(e:any){if(e.name!=='AbortError'){details.value=await businessApi.getTask(taskID);if(!terminal(details.value.task.status))reconnectTimer=window.setTimeout(()=>watchTask(taskID),1000)}}}
async function selectTask(task:AgentTask){eventController?.abort();details.value=await businessApi.getTask(task.task_id);if(!terminal(details.value.task.status))void watchTask(task.task_id)}
async function analyze(){if(!canAnalyze.value)return;running.value=true;error.value='';try{details.value=await businessApi.analyze({game_id:campaigns.value.find((x)=>x.id===campaignID.value)?.game_id||'',campaign_id:campaignID.value,analysis_date:analysisDate.value});ElMessage.success(terminal(details.value.task.status)?'已返回幂等任务':'任务已接收并进入异步队列');tasks.value=await businessApi.listTasks();if(!terminal(details.value.task.status))void watchTask(details.value.task.task_id)}catch(e:any){error.value=e.response?.data?.message||'经营分析任务提交失败'}finally{running.value=false}}
function downloadReport(){if(!details.value?.report)return;const blob=new Blob([details.value.report.content_markdown],{type:'text/markdown;charset=utf-8'});const url=URL.createObjectURL(blob);const anchor=document.createElement('a');anchor.href=url;anchor.download=`business-analysis-${details.value.task.task_id}.md`;anchor.click();URL.revokeObjectURL(url)}
onMounted(load)
onUnmounted(()=>{eventController?.abort();if(reconnectTimer)window.clearTimeout(reconnectTimer)})
</script>

<template><AppShell>
  <div class="page-heading"><div><span class="eyebrow">BUSINESS AGENT · ASYNC</span><h2>经营分析</h2><p>API 只接收任务，Worker 异步执行；模型只生成有证据的建议，不会修改广告计划。</p></div></div>
  <el-alert v-if="error" :title="error" type="error" show-icon/>
  <section class="analysis-launcher"><label>广告计划<el-select v-model="campaignID" aria-label="广告计划"><el-option v-for="campaign in campaigns" :key="campaign.id" :label="campaign.name" :value="campaign.id"/></el-select></label><label>分析日期<el-date-picker v-model="analysisDate" aria-label="分析日期" type="date" value-format="YYYY-MM-DD"/></label><el-button type="primary" :loading="running" :disabled="!canAnalyze" @click="analyze">发起经营分析</el-button><span>默认使用 Mock LLM · 相同计划与日期幂等</span></section>
  <div v-if="loading" class="loading-panel">正在加载分析任务…</div>
  <div v-else class="analysis-layout">
    <aside class="task-list"><div class="section-title"><h3>分析记录</h3><span>{{tasks.length}}</span></div><el-empty v-if="!tasks.length" description="暂无分析任务"/><button v-for="task in tasks" :key="task.task_id" :class="{active:task.task_id===details?.task.task_id}" @click="selectTask(task)"><span>{{campaigns.find((x)=>x.id===task.campaign_id)?.name||task.campaign_id}}</span><small>{{task.created_at.slice(0,10)}} · {{task.status}}</small></button></aside>
    <main class="analysis-result"><el-empty v-if="!details" description="选择计划发起一次经营分析"/><template v-else><header class="result-header"><div><span class="eyebrow">{{campaignName}}</span><h3>{{details.task.output_json?.summary||details.task.error_message||`任务处理中 · ${details.task.current_step}`}}</h3></div><div class="result-actions"><el-tag :type="statusType(details.task.status)" size="large">{{details.task.status}}</el-tag><el-button v-if="details.report" @click="downloadReport">下载报告</el-button></div></header>
      <section class="agent-meta"><div><span>当前步骤</span><strong>{{details.task.current_step}}</strong></div><div><span>队列重试</span><strong>{{details.task.queue_retry_count}} / 3</strong></div><div><span>模型提供方</span><strong>{{details.usage[0]?.provider||details.attempts[0]?.provider||'等待 Worker'}}</strong></div><div><span>Token / 延迟</span><strong>{{(details.usage[0]?.input_tokens||0)+(details.usage[0]?.output_tokens||0)}} / {{details.usage[0]?.latency_ms||0}} ms</strong></div></section>
      <section v-if="details.report" class="report-snapshot"><span>REPORT SNAPSHOT</span><strong>{{details.report.title}}</strong><p>{{details.report.summary}}</p></section>
      <div class="section-title"><h3>经营发现</h3><span>{{details.findings.length}} 条</span></div><el-empty v-if="!details.findings.length" description="没有经营风险发现"/><div v-else class="finding-list"><article v-for="finding in details.findings" :key="finding.id"><div><el-tag :type="riskType(finding.severity)">{{finding.severity}}</el-tag><strong>{{finding.conclusion}}</strong><span>置信度 {{(finding.confidence*100).toFixed(0)}}%</span></div><p>{{finding.description}}</p><code v-for="evidence in finding.evidence" :key="evidence.metric">{{evidence.metric}} = {{evidence.actual}}<template v-if="evidence.target"> / 目标 {{evidence.target}}</template></code></article></div>
      <div class="section-title"><h3>建议与审批</h3><span>{{details.approvals.length}} 个待审批请求</span></div><el-empty v-if="!details.recommendations.length" description="没有生成建议"/><el-table v-else :data="details.recommendations"><el-table-column prop="action" label="建议动作" min-width="170"/><el-table-column prop="description" label="说明" min-width="330"/><el-table-column prop="risk_level" label="风险" width="90"/><el-table-column label="边界" width="130"><template #default="s"><el-tag :type="s.row.requires_approval?'warning':'info'">{{s.row.requires_approval?'等待人工审批':'仅建议'}}</el-tag></template></el-table-column><el-table-column prop="status" label="状态" width="110"/></el-table>
    </template></main>
  </div>
</AppShell></template>
