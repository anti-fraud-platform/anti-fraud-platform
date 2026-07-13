#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"

challenge_json="$(curl -fsS http://localhost:9090/v1/challenge)"
require_challenge_shape "$challenge_json"

flagged_click_json="$(
  curl -fsS -X POST http://localhost:9090/click \
    "${browser_like_headers[@]}" \
    -d '{"campaign_id":"ci_no_challenge"}'
)"
require_flagged_click "$flagged_click_json"

wait_for_blocked_challenge_metrics
