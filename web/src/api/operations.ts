import { apiClient } from './client'
import type { ApiEnvelope } from '@/types/api'
import type { ApprovalDetail, AuditLog, ModelUsage, ModelUsageSummary, OperationsSummary, RecommendationDetail } from '@/types/operations'

async function get<T>(path: string, params?: Record<string, string | number>): Promise<T> {
  const { data } = await apiClient.get<ApiEnvelope<T>>(path, { params })
  return data.data
}

export const operationsApi = {
  recommendations: (params: Record<string, string> = {}) => get<RecommendationDetail[]>('/recommendations', params),
  recommendation: (id: string) => get<RecommendationDetail>(`/recommendations/${id}`),
  approvals: (params: Record<string, string> = {}) => get<ApprovalDetail[]>('/approvals', params),
  approval: (id: string) => get<ApprovalDetail>(`/approvals/${id}`),
  approve: async (id: string, comment: string) => { const { data } = await apiClient.post<ApiEnvelope<ApprovalDetail>>(`/approvals/${id}/approve`, { comment }); return data.data },
  reject: async (id: string, comment: string) => { const { data } = await apiClient.post<ApiEnvelope<ApprovalDetail>>(`/approvals/${id}/reject`, { comment }); return data.data },
  auditLogs: (params: Record<string, string | number> = {}) => get<AuditLog[]>('/audit-logs', params),
  modelUsage: (limit = 100) => get<ModelUsage[]>('/model-usage', { limit }),
  modelUsageSummary: () => get<ModelUsageSummary>('/model-usage/summary'),
  dashboardOperations: () => get<OperationsSummary>('/dashboard/operations'),
}
