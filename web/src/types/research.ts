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
  discovery_method: 'MANUAL' | 'WEB_SEARCH' | 'SCHEDULED_WEB_SEARCH'
  discovery_provider?: string
  discovery_query_hash?: string
  discovery_schedule_id?: string
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

export interface ResearchScheduleInput {
  name: string
  game_id?: string
  campaign_id?: string
  category: ResearchCategory
  query: string
  country?: string
  search_lang?: string
  freshness?: '' | 'pd' | 'pw' | 'pm' | 'py'
  result_count: number
  interval_minutes: number
  enabled: boolean
}

export interface ResearchSchedule extends ResearchScheduleInput {
  id: string
  tenant_id: string
  query_hash: string
  next_run_at?: string
  last_run_at?: string
  last_status?: 'SUCCEEDED' | 'FAILED'
  last_error_code?: string
  last_error_message?: string
  created_by: string
  updated_by: string
  created_at: string
  updated_at: string
}

export interface ResearchScheduleRun {
  id: string
  tenant_id: string
  schedule_id: string
  scheduled_for: string
  status: 'PROCESSING' | 'SUCCEEDED' | 'FAILED'
  provider: string
  query_hash: string
  result_count: number
  imported_count: number
  duplicate_count: number
  skipped_count: number
  error_code?: string
  error_message?: string
  requested_by: string
  started_at: string
  finished_at?: string
  created_at: string
  updated_at: string
}
