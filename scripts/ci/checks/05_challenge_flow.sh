#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"

challenge_json="$(http_get "$CI_NGINX_URL/v1/challenge")"
require_challenge_shape "$challenge_json"

flagged_click_json="$(http_post_json \
  "$CI_NGINX_URL/click" \
  '{"campaign_id":"ci_no_challenge"}' \
  "${browser_like_headers[@]}")"
require_flagged_click "$flagged_click_json"

wait_for_blocked_challenge_metrics
