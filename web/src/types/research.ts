export type ResearchCategory = 'POLICY' | 'COMPETITOR' | 'MARKET'
export type ResearchStatus = 'PENDING' | 'VERIFIED' | 'REJECTED'

export interface ResearchSource {
  id: string
  tenant_id: string
  game_id?: string
  campaign_id?: string
  category: ResearchCategory
  title: string
  summary: string
  source_url: string
  publisher: string
  published_at: string
  content_hash: string
  status: ResearchStatus
  review_comment?: string
  created_by: string
  reviewed_by?: string
  reviewed_at?: string
  created_at: string
  updated_at: string
}

export interface ResearchSourceInput {
  game_id?: string
  campaign_id?: string
  category: ResearchCategory
  title: string
  summary: string
  source_url: string
  publisher: string
  published_at: string
}
