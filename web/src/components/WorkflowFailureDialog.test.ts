import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import WorkflowFailureDialog from './WorkflowFailureDialog.vue'
import type { WorkflowDetails, WorkflowNotification } from '@/types/workflow'

const DrawerStub = defineComponent({
  props: { modelValue: Boolean },
  emits: ['update:modelValue'],
  template: '<aside v-if="modelValue" role="dialog"><slot/></aside>',
})

const notification: WorkflowNotification = {
  id: 'notice-1', tenant_id: 'tenant-1', workflow_id: 'workflow-1', event_type: 'WORKFLOW_FAILED', channel: 'INTERNAL', title: '多 Agent 分析失败', message: '工作流已停止', status: 'UNREAD', created_at: '2026-08-09T17:48:00Z', updated_at: '2026-08-09T17:48:00Z',
}

const details: WorkflowDetails = {
  workflow: { workflow_id: 'workflow-1', workflow_type: 'FULL_ANALYSIS', game_id: 'game-1', campaign_id: 'campaign-1', status: 'FAILED', current_step: 'business-agent', idempotency_key: 'key-1', error_message: 'collect business context: campaign metrics not found', triggered_by: 'user-1', created_at: '2026-08-09T17:46:00Z', updated_at: '2026-08-09T17:48:00Z', finished_at: '2026-08-09T17:48:00Z' },
  steps: [
    { step_id: 'step-1', workflow_id: 'workflow-1', agent_name: 'data-agent', sequence_number: 1, status: 'SUCCEEDED', execution_mode: 'SYNC' },
    { step_id: 'step-2', workflow_id: 'workflow-1', agent_name: 'business-agent', sequence_number: 2, status: 'FAILED', execution_mode: 'ASYNC', error_message: 'collect business context: campaign metrics not found', finished_at: '2026-08-09T17:48:00Z' },
  ],
}

describe('WorkflowFailureDialog', () => {
  it('shows the exact failed step and recovery guidance', async () => {
    const wrapper = mount(WorkflowFailureDialog, {
      props: { modelValue: true, notification, details, campaignName: 'Google JP Stable' },
      global: { stubs: { ElDrawer: DrawerStub, ElAlert: true, ElEmpty: true, ElButton: { emits: ['click'], template: '<button @click="$emit(\'click\')"><slot/></button>' }, ElTag: { template: '<span><slot/></span>' } } },
    })

    expect(wrapper.text()).toContain('collect business context: campaign metrics not found')
    expect(wrapper.text()).toContain('business-agent')
    expect(wrapper.text()).toContain('请先完成数据同步')
    expect(wrapper.text()).toContain('Google JP Stable')

    await wrapper.findAll('button').find((button) => button.text() === '定位到工作流')!.trigger('click')
    expect(wrapper.emitted('locate')).toHaveLength(1)
  })
})
