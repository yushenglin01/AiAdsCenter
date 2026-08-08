import { describe, expect, it } from 'vitest'
import { passwordRequirements, validateRegistration } from './registration'

const valid = {
  username: 'alice.chen', email: 'alice@example.com', display_name: '陈晓', department: '市场部',
  job_title: '投放分析师', password: 'SecurePass!2026',
}

describe('member registration validation', () => {
  it('accepts a complete internal member application', () => {
    expect(validateRegistration(valid, valid.password)).toBe('')
  })

  it('requires a strong password and matching confirmation', () => {
    expect(Object.values(passwordRequirements('weak-password')).every(Boolean)).toBe(false)
    expect(validateRegistration({ ...valid, password: 'weak-password' }, 'weak-password')).toContain('密码至少')
    expect(validateRegistration(valid, 'different')).toContain('不一致')
  })
})
