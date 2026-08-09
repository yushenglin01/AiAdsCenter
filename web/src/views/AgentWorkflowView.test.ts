import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AgentWorkflowView from './AgentWorkflowView.vue'

vi.mock('element-plus/es/components/badge/index', () => ({ ElBadge: { template: '<span><slot /></span>' } }))
vi.mock('element-plus/es/components/date-picker/index', () => ({ ElDatePicker: { template: '<input />' } }))
vi.mock('element-plus/es/components/message/index', () => ({ ElMessage: { success: vi.fn(), error: vi.fn() } }))
vi.mock('element-plus/es/components/badge/style/css', () => ({}))
vi.mock('element-plus/es/components/date-picker/style/css', () => ({}))

vi.mock('@/api/catalog', () => ({
  catalogApi: {
    listCampaigns: vi.fn(async () => [{ id: 'campaign-1', game_id: 'game-1', channel_id: 'channel-1', external_id: 'google-jp', name: 'Google JP Stable', country: 'JP', daily_budget: '100', currency: 'USD', status: 'ACTIVE' }]),
  },
}))

vi.mock('@/api/workflows', () => ({
  workflowApi: {
    listAgents: vi.fn(async () => []),
    listAgentRuntime: vi.fn(async () => []),
    list: vi.fn(async () => []),
    listNotifications: vi.fn(async () => [{ id: 'notice-1', tenant_id: 'tenant-1', workflow_id: 'workflow-failed', event_type: 'WORKFLOW_FAILED', channel: 'INTERNAL', title: '多 Agent 分析失败', message: '工作流已停止，请查看失败步骤后重试。', status: 'UNREAD', created_at: '2026-08-09T17:48:00Z', updated_at: '2026-08-09T17:48:00Z' }]),
    get: vi.fn(async () => ({ workflow: { workflow_id: 'workflow-failed', workflow_type: 'FULL_ANALYSIS', game_id: 'game-1', campaign_id: 'campaign-1', status: 'FAILED', current_step: 'business-agent', idempotency_key: 'key', error_message: 'campaign metrics not found', triggered_by: 'user-1', created_at: '2026-08-09T17:46:00Z', updated_at: '2026-08-09T17:48:00Z' }, steps: [{ step_id: 'step-1', workflow_id: 'workflow-failed', agent_name: 'business-agent', sequence_number: 6, status: 'FAILED', execution_mode: 'ASYNC', error_message: 'campaign metrics not found' }] })),
    openClawCommand: vi.fn(),
    markNotificationRead: vi.fn(async (id: string) => ({ id, tenant_id: 'tenant-1', workflow_id: 'workflow-failed', event_type: 'WORKFLOW_FAILED', channel: 'INTERNAL', title: '多 Agent 分析失败', message: '工作流已停止，请查看失败步骤后重试。', status: 'READ', created_at: '2026-08-09T17:48:00Z', updated_at: '2026-08-09T17:48:00Z' })),
    watch: vi.fn(async () => undefined),
  },
}))

const ButtonStub = defineComponent({
  props: { disabled: Boolean },
  emits: ['click'],
  template: '<button :disabled="disabled" @click="$emit(\'click\')"><slot /></button>',
})

const DialogStub = defineComponent({
  props: { modelValue: Boolean, title: String },
  template: '<section v-if="modelValue" role="dialog"><h2>{{ title }}</h2><slot /><footer><slot name="footer" /></footer></section>',
})

const TabPaneStub = defineComponent({
  props: { name: String, label: String },
  template: '<section :data-pane="name"><slot name="label">{{ label }}</slot><slot /></section>',
})

const FailureDialogStub = defineComponent({
  props: { modelValue: Boolean, details: Object },
  template: '<section v-if="modelValue" data-testid="failure-dialog">{{ details?.workflow?.error_message }}</section>',
})

describe('AgentWorkflowView', () => {
  it('separates secondary functions and opens the workflow launcher in a dialog', async () => {
    const wrapper = mount(AgentWorkflowView, {
      global: {
        stubs: {
          AppShell: { template: '<main><slot /></main>' },
          AgentOperationsPanel: true,
          ResearchSourceManager: true,
          WorkflowFailureDialog: FailureDialogStub,
          ElAlert: true,
          ElBadge: { template: '<span><slot /></span>' },
          ElButton: ButtonStub,
          ElDatePicker: { template: '<input aria-label="分析日期" />' },
          ElDialog: DialogStub,
          ElEmpty: { props: ['description'], template: '<div>{{ description }}<slot /></div>' },
          ElOption: true,
          ElSelect: { template: '<select><slot /></select>' },
          ElTabPane: TabPaneStub,
          ElTabs: { template: '<div><slot /></div>' },
          ElTag: { template: '<span><slot /></span>' },
        },
      },
    })
    await flushPromises()

    expect(wrapper.find('.workflow-launcher').exists()).toBe(false)
    expect(wrapper.findAll('[data-pane]')).toHaveLength(4)
    expect(wrapper.text()).toContain('Agent 运行')
    expect(wrapper.text()).toContain('来源治理')
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)

    const launchButton = wrapper.findAll('button').find((button) => button.text() === '启动多 Agent 工作流')
    expect(launchButton).toBeDefined()
    await launchButton!.trigger('click')

    const dialog = wrapper.get('[role="dialog"]')
    expect(dialog.text()).toContain('启动完整经营分析')
    expect(dialog.text()).toContain('广告计划')
    expect(dialog.text()).toContain('分析日期')
  })

  it('loads the linked workflow when a failed notification is opened', async () => {
    const wrapper = mount(AgentWorkflowView, {
      global: {
        stubs: {
          AppShell: { template: '<main><slot /></main>' }, AgentOperationsPanel: true, ResearchSourceManager: true, WorkflowFailureDialog: FailureDialogStub,
          ElAlert: true, ElBadge: { template: '<span><slot /></span>' }, ElButton: ButtonStub, ElDatePicker: true, ElDialog: DialogStub, ElEmpty: true, ElOption: true, ElSelect: true, ElTabPane: TabPaneStub, ElTabs: { template: '<div><slot /></div>' }, ElTag: { template: '<span><slot /></span>' },
        },
      },
    })
    await flushPromises()

    await wrapper.get('.notification-list article').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="failure-dialog"]').text()).toContain('campaign metrics not found')
  })
})
