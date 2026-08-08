import { describe, expect, it } from 'vitest'
import { agentAvailabilityType, parseWorkflowEventFrame, unreadNotificationCount, workflowStatusType, workflowTerminal } from './workflow'

describe('workflow presentation state', () => {
  it('recognizes terminal workflow states', () => {
    expect(workflowTerminal('COMPLETED')).toBe(true)
    expect(workflowTerminal('WAITING_APPROVAL')).toBe(false)
  })

  it('maps workflow and agent states to semantic tags', () => {
    expect(workflowStatusType('FAILED')).toBe('danger')
    expect(workflowStatusType('WAITING_AGENT')).toBe('warning')
    expect(agentAvailabilityType('READY')).toBe('success')
    expect(agentAvailabilityType('DEGRADED')).toBe('warning')
  })

  it('parses workflow SSE frames and ignores reconnect events', () => {
    const frame = 'event: workflow\r\ndata: {"workflow":{"workflow_id":"wf-1","status":"RUNNING"},"steps":[]}\r\n'
    expect(parseWorkflowEventFrame(frame)?.workflow.workflow_id).toBe('wf-1')
    expect(parseWorkflowEventFrame('event: reconnect\ndata: {}')).toBeUndefined()
  })

  it('counts only unread internal messages', () => {
    expect(unreadNotificationCount([{ status: 'UNREAD' }, { status: 'READ' }, { status: 'UNREAD' }] as any)).toBe(2)
  })
})
