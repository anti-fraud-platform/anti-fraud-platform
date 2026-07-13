#!/usr/bin/env bash

set -euo pipefail

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
