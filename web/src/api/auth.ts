import { apiClient } from './client'
import type { ApiEnvelope, CurrentUser, Tenant, TokenPair } from '@/types/api'

export interface LoginPayload { username: string; password: string }

export async function login(payload: LoginPayload): Promise<TokenPair> {
  const { data } = await apiClient.post<ApiEnvelope<TokenPair>>('/auth/login', payload)
  return data.data
}

export async function getCurrentUser(): Promise<CurrentUser> {
  const { data } = await apiClient.get<ApiEnvelope<CurrentUser>>('/auth/me')
  return data.data
}

export async function getCurrentTenant(): Promise<Tenant> {
  const { data } = await apiClient.get<ApiEnvelope<Tenant>>('/tenants/current')
  return data.data
}
