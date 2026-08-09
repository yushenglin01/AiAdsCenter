import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AgentOperationsPanel from './AgentOperationsPanel.vue'

const { webCapability, searchWeb, importWebResult } = vi.hoisted(() => ({ webCapability: vi.fn(), searchWeb: vi.fn(), importWebResult: vi.fn() }))

vi.mock('@/api/research', () => ({ researchApi: { webCapability, searchWeb, importWebResult } }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ hasAnyRole: () => true }) }))
vi.mock('element-plus/es/components/message/index', () => ({ ElMessage: { success: vi.fn(), error: vi.fn(), info: vi.fn() } }))

const ButtonStub = defineComponent({ props: { disabled: Boolean }, emits: ['click'], template: '<button :disabled="disabled" @click="$emit(\'click\')"><slot /></button>' })
const DrawerStub = defineComponent({ props: { modelValue: Boolean }, template: '<aside v-if="modelValue" role="dialog"><slot /></aside>' })
const InputStub = defineComponent({ props: { modelValue: String }, emits: ['update:modelValue'], template: '<textarea :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />' })
const TabPaneStub = defineComponent({ template: '<section><slot /></section>' })

const agent = {
  definition: {
    spec: { name: 'research-agent', version: '1.2.0', description: '联网检索并保留来源', execution_mode: 'VERIFIED_RESEARCH', tools: ['search_public_web', 'preserve_provenance'], permissions: ['SEARCH_PUBLIC_WEB'], max_steps: 8, timeout: 120_000_000_000 },
    availability: 'READY',
  },
  health: { status: 'UP', provider: 'verified-source-repository+brave', details: ['live_web_search_ready'] },
}

const runtime = {
  agent_name: 'research-agent', runtime_status: 'RUNNING' as const,
  active_tasks: [{ agent_name: 'research-agent', workflow_id: 'workflow-1', workflow_status: 'RUNNING', campaign_id: 'campaign-1', campaign_name: 'Google JP Stable', status: 'RUNNING', execution_mode: 'VERIFIED_RESEARCH', started_at: new Date().toISOString(), updated_at: new Date().toISOString() }],
}

describe('AgentOperationsPanel', () => {
  beforeEach(() => {
    webCapability.mockResolvedValue({ configured: true, provider: 'brave', import_enabled: true, max_results: 8, details: [] })
    searchWeb.mockResolvedValue({ capability: { configured: true, provider: 'brave', import_enabled: true, max_results: 8, details: [] }, query_hash: 'hash', searched_at: new Date().toISOString(), results: [{ title: '最新市场报告', url: 'https://example.com/report', description: '市场信号摘要', publisher: 'Example' }] })
    importWebResult.mockResolvedValue({ id: 'source-1', status: 'PENDING', discovery_method: 'WEB_SEARCH' })
  })

  it('shows active work and supports live web discovery into pending review', async () => {
    const wrapper = mount(AgentOperationsPanel, {
      props: { agents: [agent], runtimes: [runtime], campaigns: [{ id: 'campaign-1', game_id: 'game-1', channel_id: 'channel-1', external_id: 'google-jp', name: 'Google JP Stable', country: 'JP', daily_budget: '100', currency: 'USD', status: 'ACTIVE' }] },
      global: { stubs: { ElAlert: true, ElButton: ButtonStub, ElDrawer: DrawerStub, ElEmpty: true, ElInput: InputStub, ElOption: true, ElSelect: { template: '<div><slot /></div>' }, ElTabPane: TabPaneStub, ElTabs: { template: '<div><slot /></div>' }, ElTag: { template: '<span><slot /></span>' } } },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Google JP Stable')
    expect(wrapper.text()).toContain('处理中')
    expect(wrapper.text()).toContain('实时联网 · brave')

    await wrapper.get('.agent-operation-card').trigger('click')
    expect(wrapper.get('[role="dialog"]').text()).toContain('实时联网已就绪')
    await wrapper.get('textarea').setValue('日本手游广告最新政策')
    const searchButton = wrapper.findAll('button').find((button) => button.text() === '实时联网查询')
    await searchButton!.trigger('click')
    await flushPromises()
    expect(searchWeb).toHaveBeenCalled()
    expect(wrapper.text()).toContain('最新市场报告')

    const importButton = wrapper.findAll('button').find((button) => button.text() === '加入待核验来源')
    await importButton!.trigger('click')
    await flushPromises()
    expect(wrapper.emitted('imported')).toHaveLength(1)
    wrapper.unmount()
  })
})
