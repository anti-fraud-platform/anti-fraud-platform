#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"

wait_for_url "http://localhost:9090/v1/challenge" 40 2
wait_for_url "http://localhost:8082/v1/analytics/stats" 40 2
wait_for_url "http://localhost:9091/-/ready" 40 2
wait_for_url "http://localhost:3000/api/health" 40 2

curl -fsS http://localhost:9090/v1/challenge >/dev/null
curl -fsS http://localhost:8082/v1/analytics/stats >/dev/null
curl -fsS http://localhost:9091/-/ready >/dev/null
curl -fsS http://localhost:3000/api/health | grep -q '"database":"ok"'
