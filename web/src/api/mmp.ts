import { apiClient } from './client'
import type { ApiEnvelope } from '@/types/api'
import type { MMPConnection, MMPSyncRun } from '@/types/catalog'

export type MMPProvider = 'APPSFLYER' | 'ADJUST'
export interface ConfigureMMPPayload { game_id: string; external_app_id: string; status: 'ACTIVE' | 'DISABLED' }

export async function listMMPConnections(): Promise<MMPConnection[]> {
  const { data } = await apiClient.get<ApiEnvelope<MMPConnection[]>>('/mmp-connections')
  return data.data
}

export async function configureMMPConnection(provider: MMPProvider, payload: ConfigureMMPPayload): Promise<MMPConnection> {
	const { data } = await apiClient.put<ApiEnvelope<MMPConnection>>(`/mmp-connections/${provider.toLowerCase()}`, payload)
  return data.data
}

export async function listMMPSyncRuns(): Promise<MMPSyncRun[]> {
  const { data } = await apiClient.get<ApiEnvelope<MMPSyncRun[]>>('/mmp-sync-runs')
  return data.data
}

export async function syncMMPConnection(id: string, from: string, to: string): Promise<MMPSyncRun> {
  const { data } = await apiClient.post<ApiEnvelope<MMPSyncRun>>(`/mmp-connections/${id}/sync`, { from, to }, { timeout: 60_000 })
  return data.data
}
