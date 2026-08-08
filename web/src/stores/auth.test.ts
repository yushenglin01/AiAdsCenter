import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAuthStore } from './auth'

vi.mock('@/api/auth', () => ({
  login: vi.fn(async () => ({ access_token: 'token', refresh_token: 'refresh', token_type: 'Bearer', expires_in: 900 })),
  getCurrentUser: vi.fn(async () => ({ id: 'u1', tenant_id: 't1', username: 'manager', display_name: 'Manager', roles: ['MANAGER'] })),
  getCurrentTenant: vi.fn(async () => ({ id: 't1', slug: 'demo-company', name: 'Demo Company' })),
}))

describe('auth store', () => {
  beforeEach(() => {
    const values = new Map<string, string>()
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => values.get(key) ?? null,
      setItem: (key: string, value: string) => values.set(key, value),
      removeItem: (key: string) => values.delete(key),
      clear: () => values.clear(),
    })
    setActivePinia(createPinia())
  })
  it('loads tenant-bound session and evaluates roles', async () => {
    const store = useAuthStore()
	await store.signIn({ username: 'manager', password: 'Demo@123456' })
    expect(store.tenant?.slug).toBe('demo-company')
    expect(store.hasAnyRole(['MANAGER'])).toBe(true)
    expect(store.hasAnyRole(['ADMIN'])).toBe(false)
  })
})
