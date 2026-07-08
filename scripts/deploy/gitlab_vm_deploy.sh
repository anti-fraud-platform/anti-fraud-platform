#!/usr/bin/env sh

set -eu

DEPLOY_BRANCH="${DEPLOY_BRANCH:-${CI_DEFAULT_BRANCH:-main}}"

ssh -p "${VM_PORT:-22}" "$VM_USER@$VM_HOST" \
  "set -e; \
  cd '$DEPLOY_PATH'; \
  git fetch origin; \
  git checkout '$DEPLOY_BRANCH'; \
  git pull --ff-only origin '$DEPLOY_BRANCH'; \
  bash scripts/deploy/vm_refresh_stack.sh; \
  bash scripts/deploy/vm_smoke.sh"
