import { apiClient } from './client'
import type { ApiEnvelope } from '@/types/api'
import type { AttributionFinding, CampaignMetric, CreativeFinding, Overview, RecalculateResult, Rule, RuleOverview, TrendPoint } from '@/types/metrics'

async function get<T>(path: string, params?: Record<string, string>): Promise<T> {
  const { data } = await apiClient.get<ApiEnvelope<T>>(path, { params })
  return data.data
}

export const metricsApi = {
  overview: (gameID = '') => get<Overview>('/metrics/overview', gameID ? { game_id: gameID } : undefined),
  campaigns: (gameID = '') => get<CampaignMetric[]>('/metrics/campaigns', gameID ? { game_id: gameID } : undefined),
  trends: (gameID = '', campaignID = '') => get<TrendPoint[]>('/metrics/trends', { ...(gameID ? { game_id: gameID } : {}), ...(campaignID ? { campaign_id: campaignID } : {}) }),
  recalculate: async (gameID: string) => {
    const { data } = await apiClient.post<ApiEnvelope<RecalculateResult>>('/metrics/recalculate', undefined, { params: { game_id: gameID } })
    return data.data
  },
  rules: (gameID = '') => get<RuleOverview>('/analysis/rules', gameID ? { game_id: gameID } : undefined),
  updateRule: async (id: string, payload: Pick<Rule, 'threshold' | 'consecutive_days' | 'enabled'>) => {
    const { data } = await apiClient.put<ApiEnvelope<Rule>>(`/analysis/rules/${id}`, payload)
    return data.data
  },
  attribution: (gameID = '') => get<AttributionFinding[]>('/analysis/attribution', gameID ? { game_id: gameID } : undefined),
  creative: (gameID = '') => get<CreativeFinding[]>('/analysis/creative', gameID ? { game_id: gameID } : undefined),
}
