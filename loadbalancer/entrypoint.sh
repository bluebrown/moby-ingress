#!/bin/sh

set -eu

# set the last seen hash to the curent config hash
last_seen_hash="$(md5sum haproxy.cfg | cut -d ' ' -f 1)"

fetch_config() {
  # fetch config, if it fails wait 3 seonds and
  # reuturn with non-zero exit code
  if ! curl -H "Config-Hash: $last_seen_hash" --url "$1" --max-time 70 --output new.cfg --fail --silent --show-error --location; then
    # if curl failed, sleep 3 seconds
    # to avoid hammering the server
    sleep 3
    return 1
  fi

  # compute the next hash from the new config
  next_hash=$(md5sum new.cfg | cut -d ' ' -f 1)

  # compare the checksums of the last and the next config
  # if they are the same, the config has not changed
  # and we can return early with non zero exit code
  if [ "$last_seen_hash" = "$next_hash" ]; then
    return 2
  fi

  # otherwise, mark the next hash as last seen hash
  # even thouhg the config may not be valid
  # this will avoid hammering the server if it sent us
  # an invalid config
  last_seen_hash="$next_hash"

  # validate the config file using haproxy's syntax checker
  # if its not valid we return early with non zero exit code
  if ! haproxy -c -f new.cfg; then
    return 3
  fi

  # if all checks pass, we can replace the old config
  # and indicate that we need to restart by returning 0
  cp new.cfg haproxy.cfg
  return 0
}

# srape the config from the manager if it has changed and is valid,
# SIGUSR2 is sent to process 1 process 1 is the haproxy master
# and this signal with cause a hitless reload
scrape_config() {
  while :; do
    if fetch_config "$1"; then
      kill -s SIGUSR2 1
    fi
  done
}

# try to fetch the initial config from the manager
# before booting haproxy to avoid an instant reload
fetch_config "$MANAGER_ENDPOINT" || true

# start a long polling loop to fetch config
scrape_config "$MANAGER_ENDPOINT" &

# exec original entrypoint to make it pid 1
exec docker-entrypoint.sh "$@"
