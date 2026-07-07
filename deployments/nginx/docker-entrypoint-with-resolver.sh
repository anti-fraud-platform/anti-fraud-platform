#!/bin/sh

set -eu

if [ -z "${UPSTREAM_RESOLVER:-}" ]; then
  UPSTREAM_RESOLVER="$(awk '/^nameserver[[:space:]]+/ { print $2; exit }' /etc/resolv.conf)"
  export UPSTREAM_RESOLVER
fi

if [ -z "${UPSTREAM_RESOLVER:-}" ]; then
  echo "Failed to determine UPSTREAM_RESOLVER from /etc/resolv.conf" >&2
  exit 1
fi

exec /docker-entrypoint.sh "$@"
