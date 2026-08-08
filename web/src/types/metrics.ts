export type DecimalValue = string | number

export interface CampaignMetric {
  campaign_id: string
  campaign_name: string
  country: string
  currency: string
  spend: DecimalValue
  daily_budget: DecimalValue
  impressions: number
  clicks: number
  installs: number
  registrations: number
  active_users: number
  payers: number
  revenue_d1: DecimalValue
  revenue_d3: DecimalValue
  revenue_d7: DecimalValue
  ctr: DecimalValue
  cvr: DecimalValue
  cpi: DecimalValue
  cpa: DecimalValue
  payer_rate: DecimalValue
  roas_d1: DecimalValue
  roas_d3: DecimalValue
  roas_d7: DecimalValue
  budget_consumption_rate: DecimalValue
  ltv_d7: DecimalValue
}

export interface Overview extends CampaignMetric {
  high_risk_campaigns: number
  attribution_anomalies: number
  fatigued_creatives: number
}

export interface TrendPoint {
  date: string
  spend: DecimalValue
  revenue_d1: DecimalValue
  revenue_d7: DecimalValue
  cpi: DecimalValue
  roas_d1: DecimalValue
  roas_d7: DecimalValue
}

export interface Rule {
  id: string
  code: string
  category: string
  name: string
  severity: string
  threshold: number
  consecutive_days: number
  enabled: boolean
}

export interface RuleFinding {
  id: string
  campaign_id: string
  rule_code: string
  severity: string
  title: string
  description: string
  evidence: Record<string, unknown>
}

export interface Benchmark { id: string; metric_code: string; value: DecimalValue; period_start: string; period_end: string }
export interface RuleOverview { rules: Rule[]; benchmarks: Benchmark[]; findings: RuleFinding[] }

export interface AttributionFinding {
  id: string
  campaign_id: string
  campaign_name: string
  rule_code: string
  severity: string
  title: string
  description: string
  difference_rate: DecimalValue
  evidence: Record<string, unknown>
}

export interface CreativeFinding {
  id: string
  campaign_id: string
  creative_id: string
  creative_name: string
  rule_code: string
  severity: string
  title: string
  description: string
  fatigue_score: DecimalValue
  ctr_change_7d: DecimalValue
  frequency: DecimalValue
  evidence: Record<string, unknown>
}

export interface RecalculateResult {
  calculated_rows: number
  business_findings: number
  attribution_findings: number
  creative_findings: number
}
