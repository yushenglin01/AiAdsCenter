<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import AppShell from '@/components/AppShell.vue'
import { catalogApi, type GameInput } from '@/api/catalog'
import { useAuthStore } from '@/stores/auth'
import type { Game } from '@/types/catalog'

const auth = useAuthStore()
const rows = ref<Game[]>([]); const loading = ref(false); const saving = ref(false); const error = ref(''); const dialog = ref(false); const editingID = ref('')
const form = reactive<GameInput>({ code: '', name: '', package_name: '', timezone: 'UTC', currency: 'USD', status: 'ACTIVE' })
const canManage = () => auth.hasAnyRole(['ADMIN', 'MANAGER'])
function messageOf(value: any) { return value.response?.data?.message || '请求失败，请稍后重试。' }
async function load() { loading.value = true; error.value = ''; try { rows.value = await catalogApi.listGames() } catch (e) { error.value = messageOf(e) } finally { loading.value = false } }
function open(row?: Game) { editingID.value = row?.id || ''; Object.assign(form, row ? { code: row.code, name: row.name, package_name: row.package_name, timezone: row.timezone, currency: row.currency, status: row.status } : { code: '', name: '', package_name: '', timezone: 'UTC', currency: 'USD', status: 'ACTIVE' }); dialog.value = true }
async function save() { if (!form.code.trim() || !form.name.trim()) { error.value = '游戏编码和名称不能为空。'; return }; saving.value = true; try { if (editingID.value) await catalogApi.updateGame(editingID.value, form); else await catalogApi.createGame(form); dialog.value = false; await load() } catch (e) { error.value = messageOf(e) } finally { saving.value = false } }
onMounted(load)
</script>

<template><AppShell>
  <div class="page-heading"><div><span class="eyebrow">FOUNDATION DATA</span><h2>游戏管理</h2><p>维护游戏基础信息、时区和记账币种。</p></div><el-button v-if="canManage()" type="primary" @click="open()">新增游戏</el-button></div>
  <el-alert v-if="error" :title="error" type="error" show-icon closable @close="error=''" />
  <div v-if="loading" class="loading-panel">正在加载游戏…</div>
  <el-empty v-else-if="!rows.length" description="暂无游戏项目" />
  <el-table v-else :data="rows" class="data-table">
    <el-table-column prop="code" label="游戏编码" /><el-table-column prop="name" label="名称" /><el-table-column prop="package_name" label="包名" min-width="220" /><el-table-column prop="timezone" label="时区" /><el-table-column prop="currency" label="币种" width="90" /><el-table-column label="状态" width="100"><template #default="scope"><el-tag :type="scope.row.status === 'ACTIVE' ? 'success' : 'info'">{{ scope.row.status }}</el-tag></template></el-table-column><el-table-column v-if="canManage()" label="操作" width="90"><template #default="scope"><el-button link @click="open(scope.row)">编辑</el-button></template></el-table-column>
  </el-table>
  <el-dialog v-model="dialog" :title="editingID ? '编辑游戏' : '新增游戏'" width="520px">
    <el-form label-position="top"><el-form-item label="游戏编码"><el-input v-model="form.code" /></el-form-item><el-form-item label="名称"><el-input v-model="form.name" /></el-form-item><el-form-item label="包名"><el-input v-model="form.package_name" /></el-form-item><div class="form-grid"><el-form-item label="时区"><el-input v-model="form.timezone" /></el-form-item><el-form-item label="币种"><el-input v-model="form.currency" maxlength="3" /></el-form-item></div><el-form-item label="状态"><el-select v-model="form.status"><el-option label="ACTIVE" value="ACTIVE" /><el-option label="INACTIVE" value="INACTIVE" /></el-select></el-form-item></el-form>
    <template #footer><el-button @click="dialog=false">取消</el-button><el-button type="primary" :loading="saving" @click="save">保存</el-button></template>
  </el-dialog>
</AppShell></template>
