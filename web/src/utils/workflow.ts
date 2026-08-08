import type { WorkflowDetails, WorkflowNotification } from '@/types/workflow'

export const workflowTerminal = (status: string) => ['COMPLETED', 'FAILED', 'MANUAL_REVIEW', 'CANCELLED'].includes(status)

export const unreadNotificationCount = (items: WorkflowNotification[]) => items.filter((item) => item.status === 'UNREAD').length

export const parseWorkflowEventFrame = (frame: string): WorkflowDetails | undefined => {
  const lines = frame.replace(/\r\n/g, '\n').split('\n')
  const event = lines.find((line) => line.startsWith('event:'))?.slice(6).trim()
  const raw = lines.filter((line) => line.startsWith('data:')).map((line) => line.slice(5).trim()).join('')
  return event === 'workflow' && raw ? JSON.parse(raw) as WorkflowDetails : undefined
}

export const workflowStatusType = (status: string) => {
  if (status === 'COMPLETED' || status === 'SUCCEEDED') return 'success'
  if (status === 'FAILED' || status === 'MANUAL_REVIEW' || status === 'CANCELLED') return 'danger'
  if (status === 'WAITING_APPROVAL' || status === 'WAITING_AGENT' || status === 'RUNNING' || status === 'RETRYING') return 'warning'
  return 'info'
}

export const agentAvailabilityType = (status: string) => {
  if (status === 'READY' || status === 'UP') return 'success'
  if (status === 'DEGRADED') return 'warning'
  return 'info'
}
