<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { resendVerification, verifyEmail } from '@/api/auth'
import BrandGlyph from '@/components/BrandGlyph.vue'

const route = useRoute()
const state = ref<'loading' | 'success' | 'error'>('loading')
const message = ref('正在确认你的公司邮箱…')
const email = ref('')
const resending = ref(false)
const resendMessage = ref('')

function messageOf(cause: any) { return cause.response?.data?.message || '验证失败，请检查链接或重新发送确认邮件。' }

async function resend() {
  if (!email.value.trim()) { resendMessage.value = '请输入申请时使用的公司邮箱。'; return }
  resending.value = true
  try {
    const result = await resendVerification(email.value.trim())
    resendMessage.value = result.message
  } catch (cause) {
    resendMessage.value = messageOf(cause)
  } finally {
    resending.value = false
  }
}

onMounted(async () => {
  const token = typeof route.query.token === 'string' ? route.query.token : ''
  if (!token) { state.value = 'error'; message.value = '验证链接缺少必要 Token。'; return }
  try {
    const result = await verifyEmail(token)
    state.value = 'success'
    message.value = result.message
  } catch (cause) {
    state.value = 'error'
    message.value = messageOf(cause)
  }
})
</script>

<template>
  <main class="verification-page">
    <section class="verification-card">
      <header class="verification-brand"><BrandGlyph /><div><strong>AdNova</strong><span>企业成员认证</span></div></header>
      <div v-if="state === 'loading'" class="verification-state"><span class="verification-spinner"></span><h1>正在确认邮箱</h1><p>{{ message }}</p></div>
      <el-result v-else-if="state === 'success'" icon="success" title="公司邮箱已确认" :sub-title="message"><template #extra><router-link class="verification-link" to="/login">返回登录页</router-link></template></el-result>
      <template v-else>
        <el-result icon="error" title="邮箱确认未完成" :sub-title="message" />
        <div class="resend-panel"><label for="resend-email">重新发送确认邮件</label><div><el-input id="resend-email" v-model="email" autocomplete="email" placeholder="申请时使用的公司邮箱" @keyup.enter="resend" /><el-button :loading="resending" @click="resend">发送</el-button></div><p v-if="resendMessage" aria-live="polite">{{ resendMessage }}</p></div>
        <router-link class="verification-link" to="/login">返回登录页</router-link>
      </template>
    </section>
  </main>
</template>
