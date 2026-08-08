<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import AppShell from '@/components/AppShell.vue'
import { operationsApi } from '@/api/operations'
import { useAuthStore } from '@/stores/auth'
import type { ApprovalDetail } from '@/types/operations'

const auth=useAuthStore();const rows=ref<ApprovalDetail[]>([]);const loading=ref(true);const deciding=ref(false);const error=ref('');const status=ref('PENDING');const dialog=ref(false);const selected=ref<ApprovalDetail>();const decision=ref<'APPROVED'|'REJECTED'>('APPROVED');const comment=ref('')
const canDecide=computed(()=>auth.hasAnyRole(['ADMIN','MANAGER']))
const dialogTitle=computed(()=>decision.value==='APPROVED'?'确认批准建议':'确认驳回建议')
const statusType=(value:string)=>value==='APPROVED'?'success':value==='REJECTED'?'danger':'warning'
async function load(){loading.value=true;error.value='';try{rows.value=await operationsApi.approvals(status.value?{status:status.value}:{})}catch(e:any){error.value=e.response?.data?.message||'审批列表加载失败'}finally{loading.value=false}}
function open(row:ApprovalDetail,value:'APPROVED'|'REJECTED'){selected.value=row;decision.value=value;comment.value='';dialog.value=true}
async function submit(){if(!selected.value)return;if(decision.value==='REJECTED'&&!comment.value.trim()){error.value='驳回时必须填写原因';return}deciding.value=true;error.value='';try{if(decision.value==='APPROVED')await operationsApi.approve(selected.value.approval.id,comment.value);else await operationsApi.reject(selected.value.approval.id,comment.value);dialog.value=false;ElMessage.success(decision.value==='APPROVED'?'建议已批准，仅更新系统状态':'建议已驳回');await load()}catch(e:any){error.value=e.response?.data?.message||'审批决策失败'}finally{deciding.value=false}}
onMounted(load)
</script>

<template><AppShell>
  <div class="page-heading"><div><span class="eyebrow">HUMAN APPROVAL</span><h2>审批中心</h2><p>批准只更新建议状态，不会调用 Meta、Google 或 TikTok。</p></div><el-button :loading="loading" @click="load">刷新</el-button></div>
  <el-alert v-if="error" :title="error" type="error" show-icon/>
  <el-alert v-if="!canDecide" title="当前角色可以查看审批，但只有 MANAGER 或 ADMIN 可以决策" type="info" show-icon/>
  <section class="filter-bar" aria-label="审批筛选"><label>状态<el-select v-model="status" clearable placeholder="全部状态" @change="load"><el-option label="待审批" value="PENDING"/><el-option label="已批准" value="APPROVED"/><el-option label="已驳回" value="REJECTED"/></el-select></label><span>{{rows.length}} 个审批请求</span></section>
  <div v-if="loading" class="loading-panel">正在加载审批请求…</div>
  <el-table v-else-if="rows.length" :data="rows" class="data-table">
    <el-table-column prop="campaign_name" label="广告计划" min-width="150"/>
    <el-table-column label="动作" min-width="160"><template #default="s"><strong>{{s.row.approval.action}}</strong><small class="table-subtitle">{{s.row.approval.suggested_value||'无数值变更'}}</small></template></el-table-column>
    <el-table-column label="审批理由" min-width="300"><template #default="s">{{s.row.approval.reason||s.row.recommendation.description}}</template></el-table-column>
    <el-table-column label="风险" width="90"><template #default="s"><el-tag type="danger">{{s.row.approval.risk_level}}</el-tag></template></el-table-column>
    <el-table-column label="状态" width="110"><template #default="s"><el-tag :type="statusType(s.row.approval.status)">{{s.row.approval.status}}</el-tag></template></el-table-column>
    <el-table-column label="决策" width="180"><template #default="s"><div v-if="canDecide&&s.row.approval.status==='PENDING'" class="row-actions"><el-button size="small" type="success" @click="open(s.row,'APPROVED')">批准</el-button><el-button size="small" type="danger" plain @click="open(s.row,'REJECTED')">驳回</el-button></div><span v-else>{{s.row.approval.decision_comment||'—'}}</span></template></el-table-column>
  </el-table>
  <el-empty v-else description="当前没有审批请求"/>
  <el-dialog v-model="dialog" :title="dialogTitle" width="520px"><div v-if="selected" class="decision-summary"><strong>{{selected.approval.action}}</strong><p>{{selected.approval.reason}}</p><el-alert title="该操作只改变建议与审批状态，不触发广告平台调用" type="warning" show-icon/></div><label class="dialog-field">决策意见<el-input v-model="comment" type="textarea" :rows="4" maxlength="1000" show-word-limit :placeholder="decision==='REJECTED'?'驳回原因（必填）':'批准说明（可选）'"/></label><template #footer><el-button @click="dialog=false">取消</el-button><el-button :type="decision==='APPROVED'?'success':'danger'" :loading="deciding" @click="submit">{{decision==='APPROVED'?'确认批准':'确认驳回'}}</el-button></template></el-dialog>
</AppShell></template>
