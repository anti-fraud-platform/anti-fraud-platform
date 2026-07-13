#!/usr/bin/env bash

set -euo pipefail

readonly COMMON_LIB_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly REPO_ROOT="$(cd -- "$COMMON_LIB_DIR/../.." && pwd)"
readonly CI_COMPOSE_FILE="${ANTI_FRAUD_CI_COMPOSE_FILE:-$REPO_ROOT/docker-compose.ci.yml}"

browser_like_headers=(
  -H "Content-Type: application/json"
  -H "User-Agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
  -H "Accept: */*"
  -H "Accept-Language: en-US,en;q=0.9"
  -H "Accept-Encoding: gzip, deflate, br"
  -H "Sec-Fetch-Site: same-origin"
  -H "Sec-Fetch-Mode: cors"
  -H 'Sec-Ch-Ua: "Chromium";v="126", "Not.A/Brand";v="99", "Google Chrome";v="126"'
)

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -f "$CI_COMPOSE_FILE" "$@"
    return
  fi

  docker-compose -f "$CI_COMPOSE_FILE" "$@"
}

wait_for_url() {
  local url="$1"
  local attempts="${2:-30}"
  local sleep_seconds="${3:-2}"

  for _ in $(seq 1 "$attempts"); do
    if curl -fsS "$url" >/dev/null; then
      return 0
    fi
    sleep "$sleep_seconds"
  done

  echo "Timed out waiting for $url" >&2
  return 1
}

require_page_contains() {
  local url="$1"
  local expected_fragment="$2"
  local description="$3"
  local body

  body="$(curl -fsS "$url")"
  if [[ "$body" != *"$expected_fragment"* ]]; then
    echo "$description" >&2
    exit 1
  fi
}

require_json_fields() {
  local url="$1"
  shift
  local payload

  payload="$(curl -fsS "$url")"
  python3 - "$payload" "$@" <<'PY'
import json
import sys

payload = json.loads(sys.argv[1])
required_fields = sys.argv[2:]
missing = [field for field in required_fields if field not in payload]

if missing:
    raise SystemExit(f"JSON payload is missing fields: {missing}")
PY
}

require_challenge_shape() {
  local payload="$1"

  python3 - "$payload" <<'PY'
import json
import sys

challenge = json.loads(sys.argv[1])
if not challenge.get("challenge_id") or not challenge.get("nonce"):
    raise SystemExit("challenge response is missing challenge_id or nonce")
PY
}

require_flagged_click() {
  local payload="$1"

  python3 - "$payload" <<'PY'
import json
import sys

click = json.loads(sys.argv[1])
if click.get("status") != "flagged":
    raise SystemExit(f"expected click to be flagged, got: {click}")
PY
}

wait_for_blocked_challenge_metrics() {
  local attempts="${1:-20}"
  local sleep_seconds="${2:-1}"

  for _ in $(seq 1 "$attempts"); do
    local stats
    stats="$(curl -fsS http://localhost:8082/v1/analytics/stats)"

    if python3 - "$stats" <<'PY'
import json
import sys

stats = json.loads(sys.argv[1])
reason_breakdown = stats.get("reason_breakdown")
js_challenge_blocked = stats.get("js_challenge_blocked")

if not isinstance(reason_breakdown, dict):
    raise SystemExit(1)
if reason_breakdown.get("no_js_challenge", 0) < 1:
    raise SystemExit(1)
if not isinstance(js_challenge_blocked, int) or js_challenge_blocked < 1:
    raise SystemExit(1)
PY
    then
      return 0
    fi

    sleep "$sleep_seconds"
  done

  echo "Timed out waiting for analytics to reflect a blocked JS challenge click" >&2
  return 1
}
