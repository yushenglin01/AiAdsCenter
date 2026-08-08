<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowRight,
  Bell,
  CircleCheck,
  Close,
  Coin,
  Collection,
  Connection,
  Cpu,
  DataAnalysis,
  Document,
  MagicStick,
  Menu,
  Monitor,
  PictureFilled,
  Setting,
  SwitchButton,
  TrendCharts,
  UploadFilled,
} from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import BrandGlyph from '@/components/BrandGlyph.vue'

type NavItem = {
  label: string
  to: string
  icon: object
  roles?: string[]
}

type NavSection = {
  label: string
  icon: object
  items: NavItem[]
}

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const mobileOpen = ref(false)

const navigation: NavSection[] = [
  {
    label: '经营洞察',
    icon: DataAnalysis,
    items: [
      { label: '经营总览', to: '/', icon: DataAnalysis },
      { label: '计划指标', to: '/metrics', icon: TrendCharts },
      { label: '归因分析', to: '/attribution', icon: Connection },
      { label: '素材分析', to: '/creative-analysis', icon: PictureFilled },
      { label: '规则中心', to: '/rules', icon: Setting },
    ],
  },
  {
    label: '智能协作',
    icon: Cpu,
    items: [
      { label: '经营分析', to: '/business-analysis', icon: MagicStick },
      { label: 'Agent 工作台', to: '/agent-workflows', icon: Cpu },
      { label: '建议中心', to: '/recommendations', icon: Bell },
      { label: '审批中心', to: '/approvals', icon: CircleCheck },
    ],
  },
  {
    label: '数据与系统',
    icon: Collection,
    items: [
      { label: '模型用量', to: '/model-usage', icon: Coin, roles: ['ADMIN', 'MANAGER'] },
      { label: '审计日志', to: '/audit-logs', icon: Document, roles: ['ADMIN'] },
      { label: '游戏管理', to: '/games', icon: Monitor },
      { label: '投放资产', to: '/assets', icon: Collection },
      { label: '数据导入', to: '/imports', icon: UploadFilled },
      { label: '系统管理', to: '/admin', icon: Setting, roles: ['ADMIN'] },
    ],
  },
]

const sectionForPath = (path: string) => navigation.find((section) => section.items.some((item) => item.to === path))?.label || navigation[0].label
const openSection = ref(sectionForPath(route.path))

const visibleNavigation = computed(() => navigation.map((section) => ({
  ...section,
  items: section.items.filter((item) => !item.roles || auth.hasAnyRole(item.roles)),
})).filter((section) => section.items.length))

const currentItem = computed(() => navigation.flatMap((section) => section.items).find((item) => item.to === route.path))
const initials = computed(() => (auth.user?.display_name || auth.user?.username || 'AN').trim().slice(0, 2).toUpperCase())

watch(() => route.fullPath, () => {
  mobileOpen.value = false
  openSection.value = sectionForPath(route.path)
})

function toggleSection(label: string) {
  openSection.value = openSection.value === label ? '' : label
}

function logout() {
  auth.signOut()
  router.push('/login')
}
</script>

<template>
  <div class="app-shell" :class="{ 'nav-open': mobileOpen }">
    <button v-if="mobileOpen" class="nav-scrim" aria-label="关闭导航" @click="mobileOpen = false"></button>
    <aside class="sidebar" aria-label="主导航">
      <div class="sidebar-brand">
        <BrandGlyph />
        <div class="brand-copy"><strong>AdNova</strong><span>星曜智投</span></div>
        <button class="sidebar-close" type="button" aria-label="关闭导航" @click="mobileOpen = false"><Close /></button>
      </div>

      <nav>
        <section v-for="(section, index) in visibleNavigation" :key="section.label" class="nav-section">
          <button
            class="nav-group-toggle"
            :class="{ active: section.items.some((item) => item.to === route.path), open: openSection === section.label }"
            type="button"
            :aria-expanded="openSection === section.label"
            :aria-controls="`nav-group-${index}`"
            @click="toggleSection(section.label)"
          >
            <component :is="section.icon" class="nav-group-icon" />
            <span>{{ section.label }}</span>
            <ArrowRight class="nav-chevron" aria-hidden="true" />
          </button>
          <div v-show="openSection === section.label" :id="`nav-group-${index}`" class="nav-submenu">
            <router-link v-for="item in section.items" :key="item.to" :to="item.to">
              <component :is="item.icon" />
              <span>{{ item.label }}</span>
              <i aria-hidden="true"></i>
            </router-link>
          </div>
        </section>
      </nav>

      <div class="stage-note">
        <div><span class="live-dot"></span><strong>DATA PIPELINE</strong><small>ONLINE</small></div>
        <p>AppsFlyer 数据闭环</p>
        <span>PHASE 11 · v1.1.1</span>
      </div>
    </aside>

    <main class="workspace">
      <header class="topbar">
        <div class="topbar-title">
          <button class="menu-trigger" type="button" aria-label="打开导航" @click="mobileOpen = true"><Menu /></button>
          <div>
            <span class="eyebrow">ADNOVA · {{ currentItem?.label || '星曜智投' }}</span>
            <h1>{{ auth.tenant?.name || '星曜智投' }}</h1>
          </div>
        </div>

        <div class="topbar-actions">
          <div class="system-status"><span></span><div><small>系统状态</small><strong>运行正常</strong></div></div>
          <div class="account">
            <span class="account-avatar">{{ initials }}</span>
            <div><strong>{{ auth.user?.display_name }}</strong><span>{{ auth.user?.roles.join(' · ') }}</span></div>
          </div>
          <button class="logout-button" type="button" aria-label="退出登录" title="退出登录" @click="logout"><SwitchButton /></button>
        </div>
      </header>
      <section class="page-content"><slot /></section>
    </main>
  </div>
</template>
