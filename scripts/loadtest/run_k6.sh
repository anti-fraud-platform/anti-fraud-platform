#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly REPO_ROOT="$(cd -- "$SCRIPT_DIR/../.." && pwd)"

script_name="${1:-k6_real_click_ramp.js}"
shift || true

output_dir="${K6_OUTPUT_DIR:-$REPO_ROOT/loadtest-artifacts}"
mkdir -p "$output_dir"

timestamp="$(date +%Y%m%d-%H%M%S)"
summary_file="$output_dir/${script_name%.js}-$timestamp.json"

if command -v k6 >/dev/null 2>&1; then
	k6 run --summary-export "$summary_file" "$SCRIPT_DIR/$script_name" "$@"
	if [ ! -f "$summary_file" ]; then
		printf 'k6 finished but did not write summary file: %s\n' "$summary_file" >&2
		exit 1
	fi
	printf 'Saved k6 summary to %s\n' "$summary_file"
	exit 0
fi

docker run --rm \
	--user "$(id -u):$(id -g)" \
	-v "$SCRIPT_DIR:/scripts:ro" \
	-v "$output_dir:/results" \
	-e BASE_URL="${BASE_URL:-http://localhost:9090}" \
	-e STAGES="${STAGES:-}" \
	-e START_RATE="${START_RATE:-}" \
	-e PREALLOCATED_VUS="${PREALLOCATED_VUS:-}" \
	-e MAX_VUS="${MAX_VUS:-}" \
	-e SOLVE_DELAY_MS="${SOLVE_DELAY_MS:-}" \
	-e ALLOWED_IPS="${ALLOWED_IPS:-}" \
	-e RATE_LIMIT_IP="${RATE_LIMIT_IP:-}" \
	-e GEO_BLOCKED_IP="${GEO_BLOCKED_IP:-}" \
	-e DURATION="${DURATION:-}" \
	grafana/k6:0.52.0 \
	run --summary-export "/results/$(basename "$summary_file")" "/scripts/$script_name" "$@"

if [ ! -f "$summary_file" ]; then
	printf 'k6 container finished but did not write summary file: %s\n' "$summary_file" >&2
	exit 1
fi

printf 'Saved k6 summary to %s\n' "$summary_file"
