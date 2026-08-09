<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import AppShell from '@/components/AppShell.vue'
import {
  approveRegistration,
  listRegistrationApplications,
  listRoles,
  rejectRegistration,
} from '@/api/auth'
import type { RegistrationApplication, RoleOption } from '@/types/api'

const rows = ref<RegistrationApplication[]>([])
const roles = ref<RoleOption[]>([])
const status = ref('PENDING_APPROVAL')
const loading = ref(false)
const error = ref('')
const approving = ref(false)
const dialogVisible = ref(false)
const selected = ref<RegistrationApplication | null>(null)
const selectedRoles = ref<string[]>(['VIEWER'])
const pendingCount = computed(() => rows.value.filter((item) => item.status === 'PENDING_APPROVAL').length)

const statusOptions = [
  { label: '等待授权', value: 'PENDING_APPROVAL' },
  { label: '等待邮箱确认', value: 'PENDING_EMAIL' },
  { label: '已启用', value: 'ACTIVE' },
  { label: '已驳回', value: 'REJECTED' },
  { label: '已停用', value: 'DISABLED' },
  { label: '全部申请', value: '' },
]

function messageOf(cause: any) { return cause.response?.data?.message || '请求失败，请稍后重试。' }
function statusLabel(value: string) { return statusOptions.find((item) => item.value === value)?.label || value }
function statusType(value: string) { return value === 'ACTIVE' ? 'success' : value === 'REJECTED' || value === 'DISABLED' ? 'danger' : 'warning' }
function formatDate(value?: string) { return value ? new Date(value).toLocaleString() : '—' }

async function load() {
  loading.value = true
  error.value = ''
  try {
    ;[rows.value, roles.value] = await Promise.all([listRegistrationApplications(status.value), listRoles()])
  } catch (cause) {
    error.value = messageOf(cause)
  } finally {
    loading.value = false
  }
}

function openApproval(item: RegistrationApplication) {
  selected.value = item
  selectedRoles.value = item.roles.length ? [...item.roles] : ['VIEWER']
  dialogVisible.value = true
}

async function approve() {
  if (!selected.value || !selectedRoles.value.length) {
    ElMessage.warning('请至少分配一个角色')
    return
  }
  approving.value = true
  try {
    await approveRegistration(selected.value.id, selectedRoles.value)
    ElMessage.success(`${selected.value.display_name} 已获授权，可以登录`)
    dialogVisible.value = false
    await load()
  } catch (cause) {
    ElMessage.error(messageOf(cause))
  } finally {
    approving.value = false
  }
}

async function reject(item: RegistrationApplication) {
  try {
    const { value } = await ElMessageBox.prompt('请填写驳回原因，便于内部成员联系管理员处理', '驳回成员申请', {
      confirmButtonText: '确认驳回',
      cancelButtonText: '取消',
      inputValidator: (text) => Boolean(text && text.trim().length >= 2) || '驳回原因至少 2 个字符',
    })
    await rejectRegistration(item.id, value.trim())
    ElMessage.success('申请已驳回')
    await load()
  } catch (cause: any) {
    if (cause !== 'cancel' && cause !== 'close') ElMessage.error(messageOf(cause))
  }
}

onMounted(load)
</script>

<template>
  <AppShell>
    <div class="page-heading">
      <div><span class="eyebrow">MEMBER ACCESS</span><h2>内部成员管理</h2><p>公司邮箱确认后，由管理员核验身份并分配最小必要角色。</p></div>
      <div class="member-admin-actions"><el-tag type="warning">{{ pendingCount }} 个待授权</el-tag><el-button @click="load">刷新</el-button></div>
    </div>
    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
    <section class="member-filter" aria-label="成员申请筛选">
      <label>申请状态<el-select v-model="status" @change="load"><el-option v-for="item in statusOptions" :key="item.label" :label="item.label" :value="item.value" /></el-select></label>
    </section>
    <div v-if="loading" class="loading-panel">正在加载成员申请…</div>
    <el-empty v-else-if="!rows.length" description="当前筛选条件下没有成员申请" />
    <el-table v-else :data="rows" class="data-table">
      <el-table-column label="申请人" min-width="190"><template #default="scope"><strong>{{ scope.row.display_name }}</strong><small class="table-subtitle">@{{ scope.row.username }}</small></template></el-table-column>
      <el-table-column label="公司信息" min-width="220"><template #default="scope">{{ scope.row.department }}<small class="table-subtitle">{{ scope.row.job_title || '未填写职位' }}</small></template></el-table-column>
      <el-table-column prop="email" label="公司邮箱" min-width="230" />
      <el-table-column label="状态" width="130"><template #default="scope"><el-tag :type="statusType(scope.row.status)">{{ statusLabel(scope.row.status) }}</el-tag></template></el-table-column>
      <el-table-column label="邮箱确认" width="180"><template #default="scope">{{ formatDate(scope.row.email_verified_at) }}</template></el-table-column>
      <el-table-column label="申请时间" width="180"><template #default="scope">{{ formatDate(scope.row.created_at) }}</template></el-table-column>
      <el-table-column label="角色 / 原因" min-width="180"><template #default="scope">{{ scope.row.roles.join(' · ') || scope.row.rejection_reason || '—' }}</template></el-table-column>
      <el-table-column label="操作" width="170" fixed="right"><template #default="scope"><template v-if="scope.row.status === 'PENDING_APPROVAL'"><el-button link type="primary" @click="openApproval(scope.row)">授权</el-button><el-button link type="danger" @click="reject(scope.row)">驳回</el-button></template><span v-else>—</span></template></el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" title="授权内部成员" width="520px">
      <div v-if="selected" class="member-approval-summary"><strong>{{ selected.display_name }}</strong><span>{{ selected.email }}</span><small>{{ selected.department }} · {{ selected.job_title || '未填写职位' }}</small></div>
      <el-form label-position="top">
        <el-form-item label="分配角色（遵循最小权限原则）">
          <el-select v-model="selectedRoles" multiple placeholder="选择角色">
            <el-option v-for="role in roles" :key="role.code" :label="`${role.name} · ${role.code}`" :value="role.code" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer><el-button @click="dialogVisible = false">取消</el-button><el-button type="primary" :loading="approving" @click="approve">确认授权</el-button></template>
    </el-dialog>
  </AppShell>
</template>
