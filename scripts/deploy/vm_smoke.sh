#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"

echo "Smoke: waiting for engine challenge endpoint"
wait_for_url "http://localhost:9090/v1/challenge" 40 2
echo "Smoke: waiting for analytics auth endpoint"
wait_for_admin_login 40 2
echo "Smoke: waiting for Prometheus readiness"
wait_for_url "http://localhost:9091/-/ready" 40 2
echo "Smoke: waiting for Grafana health endpoint"
wait_for_url "http://localhost:3000/api/health" 40 2

echo "Smoke: verifying engine challenge payload"
curl -fsS http://localhost:9090/v1/challenge >/dev/null
echo "Smoke: verifying analytics stats payload"
http_get_analytics_authed "/v1/analytics/stats" >/dev/null
echo "Smoke: verifying Prometheus readiness payload"
curl -fsS http://localhost:9091/-/ready >/dev/null

echo "Smoke: verifying Grafana health payload"
grafana_health="$(curl -fsS http://localhost:3000/api/health)"
if ! printf '%s\n' "$grafana_health" | grep -Eq '"database"[[:space:]]*:[[:space:]]*"ok"'; then
	echo "Unexpected Grafana health payload: $grafana_health" >&2
	exit 1
fi

echo "Smoke: all VM checks passed"
