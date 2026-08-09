import { describe, expect, it } from 'vitest'
import { agentPresentation, elapsedLabel, runtimeStatusLabel, toolMetadata } from './agent'

describe('agent presentation helpers', () => {
  it('provides business-facing names and tool descriptions', () => {
    expect(agentPresentation('research-agent').displayName).toBe('研究 Agent')
    expect(toolMetadata('search_public_web')).toMatchObject({ label: '实时联网检索', access: '受控操作' })
  })

  it('formats runtime state and elapsed time', () => {
    expect(runtimeStatusLabel('RUNNING')).toBe('处理中')
    expect(elapsedLabel('2026-08-10T00:00:00Z', Date.parse('2026-08-10T00:01:05Z'))).toBe('1 分 5 秒')
  })
})
