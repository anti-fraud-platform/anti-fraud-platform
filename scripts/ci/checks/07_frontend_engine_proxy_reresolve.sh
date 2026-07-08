#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"

challenge_after_recreate="$(fetch_url "http://127.0.0.1/engine/v1/challenge" "frontend")"
require_challenge_shape "$challenge_after_recreate"
