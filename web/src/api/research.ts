import { apiClient } from './client'
import type { ApiEnvelope } from '@/types/api'
import type { ResearchSource, ResearchSourceInput } from '@/types/research'

export const researchApi = {
  list: async () => {
    const { data } = await apiClient.get<ApiEnvelope<ResearchSource[]>>('/research/sources')
    return data.data
  },
  create: async (payload: ResearchSourceInput) => {
    const { data } = await apiClient.post<ApiEnvelope<ResearchSource>>('/research/sources', payload)
    return data.data
  },
  verify: async (id: string, comment = '') => {
    const { data } = await apiClient.post<ApiEnvelope<ResearchSource>>(`/research/sources/${id}/verify`, { comment })
    return data.data
  },
  reject: async (id: string, comment: string) => {
    const { data } = await apiClient.post<ApiEnvelope<ResearchSource>>(`/research/sources/${id}/reject`, { comment })
    return data.data
  },
}
