#!/bin/sh

set -eu

fetch_config() {
  # fetch config
  if ! curl --url "$1" --max-time 70 --output new.cfg --fail --silent --show-error --location; then
    sleep 1
    return 1
  fi
  # compare checksums
  if [ "$(md5sum haproxy.cfg | cut -d ' ' -f 1)" = "$(md5sum new.cfg | cut -d ' ' -f 1)" ]; then
    return 2
  fi
  # validate
  if ! haproxy -c -f new.cfg; then
    return 3
  fi
  # use valid file once all checks are passed
  cp new.cfg haproxy.cfg
  return 0
}

scrape_config() {
  while :; do
    if fetch_config "$1"; then
      kill -s SIGUSR2 1
      continue
    fi
  done
}

# start a long polling loop to fetch config
scrape_config "$MANAGER_ENDPOINT" &

# exec original entrypoint to make it pid 1
exec docker-entrypoint.sh "$@"
