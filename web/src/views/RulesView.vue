<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElInputNumber } from 'element-plus/es/components/input-number/index'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElSwitch } from 'element-plus/es/components/switch/index'
import 'element-plus/es/components/input-number/style/css'
import 'element-plus/es/components/switch/style/css'
import AppShell from '@/components/AppShell.vue'
import { metricsApi } from '@/api/metrics'
import { useAuthStore } from '@/stores/auth'
import type { Rule, RuleFinding } from '@/types/metrics'
const auth=useAuthStore();const rules=ref<Rule[]>([]);const findings=ref<RuleFinding[]>([]);const loading=ref(true);const error=ref('');const saving=ref('')
const canEdit=auth.hasAnyRole(['ADMIN','MANAGER'])
async function load(){loading.value=true;try{const data=await metricsApi.rules();rules.value=data.rules.map((row)=>({...row,threshold:Number(row.threshold)}));findings.value=data.findings}catch(e:any){error.value=e.response?.data?.message||'规则加载失败'}finally{loading.value=false}}
async function save(row:Rule){saving.value=row.id;try{await metricsApi.updateRule(row.id,{threshold:row.threshold,consecutive_days:row.consecutive_days,enabled:row.enabled});ElMessage.success(`${row.name} 已更新`)}catch(e:any){error.value=e.response?.data?.message||'规则保存失败';await load()}finally{saving.value=''}}
const category=(v:string)=>({BUSINESS:'经营',ATTRIBUTION:'归因',CREATIVE:'素材'}[v]||v)
onMounted(load)
</script>
<template><AppShell><div class="page-heading"><div><span class="eyebrow">RULE CENTER</span><h2>分析规则</h2><p>规则由代码执行，阈值和连续天数可配置，修改后需重新执行分析。</p></div></div><el-alert v-if="error" :title="error" type="error" show-icon/><div v-if="loading" class="loading-panel">正在加载规则配置…</div><template v-else><el-table :data="rules"><el-table-column label="分类" width="90"><template #default="s">{{category(s.row.category)}}</template></el-table-column><el-table-column prop="name" label="规则" min-width="190"/><el-table-column prop="code" label="编码" min-width="210"/><el-table-column prop="severity" label="等级" width="90"/><el-table-column label="阈值" width="150"><template #default="s"><el-input-number v-model="s.row.threshold" :disabled="!canEdit" :precision="3" :step="0.01" size="small"/></template></el-table-column><el-table-column label="连续天数" width="130"><template #default="s"><el-input-number v-model="s.row.consecutive_days" :disabled="!canEdit" :min="1" :max="30" size="small"/></template></el-table-column><el-table-column label="启用" width="80"><template #default="s"><el-switch v-model="s.row.enabled" :disabled="!canEdit"/></template></el-table-column><el-table-column v-if="canEdit" label="操作" width="90"><template #default="s"><el-button link type="primary" :loading="saving===s.row.id" @click="save(s.row)">保存</el-button></template></el-table-column></el-table><div class="section-title"><h3>经营风险发现</h3><span>{{findings.length}} 条</span></div><el-empty v-if="!findings.length" description="当前没有经营规则命中"/><el-table v-else :data="findings"><el-table-column prop="severity" label="等级" width="90"/><el-table-column prop="rule_code" label="规则" min-width="210"/><el-table-column prop="title" label="结论" min-width="190"/><el-table-column prop="description" label="证据说明" min-width="420"/></el-table></template></AppShell></template>
