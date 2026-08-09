export interface ApiEnvelope<T> {
  code: number
  message: string
  data: T
  request_id: string
}

export interface TokenPair {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
}

export interface CurrentUser {
  id: string
  tenant_id: string
  username: string
  email: string
  display_name: string
  roles: string[]
}

export type RegistrationStatus = 'PENDING_EMAIL' | 'PENDING_APPROVAL' | 'ACTIVE' | 'REJECTED' | 'DISABLED'

export interface RegistrationConfig {
  enabled: boolean
  allowed_email_domains: string[]
}

export interface RegistrationAccepted {
  status: RegistrationStatus
  message: string
}

export interface RegistrationPayload {
  username: string
  email: string
  display_name: string
  department: string
  job_title: string
  password: string
}

export interface RegistrationApplication {
  id: string
  username: string
  email: string
  display_name: string
  department: string
  job_title: string
  status: RegistrationStatus
  roles: string[]
  email_verified_at?: string
  approved_at?: string
  approved_by?: string
  rejected_at?: string
  rejected_by?: string
  rejection_reason?: string
  created_at: string
}

export interface RoleOption {
  code: string
  name: string
  description: string
}

export interface Tenant {
  id: string
  slug: string
  name: string
}
