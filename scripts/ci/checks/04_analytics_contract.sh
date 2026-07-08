#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"

require_json_fields \
  "http://127.0.0.1:8081/v1/analytics/stats" \
  "analytics" \
  "reason_breakdown" \
  "js_challenge_blocked" \
  "header_heuristic_blocked" \
  "blocked_count" \
  "allowed_count"
