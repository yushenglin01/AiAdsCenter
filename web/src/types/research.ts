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
  discovery_method: 'MANUAL' | 'WEB_SEARCH'
  discovery_provider?: string
  discovery_query_hash?: string
  discovered_at?: string
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

export interface WebSearchCapability {
  configured: boolean
  provider: string
  import_enabled: boolean
  max_results: number
  details: string[]
}

export interface WebSearchResult {
  title: string
  url: string
  description: string
  publisher: string
  published_at?: string
  language?: string
}

export interface WebSearchInput {
  query: string
  game_id?: string
  campaign_id?: string
  category: ResearchCategory
  count: number
  country?: string
  search_lang?: string
  freshness?: '' | 'pd' | 'pw' | 'pm' | 'py'
}

export interface WebSearchResponse {
  capability: WebSearchCapability
  query_hash: string
  results: WebSearchResult[]
  searched_at: string
}

export interface WebImportInput {
  query: string
  game_id?: string
  campaign_id?: string
  category: ResearchCategory
  result: WebSearchResult
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
