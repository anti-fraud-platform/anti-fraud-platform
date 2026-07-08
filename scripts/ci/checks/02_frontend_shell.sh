#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"

require_page_contains \
  "http://127.0.0.1/" \
  '<div id="root"' \
  "Frontend did not serve the expected React shell" \
  "frontend"
