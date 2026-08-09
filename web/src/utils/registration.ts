import type { RegistrationPayload } from '@/types/api'

export const usernamePattern = /^[a-z0-9][a-z0-9._-]{2,79}$/

export function passwordRequirements(password: string) {
  return {
    length: password.length >= 12,
    upper: /[A-Z]/.test(password),
    lower: /[a-z]/.test(password),
    digit: /\d/.test(password),
    symbol: /[^A-Za-z0-9]/.test(password),
  }
}

export function validateRegistration(payload: RegistrationPayload, confirmation: string): string {
  if (!usernamePattern.test(payload.username)) return '用户名需为 3–80 位小写字母、数字、点、下划线或连字符。'
  if (!payload.display_name.trim() || !payload.department.trim()) return '请填写姓名和所属部门。'
  if (!/^\S+@\S+\.\S+$/.test(payload.email)) return '请输入有效的公司邮箱。'
  if (!Object.values(passwordRequirements(payload.password)).every(Boolean)) return '密码至少 12 位，并包含大小写字母、数字和特殊字符。'
  if (payload.password !== confirmation) return '两次输入的密码不一致。'
  return ''
}
