<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Lock, User } from '@element-plus/icons-vue'
import { ElIcon } from 'element-plus/es/components/icon/index'
import { useAuthStore } from '@/stores/auth'
import BrandGlyph from '@/components/BrandGlyph.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const errorMessage = ref('')
const form = reactive({ username: '', password: '' })

async function submit() {
  errorMessage.value = ''
  try {
    await auth.signIn(form)
    await router.replace(typeof route.query.redirect === 'string' ? route.query.redirect : '/')
  } catch (error: any) {
    errorMessage.value = error.response?.data?.message || '登录失败，请检查账号和密码。'
  }
}
</script>

<template>
  <main class="login-page">
    <section class="login-stage">
      <header class="login-brand">
        <BrandGlyph />
        <div><strong>AdNova</strong><span>星曜智投</span></div>
      </header>

      <div class="login-orbit" aria-hidden="true">
        <div class="orbit-ring orbit-ring--outer"><i></i><i></i></div>
        <div class="orbit-ring orbit-ring--middle"><i></i></div>
        <div class="orbit-ring orbit-ring--inner"></div>
        <div class="orbit-core"><small>AI SIGNAL</small><strong>LIVE</strong><span>09:42:18 UTC</span></div>
        <div class="orbit-coordinate orbit-coordinate--a">META / US</div>
        <div class="orbit-coordinate orbit-coordinate--b">ROAS 1.84</div>
        <div class="orbit-coordinate orbit-coordinate--c">MMP SYNC</div>
      </div>

      <div class="login-story">
        <span class="eyebrow">GLOBAL AD INTELLIGENCE</span>
        <h1>让每一笔投放，<br /><em>穿越数据噪声。</em></h1>
        <p>汇聚渠道、归因、收入与素材信号，把复杂经营数据转化为可验证、可审批的决策依据。</p>
        <div class="boundary"><span><i></i>确定性指标</span><span><i></i>AI 经营解释</span><span><i></i>人工审批边界</span></div>
      </div>

      <footer class="signal-rail">
        <span><i></i> SYSTEM ONLINE</span>
        <span>APPSFLYER CONNECTED</span>
        <span>PHASE 12</span>
      </footer>
    </section>

    <section class="login-access">
      <div class="login-panel" aria-labelledby="login-title">
        <div class="access-heading">
          <span class="eyebrow">COMMAND ACCESS</span>
          <h2 id="login-title">进入指挥舱</h2>
          <p>仅限已完成邮箱确认并通过管理员授权的内部成员</p>
        </div>
        <el-alert v-if="errorMessage" :title="errorMessage" type="error" show-icon :closable="false" />
        <el-form label-position="top" @submit.prevent="submit">
          <el-form-item label="邮箱或用户名">
            <el-input v-model="form.username" size="large" autocomplete="username" placeholder="输入公司邮箱或用户名">
              <template #prefix><el-icon><User /></el-icon></template>
            </el-input>
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="form.password" size="large" type="password" show-password autocomplete="current-password" placeholder="输入密码" @keyup.enter="submit">
              <template #prefix><el-icon><Lock /></el-icon></template>
            </el-input>
          </el-form-item>
          <el-button class="submit-button" type="primary" :loading="auth.loading" @click="submit">
            进入 AdNova <span aria-hidden="true">↗</span>
          </el-button>
        </el-form>
        <div class="auth-switch">还没有内部成员账号？<router-link to="/register">提交注册申请</router-link></div>
        <p class="security-note"><span aria-hidden="true">◇</span> 只生成建议与审批单，不直接操作广告平台</p>
      </div>
      <p class="access-footnote">AdNova Intelligence System · v1.2.0</p>
    </section>
  </main>
</template>
