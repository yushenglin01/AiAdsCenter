import { apiClient } from './client'
import type { ApiEnvelope } from '@/types/api'
import type { AgentTask, AnalysisReport, TaskDetails } from '@/types/business'

const terminal = (status: string) => ['WAITING_APPROVAL', 'SUCCEEDED', 'FAILED', 'MANUAL_REVIEW', 'CANCELLED'].includes(status)

export const businessApi = {
  listTasks: async () => { const { data } = await apiClient.get<ApiEnvelope<AgentTask[]>>('/analysis/business/tasks'); return data.data },
  getTask: async (id: string) => { const { data } = await apiClient.get<ApiEnvelope<TaskDetails>>(`/analysis/business/tasks/${id}`); return data.data },
  analyze: async (payload: { game_id: string; campaign_id: string; analysis_date: string }) => { const { data } = await apiClient.post<ApiEnvelope<TaskDetails>>('/analysis/business', payload); return data.data },
  getReport: async (id: string) => { const { data } = await apiClient.get<ApiEnvelope<AnalysisReport>>(`/analysis/business/tasks/${id}/report`); return data.data },
  watchTask: async (id: string, onTask: (details: TaskDetails) => void, signal: AbortSignal) => {
    const token = localStorage.getItem('gai_access_token')
    const base = String(apiClient.defaults.baseURL || '/api/v1').replace(/\/$/, '')
    const response = await fetch(`${base}/analysis/business/tasks/${id}/events`, { headers: token ? { Authorization: `Bearer ${token}` } : {}, signal })
    if (!response.ok || !response.body) throw new Error(`SSE request failed: ${response.status}`)
    const reader = response.body.getReader(); const decoder = new TextDecoder(); let buffer = ''
    while (true) {
      const { value, done } = await reader.read(); if (done) break
      buffer += decoder.decode(value, { stream: true })
      const frames = buffer.split('\n\n'); buffer = frames.pop() || ''
      for (const frame of frames) {
        const event = frame.split('\n').find((line) => line.startsWith('event:'))?.slice(6).trim()
        const raw = frame.split('\n').filter((line) => line.startsWith('data:')).map((line) => line.slice(5).trim()).join('')
        if (event === 'task' && raw) { const details = JSON.parse(raw) as TaskDetails; onTask(details); if (terminal(details.task.status)) return }
      }
    }
  },
}
