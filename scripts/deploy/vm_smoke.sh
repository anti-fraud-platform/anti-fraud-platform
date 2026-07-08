#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"

wait_for_url "http://localhost:3001"
wait_for_url "http://localhost:8082/v1/analytics/stats"
wait_for_url "http://localhost:9090/v1/challenge"

curl -fsS http://localhost:3001 | grep -q '<div id="root"'
curl -fsS http://localhost:8082/v1/analytics/stats | grep -q '"reason_breakdown"'
curl -fsS http://localhost:9090/v1/challenge | grep -q '"challenge_id"'

curl -fsS -X POST http://localhost:9090/click \
  -H "Content-Type: application/json" \
  -d '{"campaign_id":"gitlab_cd"}' | grep -q '"status":"flagged"'
