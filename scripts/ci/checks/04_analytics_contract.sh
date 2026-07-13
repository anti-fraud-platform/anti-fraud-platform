#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"

require_json_fields \
  "http://localhost:8082/v1/analytics/stats" \
  "reason_breakdown" \
  "js_challenge_blocked" \
  "header_heuristic_blocked" \
  "blocked_count" \
  "allowed_count"
