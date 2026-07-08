#!/usr/bin/env sh

set -eu

apk add --no-cache openssh-client

eval "$(ssh-agent -s)"

if [ -f "${SSH_PRIVATE_KEY:-}" ]; then
  ssh-add "$SSH_PRIVATE_KEY"
else
  printf '%s\n' "$SSH_PRIVATE_KEY" | tr -d '\r' | ssh-add -
fi

mkdir -p "$HOME/.ssh"
chmod 700 "$HOME/.ssh"
ssh-keyscan -p "${VM_PORT:-22}" "$VM_HOST" >> "$HOME/.ssh/known_hosts"
chmod 644 "$HOME/.ssh/known_hosts"
