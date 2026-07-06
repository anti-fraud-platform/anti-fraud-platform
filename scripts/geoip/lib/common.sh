#!/usr/bin/env bash

set -euo pipefail

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
    return
  fi

  docker-compose "$@"
}

require_file() {
  local path="$1"
  if [[ ! -f "$path" ]]; then
    echo "Missing file: $path" >&2
    exit 1
  fi
}

require_public_ip() {
  local ip="$1"

  python3 - "$ip" <<'PY'
import ipaddress
import sys

ip = ipaddress.ip_address(sys.argv[1])
if not ip.is_global:
    raise SystemExit(f"{ip} is not a public global IP address")
PY
}

wait_for_logged_campaign() {
  local campaign_id="$1"
  local attempts="${2:-20}"
  local sleep_seconds="${3:-1}"

  for _ in $(seq 1 "$attempts"); do
    local row_count
    row_count="$(
      docker exec antifraud-postgres psql -U antifraud -d analytics -t -A -c \
        "SELECT COUNT(*) FROM click_logs WHERE campaign_id = '${campaign_id}';" | tr -d '[:space:]'
    )"

    if [[ "${row_count:-0}" =~ ^[0-9]+$ ]] && (( row_count > 0 )); then
      return 0
    fi

    sleep "$sleep_seconds"
  done

  echo "Timed out waiting for click_logs row for campaign_id=$campaign_id" >&2
  return 1
}

browser_headers() {
  cat <<'EOF'
-H
Content-Type: application/json
-H
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36
-H
Accept: */*
-H
Accept-Language: en-US,en;q=0.9
-H
Accept-Encoding: gzip, deflate, br
-H
Sec-Fetch-Site: same-origin
-H
Sec-Fetch-Mode: cors
-H
Sec-Ch-Ua: "Chromium";v="126", "Not.A/Brand";v="99", "Google Chrome";v="126"
EOF
}
