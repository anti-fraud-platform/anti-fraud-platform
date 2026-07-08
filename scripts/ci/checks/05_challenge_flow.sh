#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"

challenge_json="$(fetch_url "http://127.0.0.1:9090/v1/challenge" "nginx_engine")"
require_challenge_shape "$challenge_json"

flagged_click_json="$(post_json "http://127.0.0.1:9090/click" '{"campaign_id":"ci_no_challenge"}' "nginx_engine")"
require_flagged_click "$flagged_click_json"

wait_for_blocked_challenge_metrics
