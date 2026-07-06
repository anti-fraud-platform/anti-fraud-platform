#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly REPO_ROOT="$(cd -- "$SCRIPT_DIR/../.." && pwd)"
source "$SCRIPT_DIR/lib/common.sh"

TARGET_IP="${1:-8.8.8.8}"
CAMPAIGN_ID="geoip-e2e-$(date +%s)"
readonly COUNTRY_DB="$REPO_ROOT/geoip/GeoLite2-Country.mmdb"
readonly CITY_DB="$REPO_ROOT/geoip/GeoLite2-City.mmdb"
readonly ASN_DB="$REPO_ROOT/geoip/GeoLite2-ASN.mmdb"

require_file "$COUNTRY_DB"
require_file "$CITY_DB"
require_file "$ASN_DB"
require_public_ip "$TARGET_IP"

echo "==> Direct lookup from all three MaxMind databases"
lookup_json="$(
  cd "$REPO_ROOT" &&
    env GOCACHE="$REPO_ROOT/.cache/go-build" GOMODCACHE="$REPO_ROOT/.cache/go-mod" \
      go run ./cmd/geoiplookup \
        -ip "$TARGET_IP" \
        -country-db "$COUNTRY_DB" \
        -city-db "$CITY_DB" \
        -asn-db "$ASN_DB"
)"
echo "$lookup_json" | python3 -m json.tool

echo
echo "==> Fetch challenge from engine"
challenge_json="$(curl -fsS http://localhost:9090/v1/challenge)"
challenge_parts="$(
  python3 - "$challenge_json" <<'PY'
import hashlib
import json
import sys

challenge = json.loads(sys.argv[1])
nonce = challenge["nonce"]
challenge_id = challenge["challenge_id"]
token = hashlib.sha256(f"{nonce}:af-js-check-v1".encode()).hexdigest()
print(f"{challenge_id}|{token}")
PY
)"
challenge_id="${challenge_parts%%|*}"
challenge_token="${challenge_parts#*|}"

sleep 0.25

echo
echo "==> Send click through nginx -> engine with X-Forwarded-For=$TARGET_IP"
declare -a curl_headers=()
while IFS= read -r line; do
  curl_headers+=("$line")
done < <(browser_headers)
curl_headers+=(-H "X-Forwarded-For: $TARGET_IP")

click_json="$(
  curl -fsS -X POST http://localhost:9090/click \
    "${curl_headers[@]}" \
    -d "{\"campaign_id\":\"$CAMPAIGN_ID\",\"challenge_id\":\"$challenge_id\",\"challenge_token\":\"$challenge_token\"}"
)"
echo "$click_json" | python3 -m json.tool

echo
echo "==> Wait for batch logger to flush the row"
wait_for_logged_campaign "$CAMPAIGN_ID"

echo
echo "==> Read stored enrichment from Postgres"
db_csv="$(
  docker exec antifraud-postgres psql -U antifraud -d analytics --csv -c \
    "SELECT ip, country, city, asn_number, asn_org, reason, is_bot FROM click_logs WHERE campaign_id = '${CAMPAIGN_ID}' ORDER BY processed_at DESC LIMIT 1;"
)"
echo "$db_csv"

echo
echo "==> Compare direct mmdb lookup with stored DB row"
python3 - "$lookup_json" "$db_csv" <<'PY'
import csv
import io
import json
import sys

lookup = json.loads(sys.argv[1])
reader = csv.DictReader(io.StringIO(sys.argv[2]))
row = next(reader, None)
if row is None:
    raise SystemExit("No row returned from click_logs")

checks = [
    ("ip", row["ip"], lookup["ip"]),
    ("country", row["country"], lookup["country_iso"]),
    ("city", row["city"], lookup["city_name"]),
    ("asn_number", row["asn_number"], str(lookup["asn_number"])),
    ("asn_org", row["asn_org"], lookup["asn_org"]),
]

failures = []
for label, actual, expected in checks:
    if actual != expected:
        failures.append(f"{label}: db={actual!r} expected={expected!r}")

if row["reason"] != "allowed":
    failures.append(f"reason: db={row['reason']!r} expected='allowed'")
if row["is_bot"] not in {"f", "false", "False"}:
    failures.append(f"is_bot: db={row['is_bot']!r} expected false")

if failures:
    raise SystemExit("GeoIP e2e mismatch:\n- " + "\n- ".join(failures))

print("GeoIP e2e check passed.")
PY

echo
echo "Campaign ID: $CAMPAIGN_ID"
