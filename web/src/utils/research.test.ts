import { describe, expect, it } from 'vitest'
import { canReviewResearchSource, validateResearchSourceInput } from './research'
import type { ResearchSource } from '@/types/research'

const pending = { status: 'PENDING' } as ResearchSource

describe('research source policy', () => {
  it('accepts complete HTTPS sources and rejects credentialed or HTTP URLs', () => {
    const base = { category: 'MARKET' as const, title: 'Market', summary: 'Summary', publisher: 'Publisher', published_at: '2026-08-05' }
    expect(validateResearchSourceInput({ ...base, source_url: 'https://example.com/report' })).toBe(true)
    expect(validateResearchSourceInput({ ...base, source_url: 'http://example.com/report' })).toBe(false)
    expect(validateResearchSourceInput({ ...base, source_url: 'https://user:secret@example.com/report' })).toBe(false)
  })

  it('allows only managers and admins to review pending sources', () => {
    expect(canReviewResearchSource(['MANAGER'], pending)).toBe(true)
    expect(canReviewResearchSource(['ANALYST'], pending)).toBe(false)
    expect(canReviewResearchSource(['ADMIN'], { ...pending, status: 'VERIFIED' })).toBe(false)
  })
})
