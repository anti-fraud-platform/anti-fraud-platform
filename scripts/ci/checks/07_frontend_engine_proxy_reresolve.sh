#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"

challenge_after_recreate="$(curl -fsS http://localhost:3001/engine/v1/challenge)"
require_challenge_shape "$challenge_after_recreate"
