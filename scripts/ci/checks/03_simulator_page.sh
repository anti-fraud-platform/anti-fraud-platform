#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"

require_page_contains \
  "http://localhost:9090/" \
  "Anti-Fraud Click Simulator" \
  "Engine simulator page did not load on :9090"
