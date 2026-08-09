import { apiClient } from './client'
import type {
  ApiEnvelope,
  CurrentUser,
  RegistrationAccepted,
  RegistrationApplication,
  RegistrationConfig,
  RegistrationPayload,
  RoleOption,
  Tenant,
  TokenPair,
} from '@/types/api'

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

export async function getRegistrationConfig(): Promise<RegistrationConfig> {
  const { data } = await apiClient.get<ApiEnvelope<RegistrationConfig>>('/auth/registration-config')
  return data.data
}

export async function registerMember(payload: RegistrationPayload): Promise<RegistrationAccepted> {
  const { data } = await apiClient.post<ApiEnvelope<RegistrationAccepted>>('/auth/register', payload)
  return data.data
}

export async function verifyEmail(token: string): Promise<RegistrationAccepted> {
  const { data } = await apiClient.post<ApiEnvelope<RegistrationAccepted>>('/auth/verify-email', { token })
  return data.data
}

export async function resendVerification(email: string): Promise<RegistrationAccepted> {
  const { data } = await apiClient.post<ApiEnvelope<RegistrationAccepted>>('/auth/resend-verification', { email })
  return data.data
}

export async function listRegistrationApplications(status = ''): Promise<RegistrationApplication[]> {
  const { data } = await apiClient.get<ApiEnvelope<RegistrationApplication[]>>('/admin/registration-applications', { params: status ? { status } : {} })
  return data.data
}

export async function listRoles(): Promise<RoleOption[]> {
  const { data } = await apiClient.get<ApiEnvelope<RoleOption[]>>('/admin/roles')
  return data.data
}

export async function approveRegistration(id: string, roles: string[]): Promise<RegistrationApplication> {
  const { data } = await apiClient.post<ApiEnvelope<RegistrationApplication>>(`/admin/registration-applications/${id}/approve`, { roles })
  return data.data
}

export async function rejectRegistration(id: string, reason: string): Promise<RegistrationApplication> {
  const { data } = await apiClient.post<ApiEnvelope<RegistrationApplication>>(`/admin/registration-applications/${id}/reject`, { reason })
  return data.data
}
