#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"

wait_for_url "http://127.0.0.1/" "frontend"
wait_for_url "http://127.0.0.1:8081/v1/analytics/stats" "analytics"
wait_for_url "http://127.0.0.1:9090/" "nginx_engine"
