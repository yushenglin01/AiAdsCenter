import { apiClient } from './client'
import type { ApiEnvelope } from '@/types/api'
import type { AgentCatalogItem, OpenClawResult, WorkflowDetails, WorkflowNotification, WorkflowRun, WorkflowStartInput } from '@/types/workflow'
import { parseWorkflowEventFrame } from '@/utils/workflow'

const terminal = (status: string) => ['WAITING_APPROVAL', 'COMPLETED', 'FAILED', 'MANUAL_REVIEW', 'CANCELLED'].includes(status)

export const workflowApi = {
  listAgents: async () => {
    const { data } = await apiClient.get<ApiEnvelope<AgentCatalogItem[]>>('/agents')
    return data.data
  },
  list: async () => {
    const { data } = await apiClient.get<ApiEnvelope<WorkflowRun[]>>('/workflows')
    return data.data
  },
  get: async (id: string) => {
    const { data } = await apiClient.get<ApiEnvelope<WorkflowDetails>>(`/workflows/${id}`)
    return data.data
  },
  start: async (payload: WorkflowStartInput) => {
    const { data } = await apiClient.post<ApiEnvelope<WorkflowDetails>>('/workflows/analysis', payload)
    return data.data
  },
  openClawCommand: async (payload: WorkflowStartInput) => {
    const { data } = await apiClient.post<ApiEnvelope<OpenClawResult<WorkflowDetails>>>('/openclaw/commands', { intent: 'RUN_FULL_ANALYSIS', input: payload })
    return data.data.data
  },
  listNotifications: async (status?: 'UNREAD' | 'READ') => {
    const { data } = await apiClient.get<ApiEnvelope<WorkflowNotification[]>>('/notifications', { params: status ? { status } : undefined })
    return data.data
  },
  markNotificationRead: async (id: string) => {
    const { data } = await apiClient.post<ApiEnvelope<WorkflowNotification>>(`/notifications/${id}/read`)
    return data.data
  },
  watch: async (id: string, onWorkflow: (details: WorkflowDetails) => void, signal: AbortSignal) => {
    const token = localStorage.getItem('gai_access_token')
    const base = String(apiClient.defaults.baseURL || '/api/v1').replace(/\/$/, '')
    const response = await fetch(`${base}/workflows/${id}/events`, { headers: token ? { Authorization: `Bearer ${token}` } : {}, signal })
    if (!response.ok || !response.body) throw new Error(`SSE request failed: ${response.status}`)
    const reader = response.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''
    while (true) {
      const { value, done } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true }).replace(/\r\n/g, '\n')
      const frames = buffer.split('\n\n')
      buffer = frames.pop() || ''
      for (const frame of frames) {
        const details = parseWorkflowEventFrame(frame)
        if (details) {
          onWorkflow(details)
          if (terminal(details.workflow.status)) return
        }
      }
    }
  },
}
