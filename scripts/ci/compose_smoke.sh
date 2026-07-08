#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

checks=(
  "$SCRIPT_DIR/checks/01_wait_for_stack.sh"
  "$SCRIPT_DIR/checks/02_frontend_shell.sh"
  "$SCRIPT_DIR/checks/03_simulator_page.sh"
  "$SCRIPT_DIR/checks/04_analytics_contract.sh"
  "$SCRIPT_DIR/checks/05_challenge_flow.sh"
  "$SCRIPT_DIR/checks/06_nginx_reresolve.sh"
  "$SCRIPT_DIR/checks/07_frontend_engine_proxy_reresolve.sh"
)

for check in "${checks[@]}"; do
  echo
  echo "==> $(basename "$check")"
  bash "$check"
done

echo
echo "All compose smoke checks passed."
