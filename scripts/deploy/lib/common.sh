#!/usr/bin/env bash

set -euo pipefail

readonly DEPLOY_ANALYTICS_URL="${DEPLOY_ANALYTICS_URL:-http://localhost:8082}"
AUTH_TOKEN=""

wait_for_url() {
	local url="$1"
	local attempts="${2:-30}"
	local sleep_seconds="${3:-2}"

	for _ in $(seq 1 "$attempts"); do
		if curl -fsS "$url" >/dev/null 2>&1; then
			return 0
		fi
		sleep "$sleep_seconds"
	done

	echo "Timed out waiting for $url" >&2
	return 1
}

get_admin_token() {
	if [[ -n "$AUTH_TOKEN" ]]; then
		echo "$AUTH_TOKEN"
		return
	fi

	AUTH_TOKEN="$(
		curl -fsS -X POST "$DEPLOY_ANALYTICS_URL/v1/auth/login" \
			-H "Content-Type: application/json" \
			-d '{"username":"admin","password":"admin123"}' \
		| python3 -c "import sys, json; print(json.load(sys.stdin)['token'])"
	)"
	echo "$AUTH_TOKEN"
}

http_get_analytics_authed() {
	local path="$1"
	local token

	token="$(get_admin_token)"
	curl -fsS "$DEPLOY_ANALYTICS_URL$path" \
		-H "Authorization: Bearer $token"
}
