#!/usr/bin/env sh

set -eu

DEPLOY_BRANCH="${DEPLOY_BRANCH:-${CI_DEFAULT_BRANCH:-main}}"
REMOTE_SUDO="${REMOTE_SUDO:-}"
KEY_PATH="${SSH_KEY_PATH:-${HOME}/.ssh/gitlab_deploy_key}"

remote_inner_command="
set -e
cd '$DEPLOY_PATH'
git fetch origin
git checkout '$DEPLOY_BRANCH'
git pull --ff-only origin '$DEPLOY_BRANCH'
bash scripts/deploy/vm_refresh_stack.sh
bash scripts/deploy/vm_smoke.sh
"

if [ -n "$REMOTE_SUDO" ]; then
  remote_command="$REMOTE_SUDO sh -lc $(printf '%s' "$remote_inner_command" | sed \"s/'/'\\\\''/g; 1s/^/'/; \$s/\$/'/\")"
else
  remote_command="$remote_inner_command"
fi

ssh -p "${VM_PORT:-22}" "$VM_USER@$VM_HOST" \
  -i "$KEY_PATH" \
  -o IdentitiesOnly=yes \
  -o BatchMode=yes \
  -o PreferredAuthentications=publickey \
  "$remote_command"
