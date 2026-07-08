#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"

compose up --build -d

# Validate the live nginx config after the rebuild and reload it once so the
# running service definitely uses the current config.
if compose ps --services --filter status=running | grep -qx "nginx_engine"; then
  compose exec -T nginx_engine nginx -t
  compose exec -T nginx_engine nginx -s reload
fi

compose ps
