#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"

compose up --build -d --force-recreate engine
sleep 2

challenge_after_recreate="$(curl -fsS http://localhost:9090/v1/challenge)"
require_challenge_shape "$challenge_after_recreate"
