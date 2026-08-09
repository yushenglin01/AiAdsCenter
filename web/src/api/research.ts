import { apiClient } from './client'
import type { ApiEnvelope } from '@/types/api'
import type { ResearchSource, ResearchSourceInput, WebImportInput, WebSearchCapability, WebSearchInput, WebSearchResponse } from '@/types/research'

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
  webCapability: async () => {
    const { data } = await apiClient.get<ApiEnvelope<WebSearchCapability>>('/research/web-search/capability')
    return data.data
  },
  searchWeb: async (payload: WebSearchInput) => {
    const { data } = await apiClient.post<ApiEnvelope<WebSearchResponse>>('/research/web-search', payload, { timeout: 15_000 })
    return data.data
  },
  importWebResult: async (payload: WebImportInput) => {
    const { data } = await apiClient.post<ApiEnvelope<ResearchSource>>('/research/web-search/import', payload)
    return data.data
  },
}
