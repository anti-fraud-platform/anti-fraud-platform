#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"

wait_for_url "http://localhost:3001"
wait_for_url "http://localhost:8082/v1/analytics/stats"
wait_for_url "http://localhost:9090/"
