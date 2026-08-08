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
  display_name: string
  roles: string[]
}

export interface Tenant {
  id: string
  slug: string
  name: string
}
