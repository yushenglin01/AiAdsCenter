<script setup lang="ts">
import { onMounted, ref } from 'vue'
import AppShell from '@/components/AppShell.vue'
import { operationsApi } from '@/api/operations'
import type { AuditLog } from '@/types/operations'

const rows=ref<AuditLog[]>([]);const loading=ref(true);const error=ref('');const action=ref('')
async function load(){loading.value=true;error.value='';try{rows.value=await operationsApi.auditLogs({...action.value?{action:action.value}:{},limit:150})}catch(e:any){error.value=e.response?.data?.message||'审计日志加载失败'}finally{loading.value=false}}
const json=(value:unknown)=>value?JSON.stringify(value,null,2):'—'
onMounted(load)
</script>

<template><AppShell>
  <div class="page-heading"><div><span class="eyebrow">IMMUTABLE AUDIT TRAIL</span><h2>审计日志</h2><p>记录 Agent Tool 调用和人工审批决策，不记录模型隐藏推理。</p></div><el-button :loading="loading" @click="load">刷新</el-button></div>
  <el-alert v-if="error" :title="error" type="error" show-icon/>
  <section class="filter-bar" aria-label="审计筛选"><label>动作<el-select v-model="action" clearable placeholder="全部动作" @change="load"><el-option label="Agent Tool 调用" value="AGENT_TOOL_CALL"/><el-option label="审批通过" value="APPROVAL_APPROVED"/><el-option label="审批驳回" value="APPROVAL_REJECTED"/></el-select></label><span>{{rows.length}} 条审计事件</span></section>
  <div v-if="loading" class="loading-panel">正在读取审计链…</div>
  <el-table v-else-if="rows.length" :data="rows" class="data-table"><el-table-column type="expand"><template #default="s"><div class="audit-detail"><div><strong>变更前</strong><pre>{{json(s.row.before)}}</pre></div><div><strong>变更后 / 元数据</strong><pre>{{json(s.row.after||s.row.metadata)}}</pre></div></div></template></el-table-column><el-table-column prop="action" label="动作" min-width="180"/><el-table-column prop="actor_type" label="主体" width="130"/><el-table-column prop="resource_type" label="资源" width="150"/><el-table-column prop="resource_id" label="资源 ID" min-width="230"/><el-table-column prop="request_id" label="Request ID" min-width="220"/><el-table-column label="时间" width="180"><template #default="s">{{new Date(s.row.created_at).toLocaleString()}}</template></el-table-column></el-table>
  <el-empty v-else description="暂无审计事件"/>
</AppShell></template>
