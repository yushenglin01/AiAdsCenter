import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

declare module 'vue-router' {
  interface RouteMeta { requiresAuth?: boolean; roles?: string[] }
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: () => import('@/views/LoginView.vue') },
    { path: '/register', name: 'register', component: () => import('@/views/RegisterView.vue') },
    { path: '/verify-email', name: 'verify-email', component: () => import('@/views/VerifyEmailView.vue') },
    { path: '/', name: 'dashboard', component: () => import('@/views/DashboardView.vue'), meta: { requiresAuth: true } },
	{ path: '/metrics', name: 'metrics', component: () => import('@/views/MetricsView.vue'), meta: { requiresAuth: true } },
	{ path: '/attribution', name: 'attribution', component: () => import('@/views/AttributionView.vue'), meta: { requiresAuth: true } },
	{ path: '/creative-analysis', name: 'creative-analysis', component: () => import('@/views/CreativeAnalysisView.vue'), meta: { requiresAuth: true } },
	{ path: '/rules', name: 'rules', component: () => import('@/views/RulesView.vue'), meta: { requiresAuth: true } },
	{ path: '/business-analysis', name: 'business-analysis', component: () => import('@/views/BusinessAnalysisView.vue'), meta: { requiresAuth: true } },
	{ path: '/agent-workflows', name: 'agent-workflows', component: () => import('@/views/AgentWorkflowView.vue'), meta: { requiresAuth: true } },
	{ path: '/recommendations', name: 'recommendations', component: () => import('@/views/RecommendationsView.vue'), meta: { requiresAuth: true } },
	{ path: '/approvals', name: 'approvals', component: () => import('@/views/ApprovalsView.vue'), meta: { requiresAuth: true } },
	{ path: '/model-usage', name: 'model-usage', component: () => import('@/views/ModelUsageView.vue'), meta: { requiresAuth: true, roles: ['ADMIN','MANAGER'] } },
	{ path: '/audit-logs', name: 'audit-logs', component: () => import('@/views/AuditLogsView.vue'), meta: { requiresAuth: true, roles: ['ADMIN'] } },
	{ path: '/games', name: 'games', component: () => import('@/views/GameManagementView.vue'), meta: { requiresAuth: true } },
	{ path: '/assets', name: 'assets', component: () => import('@/views/AssetManagementView.vue'), meta: { requiresAuth: true } },
	{ path: '/imports', name: 'imports', component: () => import('@/views/DataImportView.vue'), meta: { requiresAuth: true } },
    { path: '/admin', name: 'admin', component: () => import('@/views/AdminView.vue'), meta: { requiresAuth: true, roles: ['ADMIN'] } },
    { path: '/forbidden', name: 'forbidden', component: () => import('@/views/ForbiddenView.vue') },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (auth.isAuthenticated && !auth.initialized) {
    try { await auth.loadSession() } catch { auth.signOut() }
  }
  if (to.meta.requiresAuth && !auth.isAuthenticated) return { name: 'login', query: { redirect: to.fullPath } }
  if (['login', 'register'].includes(String(to.name)) && auth.isAuthenticated) return { name: 'dashboard' }
  if (to.meta.roles?.length && !auth.hasAnyRole(to.meta.roles)) return { name: 'forbidden' }
  return true
})

export default router
