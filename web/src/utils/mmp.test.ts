import { describe, expect, it } from 'vitest'
import { connectionHint, validateSyncRange } from './mmp'
import type { MMPConnection } from '@/types/catalog'

const base: MMPConnection = { id: '1', game_id: 'g', provider: 'APPSFLYER', external_app_id: 'app', status: 'ACTIVE', credential_configured: true, health: 'READY', created_at: '', updated_at: '' }

describe('AppsFlyer sync UI rules', () => {
  it('limits one request to seven days', () => {
    expect(validateSyncRange(['2026-08-01', '2026-08-07'])).toBe('')
    expect(validateSyncRange(['2026-08-01', '2026-08-08'])).toContain('7 天')
    expect(validateSyncRange([])).toContain('请选择')
  })

  it('explains configuration states without exposing credentials', () => {
    expect(connectionHint()).toContain('App ID')
    expect(connectionHint({ ...base, credential_configured: false, health: 'NOT_CONFIGURED' })).toContain('服务端')
    expect(connectionHint(base)).toContain('已就绪')
  })
})
