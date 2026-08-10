import { describe, expect, it } from 'vitest'
import { canSyncConnection, connectionHealthLabel, connectionHint, validateSyncRange } from './mmp'
import type { MMPConnection } from '@/types/catalog'

const base: MMPConnection = { id: '1', game_id: 'g', provider: 'APPSFLYER', external_app_id: 'app', status: 'ACTIVE', credential_configured: true, health: 'READY', created_at: '', updated_at: '' }

describe('MMP sync UI rules', () => {
  it('limits one request to seven days', () => {
    expect(validateSyncRange(['2026-08-01', '2026-08-07'])).toBe('')
    expect(validateSyncRange(['2026-08-01', '2026-08-08'])).toContain('7 天')
    expect(validateSyncRange([])).toContain('请选择')
  })

  it('explains configuration states without exposing credentials', () => {
    expect(connectionHint('APPSFLYER')).toContain('App ID')
    expect(connectionHint('APPSFLYER', { ...base, credential_configured: false, health: 'NOT_CONFIGURED' })).toContain('服务端')
    expect(connectionHint('APPSFLYER', { ...base, health: 'UNVERIFIED' })).toContain('首次同步')
    expect(connectionHint('APPSFLYER', base)).toContain('同步成功')
    expect(connectionHint('ADJUST')).toContain('App Token')
  })

  it('allows the first verification sync without claiming the connection is ready', () => {
    expect(canSyncConnection({ ...base, health: 'UNVERIFIED' })).toBe(true)
    expect(canSyncConnection({ ...base, credential_configured: false, health: 'NOT_CONFIGURED' })).toBe(false)
    expect(canSyncConnection({ ...base, status: 'DISABLED', health: 'DISABLED' })).toBe(false)
    expect(connectionHealthLabel('UNVERIFIED')).toBe('待验证')
  })
})
