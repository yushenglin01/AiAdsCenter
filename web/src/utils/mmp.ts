import type { MMPConnection } from '@/types/catalog'
import type { MMPProvider } from '@/api/mmp'

const dayMs = 86_400_000

export function validateSyncRange(range: string[], maxDays = 7): string {
  if (range.length !== 2 || !range[0] || !range[1]) return '请选择同步日期范围。'
  const from = Date.parse(`${range[0]}T00:00:00Z`)
  const to = Date.parse(`${range[1]}T00:00:00Z`)
  if (!Number.isFinite(from) || !Number.isFinite(to) || to < from) return '同步日期范围无效。'
  if (Math.floor((to - from) / dayMs) + 1 > maxDays) return `单次同步不能超过 ${maxDays} 天。`
  return ''
}

export function connectionHint(provider: MMPProvider, connection?: MMPConnection): string {
  if (!connection) return provider === 'ADJUST' ? '尚未保存该游戏的 Adjust App Token。' : '尚未保存该游戏的 AppsFlyer App ID。'
  if (connection.status === 'DISABLED') return '连接已停用。'
  if (!connection.credential_configured) return provider === 'ADJUST' ? '服务端尚未配置 Adjust API Token 与事件指标映射。' : '服务端尚未配置 AppsFlyer API Token。'
  if (connection.health === 'UNVERIFIED') return provider === 'ADJUST' ? '服务端配置已完成；请执行首次同步以验证 Token、App 映射和指标权限。' : '服务端配置已完成；请执行首次同步以验证 Token、App ID 和报表权限。'
  const verifiedAt = connection.last_sync_at?.slice(0, 16).replace('T', ' ')
  const suffix = verifiedAt ? `，记录于 ${verifiedAt}` : ''
  return provider === 'ADJUST' ? `最近一次 Adjust Report Service 同步成功${suffix}。` : `最近一次 AppsFlyer Raw Data Pull API v5 同步成功${suffix}。`
}

export function canSyncConnection(connection?: MMPConnection): boolean {
  return Boolean(connection && connection.status === 'ACTIVE' && connection.credential_configured)
}

export function connectionHealthLabel(health: MMPConnection['health']): string {
  return ({ READY: '已验证', UNVERIFIED: '待验证', NOT_CONFIGURED: '未配置', DISABLED: '已停用' } as const)[health]
}
