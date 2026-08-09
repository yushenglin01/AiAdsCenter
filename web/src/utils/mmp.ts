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
  return provider === 'ADJUST' ? '连接已就绪，可拉取 Adjust Report Service API。' : '连接已就绪，可拉取 AppsFlyer Raw Data Pull API v5。'
}
