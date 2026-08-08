#!/usr/bin/env sh
set -eu

API_BASE_URL="${API_BASE_URL:-http://localhost:8080/api/v1}"
go run ./cmd/demo-data --out examples/generated

LOGIN_FILE="$(mktemp)"
trap 'rm -f "$LOGIN_FILE"' EXIT
curl -fsS "$API_BASE_URL/auth/login" -H 'Content-Type: application/json' \
  -d '{"username":"operator","password":"Demo@123456"}' -o "$LOGIN_FILE"
TOKEN="$(node -p "JSON.parse(require('fs').readFileSync('$LOGIN_FILE','utf8')).data.access_token")"

import_file() {
  endpoint="$1"
  file="$2"
  curl -fsS "$API_BASE_URL/imports/$endpoint" \
    -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    --data-binary "@$file"
  printf '\n'
}

import_file ad-metrics examples/generated/meta_ad_metrics.json
import_file ad-metrics examples/generated/google_ad_metrics.json
import_file ad-metrics examples/generated/tiktok_ad_metrics.json
import_file mmp-metrics examples/generated/mmp_metrics.json
import_file game-revenue examples/generated/game_revenue.json
import_file creative-metrics examples/generated/creative_metrics.json
curl -fsS -X POST "$API_BASE_URL/metrics/recalculate?game_id=30000000-0000-4000-8000-000000000001" \
  -H "Authorization: Bearer $TOKEN"
printf '\n'
curl -fsS -X POST "$API_BASE_URL/analysis/business" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"game_id":"30000000-0000-4000-8000-000000000001","campaign_id":"50000000-0000-4000-8000-000000000001","analysis_date":"2026-07-30"}'
printf '\nDemo import and deterministic analysis completed. Re-running this script is idempotent.\n'
