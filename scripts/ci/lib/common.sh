#!/usr/bin/env bash

set -euo pipefail

readonly COMMON_LIB_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly REPO_ROOT="$(cd -- "$COMMON_LIB_DIR/../../.." && pwd)"
readonly CI_COMPOSE_FILE="${ANTI_FRAUD_CI_COMPOSE_FILE:-$REPO_ROOT/docker-compose.ci.yml}"
readonly SMOKE_TRANSPORT_MODE="${SMOKE_TRANSPORT:-host}"
readonly CI_FRONTEND_URL="${CI_FRONTEND_URL:-http://localhost:13001}"
readonly CI_ANALYTICS_URL="${CI_ANALYTICS_URL:-http://localhost:18082}"
readonly CI_NGINX_URL="${CI_NGINX_URL:-http://localhost:19090}"

browser_like_headers=(
  "Content-Type: application/json"
  "User-Agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
  "Accept: */*"
  "Accept-Language: en-US,en;q=0.9"
  "Accept-Encoding: gzip, deflate, br"
  "Sec-Fetch-Site: same-origin"
  "Sec-Fetch-Mode: cors"
  'Sec-Ch-Ua: "Chromium";v="126", "Not.A/Brand";v="99", "Google Chrome";v="126"'
)

AUTH_TOKEN=""

get_admin_token() {
  local login_response
  local service target

  if [[ -n "$AUTH_TOKEN" ]]; then
    echo "$AUTH_TOKEN"
    return
  fi

  if [[ "$SMOKE_TRANSPORT_MODE" == "compose_exec" ]]; then
    IFS='|' read -r service target <<<"$(resolve_transport_target "$CI_ANALYTICS_URL/v1/auth/login")"
    login_response="$(compose_exec_http \
      "POST" \
      "$service" \
      "$target" \
      '{"username":"admin","password":"admin123"}' \
      "Content-Type: application/json" 2>/dev/null || true)"
  else
    login_response="$(curl -s -X POST "$CI_ANALYTICS_URL/v1/auth/login" \
      -H "Content-Type: application/json" \
      -d '{"username":"admin","password":"admin123"}' 2>/dev/null || true)"
  fi

  AUTH_TOKEN="$(printf '%s' "$login_response" | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])" 2>/dev/null || true)"
  echo "$AUTH_TOKEN"
}

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -f "$CI_COMPOSE_FILE" "$@"
    return
  fi

  docker-compose -f "$CI_COMPOSE_FILE" "$@"
}

resolve_transport_target() {
  local url="$1"
  local base path service internal_url

  case "$url" in
    "$CI_FRONTEND_URL"/*)
      base="$CI_FRONTEND_URL"
      service="frontend"
      internal_url="http://127.0.0.1"
      ;;
    "$CI_FRONTEND_URL")
      base="$CI_FRONTEND_URL"
      service="frontend"
      internal_url="http://127.0.0.1"
      ;;
    "$CI_ANALYTICS_URL"/*)
      base="$CI_ANALYTICS_URL"
      service="analytics"
      internal_url="http://127.0.0.1:8081"
      ;;
    "$CI_ANALYTICS_URL")
      base="$CI_ANALYTICS_URL"
      service="analytics"
      internal_url="http://127.0.0.1:8081"
      ;;
    "$CI_NGINX_URL"/*)
      base="$CI_NGINX_URL"
      service="nginx_engine"
      internal_url="http://127.0.0.1:9090"
      ;;
    "$CI_NGINX_URL")
      base="$CI_NGINX_URL"
      service="nginx_engine"
      internal_url="http://127.0.0.1:9090"
      ;;
    *)
      echo "Unsupported smoke URL for compose_exec transport: $url" >&2
      return 1
      ;;
  esac

  path="${url#"$base"}"
  if [[ -z "$path" ]]; then
    path="/"
  fi

  printf '%s|%s%s\n' "$service" "$internal_url" "$path"
}

compose_exec_http() {
  local method="$1"
  local service="$2"
  local target_url="$3"
  local body="${4:-}"
  shift 4
  local -a headers=( "$@" )
  local -a cmd=( wget -qO- )
  local header
  local quoted_cmd

  if [[ "$method" == "POST" ]]; then
    cmd+=( --post-data="$body" )
  fi

  for header in "${headers[@]}"; do
    cmd+=( --header="$header" )
  done

  cmd+=( "$target_url" )
  printf -v quoted_cmd '%q ' "${cmd[@]}"
  compose exec -T "$service" sh -lc "$quoted_cmd"
}

http_get() {
  local url="$1"
  local service target
  local -a curl_args=( -fsS )
  local -a headers=()

  if [[ "$url" == "$CI_ANALYTICS_URL/"* ]]; then
    local token
    token="$(get_admin_token)"
    if [[ -n "$token" ]]; then
      headers+=( "Authorization: Bearer $token" )
      curl_args+=( -H "Authorization: Bearer $token" )
    fi
  fi

  if [[ "$SMOKE_TRANSPORT_MODE" == "compose_exec" ]]; then
    IFS='|' read -r service target <<<"$(resolve_transport_target "$url")"
    compose_exec_http "GET" "$service" "$target" "" "${headers[@]}"
    return
  fi

  curl "${curl_args[@]}" "$url"
}

http_post_json() {
  local url="$1"
  local body="$2"
  shift 2
  local -a headers=( "$@" )
  local service target
  local -a curl_args=( -fsS -X POST )
  local header

  if [[ "$url" == "$CI_ANALYTICS_URL/"* ]]; then
    local token
    token="$(get_admin_token)"
    if [[ -n "$token" ]]; then
      headers+=( "Authorization: Bearer $token" )
      curl_args+=( -H "Authorization: Bearer $token" )
    fi
  fi

  if [[ "$SMOKE_TRANSPORT_MODE" == "compose_exec" ]]; then
    IFS='|' read -r service target <<<"$(resolve_transport_target "$url")"
    compose_exec_http "POST" "$service" "$target" "$body" "${headers[@]}"
    return
  fi

  for header in "${headers[@]}"; do
    curl_args+=( -H "$header" )
  done

  curl_args+=( -d "$body" "$url" )
  curl "${curl_args[@]}"
}

wait_for_url() {
  local url="$1"
  local attempts="${2:-30}"
  local sleep_seconds="${3:-2}"

  for _ in $(seq 1 "$attempts"); do
    if http_get "$url" >/dev/null; then
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

  body="$(http_get "$url")"
  if [[ "$body" != *"$expected_fragment"* ]]; then
    echo "$description" >&2
    exit 1
  fi
}

require_json_fields() {
  local url="$1"
  shift
  local payload

  payload="$(http_get "$url")"
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
    stats="$(http_get "$CI_ANALYTICS_URL/v1/analytics/stats")"

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
