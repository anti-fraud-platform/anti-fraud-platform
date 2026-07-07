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

# nginx accepts IPv6 resolvers in bracketed form. Railway can expose the
# first nameserver from /etc/resolv.conf as a raw IPv6 literal such as
# fd12::10, so normalize it before envsubst renders the final config.
case "$UPSTREAM_RESOLVER" in
  *:*)
    case "$UPSTREAM_RESOLVER" in
      \[*\]) ;;
      *) UPSTREAM_RESOLVER="[$UPSTREAM_RESOLVER]" ;;
    esac
    export UPSTREAM_RESOLVER
    ;;
esac

exec /docker-entrypoint.sh "$@"
