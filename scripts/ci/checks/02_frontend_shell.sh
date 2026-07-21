#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"

require_page_contains \
  "$CI_FRONTEND_URL" \
  '<div id="root"' \
  "Frontend did not serve the expected React shell"
