import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ResearchSourceManager from './ResearchSourceManager.vue'

vi.mock('element-plus/es/components/date-picker/index', () => ({ ElDatePicker: { template: '<input />' } }))
vi.mock('element-plus/es/components/message/index', () => ({ ElMessage: { success: vi.fn(), warning: vi.fn(), error: vi.fn() } }))
vi.mock('element-plus/es/components/message-box/index', () => ({ ElMessageBox: { prompt: vi.fn() } }))
vi.mock('element-plus/es/components/date-picker/style/css', () => ({}))

const loadSources = vi.fn(async () => undefined)

const DialogStub = defineComponent({
  props: { modelValue: Boolean },
  template: '<section v-if="modelValue" role="dialog"><slot name="header"/><slot/></section>',
})

const SourcePanelStub = defineComponent({
  setup(_, { expose }) {
    expose({ load: loadSources })
    return {}
  },
  template: '<div data-testid="source-panel">来源管理内容</div>',
})

describe('ResearchSourceManager', () => {
  it('keeps source management out of the page until the dialog is opened', async () => {
    const wrapper = mount(ResearchSourceManager, {
      props: { campaigns: [] },
      global: {
        stubs: {
          ElButton: { emits: ['click'], template: '<button @click="$emit(\'click\')"><slot/></button>' },
          ElDialog: DialogStub,
          ResearchSourcePanel: SourcePanelStub,
        },
      },
    })

    expect(wrapper.text()).toContain('Research Agent 来源治理')
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="source-panel"]').exists()).toBe(false)

    await wrapper.get('button').trigger('click')

    expect(wrapper.get('[role="dialog"]').text()).toContain('Research Agent 来源库')
    expect(wrapper.find('[data-testid="source-panel"]').exists()).toBe(true)
  })
})
