import type { ResearchSource, ResearchSourceInput } from '@/types/research'

export function validateResearchSourceInput(input: ResearchSourceInput) {
  if (!input.title.trim() || !input.summary.trim() || !input.publisher.trim() || !input.published_at) return false
  try {
    const url = new URL(input.source_url)
    return url.protocol === 'https:' && !url.username && !url.password
  } catch {
    return false
  }
}

export function canReviewResearchSource(roles: string[], source: ResearchSource) {
  return source.status === 'PENDING' && roles.some((role) => role === 'ADMIN' || role === 'MANAGER')
}
