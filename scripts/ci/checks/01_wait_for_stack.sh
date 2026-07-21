#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"

wait_for_url "$CI_FRONTEND_URL"
wait_for_url "$CI_ANALYTICS_URL/v1/analytics/stats"
wait_for_url "$CI_NGINX_URL/"
