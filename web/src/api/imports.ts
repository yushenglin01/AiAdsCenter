import { apiClient } from './client'
import type { ApiEnvelope } from '@/types/api'
import type { ImportJob } from '@/types/catalog'

export async function listImports(): Promise<ImportJob[]> { const { data } = await apiClient.get<ApiEnvelope<ImportJob[]>>('/imports'); return data.data }
export async function uploadImport(endpoint: string, gameID: string, source: string, file: File): Promise<ImportJob> {
  const form = new FormData(); form.append('game_id', gameID); form.append('source', source); form.append('file', file)
  const { data } = await apiClient.post<ApiEnvelope<ImportJob>>(`/imports/${endpoint}`, form, { headers: { 'Content-Type': 'multipart/form-data' }, timeout: 30_000 })
  return data.data
}
