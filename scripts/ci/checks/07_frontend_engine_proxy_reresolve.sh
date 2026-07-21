#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"

challenge_after_recreate="$(http_get "$CI_FRONTEND_URL/engine/v1/challenge")"
require_challenge_shape "$challenge_after_recreate"
