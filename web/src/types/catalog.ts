export interface Game {
  id: string; code: string; name: string; package_name: string; timezone: string; currency: string; status: string
}
export interface Channel { id: string; code: string; name: string; provider: string; status: string }
export interface Campaign { id: string; game_id: string; channel_id: string; external_id: string; name: string; country: string; daily_budget: string; currency: string; status: string }
export interface Creative { id: string; campaign_id: string; external_id: string; name: string; type: string; status: string }
export interface ImportJob {
  id: string; game_id: string; import_type: string; source: string; file_name: string; period_start: string; status: string;
  total_rows: number; imported_rows: number; skipped_rows: number; error_message?: string; created_at: string
}
export interface MMPConnection {
  id: string; game_id: string; provider: 'APPSFLYER' | 'ADJUST'; external_app_id: string; status: 'ACTIVE' | 'DISABLED';
  credential_configured: boolean; health: 'READY' | 'NOT_CONFIGURED' | 'DISABLED'; last_sync_at?: string; created_at: string; updated_at: string
}
export interface MMPSyncRun {
  id: string; connection_id: string; provider: 'APPSFLYER' | 'ADJUST'; period_start: string; period_end: string;
  status: 'PROCESSING' | 'SUCCEEDED' | 'FAILED'; import_job_id?: string; source_rows: number; normalized_rows: number; skipped_rows: number;
  warning_message?: string; error_code?: string; error_message?: string; started_at: string; finished_at?: string
}
