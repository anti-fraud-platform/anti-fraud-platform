#!/usr/bin/env sh

set -eu

apk add --no-cache openssh-client

eval "$(ssh-agent -s)"

mkdir -p "$HOME/.ssh"
chmod 700 "$HOME/.ssh"

KEY_PATH="${HOME}/.ssh/gitlab_deploy_key"

if [ -f "${SSH_PRIVATE_KEY:-}" ]; then
  cp "$SSH_PRIVATE_KEY" "$KEY_PATH"
else
  printf '%s\n' "$SSH_PRIVATE_KEY" | tr -d '\r' > "$KEY_PATH"
fi

chmod 600 "$KEY_PATH"
ssh-add "$KEY_PATH"

ssh-keyscan -p "${VM_PORT:-22}" "$VM_HOST" >> "$HOME/.ssh/known_hosts"
chmod 644 "$HOME/.ssh/known_hosts"
