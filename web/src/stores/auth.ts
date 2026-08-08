import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import * as authApi from '@/api/auth'
import type { CurrentUser, Tenant } from '@/types/api'

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref(localStorage.getItem('gai_access_token'))
  const user = ref<CurrentUser | null>(null)
  const tenant = ref<Tenant | null>(null)
  const loading = ref(false)
  const initialized = ref(false)
  const isAuthenticated = computed(() => Boolean(accessToken.value))

  async function signIn(payload: authApi.LoginPayload) {
    loading.value = true
    try {
      const tokens = await authApi.login(payload)
      localStorage.setItem('gai_access_token', tokens.access_token)
      localStorage.setItem('gai_refresh_token', tokens.refresh_token)
      accessToken.value = tokens.access_token
      await loadSession()
    } finally {
      loading.value = false
    }
  }

  async function loadSession() {
    if (!accessToken.value) { initialized.value = true; return }
    loading.value = true
    try {
      ;[user.value, tenant.value] = await Promise.all([authApi.getCurrentUser(), authApi.getCurrentTenant()])
    } finally {
      loading.value = false
      initialized.value = true
    }
  }

  function signOut() {
    localStorage.removeItem('gai_access_token')
    localStorage.removeItem('gai_refresh_token')
    accessToken.value = null
    user.value = null
    tenant.value = null
  }

  function hasAnyRole(roles: string[]) { return roles.some((role) => user.value?.roles.includes(role)) }

  return { accessToken, user, tenant, loading, initialized, isAuthenticated, signIn, loadSession, signOut, hasAnyRole }
})
