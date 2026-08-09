<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Lock, Message, OfficeBuilding, User } from '@element-plus/icons-vue'
import { ElIcon } from 'element-plus/es/components/icon/index'
import { getRegistrationConfig, registerMember } from '@/api/auth'
import BrandGlyph from '@/components/BrandGlyph.vue'
import type { RegistrationConfig, RegistrationPayload } from '@/types/api'
import { passwordRequirements, validateRegistration } from '@/utils/registration'

const config = ref<RegistrationConfig>({ enabled: false, allowed_email_domains: [] })
const loadingConfig = ref(true)
const submitting = ref(false)
const error = ref('')
const success = ref('')
const confirmation = ref('')
const form = reactive<RegistrationPayload>({ username: '', email: '', display_name: '', department: '', job_title: '', password: '' })
const passwordState = computed(() => passwordRequirements(form.password))
const domainHint = computed(() => config.value.allowed_email_domains.length ? `仅接受：${config.value.allowed_email_domains.join('、')}` : '请使用公司分配的工作邮箱')

function messageOf(cause: any) { return cause.response?.data?.message || '申请提交失败，请稍后重试。' }

async function submit() {
  error.value = ''
  success.value = ''
  const validationError = validateRegistration(form, confirmation.value)
  if (validationError) { error.value = validationError; return }
  submitting.value = true
  try {
    const result = await registerMember({ ...form, username: form.username.trim().toLowerCase(), email: form.email.trim().toLowerCase() })
    success.value = result.message
  } catch (cause) {
    error.value = messageOf(cause)
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  try { config.value = await getRegistrationConfig() }
  catch (cause) { error.value = messageOf(cause) }
  finally { loadingConfig.value = false }
})
</script>

<template>
  <main class="member-auth-page">
    <section class="member-auth-brand">
      <header class="login-brand"><BrandGlyph /><div><strong>AdNova</strong><span>星曜智投</span></div></header>
      <div class="member-auth-copy"><span class="eyebrow">INTERNAL MEMBER ACCESS</span><h1>加入公司的<br /><em>投放决策工作台</em></h1><p>成员申请需要通过公司邮箱确认和管理员授权。系统不会因为注册成功自动授予任何业务权限。</p><ol><li><strong>01</strong><span>填写内部身份信息</span></li><li><strong>02</strong><span>确认公司邮箱</span></li><li><strong>03</strong><span>管理员核验并授权</span></li></ol></div>
    </section>
    <section class="member-auth-access">
      <div class="member-register-panel" aria-labelledby="register-title">
        <div class="access-heading"><span class="eyebrow">MEMBER APPLICATION</span><h2 id="register-title">内部成员注册</h2><p>{{ domainHint }}</p></div>
        <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
        <el-alert v-if="success" title="申请已受理" :description="`${success}。完成邮箱确认后，请等待管理员授权。`" type="success" show-icon :closable="false" />
        <el-alert v-if="!loadingConfig && !config.enabled" title="当前未开放成员注册，请联系管理员" type="warning" show-icon :closable="false" />
        <el-form v-if="!success && config.enabled" label-position="top" @submit.prevent="submit">
          <div class="member-form-grid">
            <el-form-item label="姓名"><el-input v-model="form.display_name" autocomplete="name" placeholder="真实姓名"><template #prefix><el-icon><User /></el-icon></template></el-input></el-form-item>
            <el-form-item label="用户名"><el-input v-model="form.username" autocomplete="username" placeholder="例如 alice.chen"><template #prefix><el-icon><User /></el-icon></template></el-input></el-form-item>
            <el-form-item label="公司邮箱"><el-input v-model="form.email" autocomplete="email" placeholder="name@company.com"><template #prefix><el-icon><Message /></el-icon></template></el-input></el-form-item>
            <el-form-item label="所属部门"><el-input v-model="form.department" autocomplete="organization" placeholder="例如 市场部"><template #prefix><el-icon><OfficeBuilding /></el-icon></template></el-input></el-form-item>
            <el-form-item label="职位（选填）"><el-input v-model="form.job_title" autocomplete="organization-title" placeholder="例如 投放优化师" /></el-form-item>
            <span></span>
            <el-form-item label="设置密码"><el-input v-model="form.password" type="password" show-password autocomplete="new-password" placeholder="至少 12 位"><template #prefix><el-icon><Lock /></el-icon></template></el-input></el-form-item>
            <el-form-item label="确认密码"><el-input v-model="confirmation" type="password" show-password autocomplete="new-password" placeholder="再次输入密码" @keyup.enter="submit"><template #prefix><el-icon><Lock /></el-icon></template></el-input></el-form-item>
          </div>
          <div class="password-checks" aria-live="polite"><span :class="{ passed: passwordState.length }">12 位以上</span><span :class="{ passed: passwordState.upper && passwordState.lower }">大小写字母</span><span :class="{ passed: passwordState.digit }">数字</span><span :class="{ passed: passwordState.symbol }">特殊字符</span></div>
          <el-button class="submit-button" type="primary" :loading="submitting" :disabled="loadingConfig" @click="submit">提交成员申请 <span aria-hidden="true">↗</span></el-button>
        </el-form>
        <div class="auth-switch"><router-link to="/login">← 返回登录</router-link></div>
      </div>
      <p class="access-footnote">内部成员申请 · 邮箱确认 · 管理员授权</p>
    </section>
  </main>
</template>
