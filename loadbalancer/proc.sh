#!/bin/sh

set -e

conf="/usr/local/etc/haproxy/haproxy.cfg"
backup="/usr/local/etc/haproxy/previous.cfg"

# take the inital checksum
sum="$(md5sum "$conf")"

while :
do
    # sleep until next iteration
    sleep "${SCRAPE_INTERVAL:=60}"

    # try to fetch the config
    if ! curl -s -f "${MANAGER_ENDPOINT:=http://manager:8080}" > "$conf";
    then
        cp "$backup" "$conf"
        continue
    fi

    # compare check sums
    old_sum="$sum"
    sum="$(md5sum "$conf")"
    if [ "$old_sum" = "$sum" ]
    then 
        continue
    fi

    # test if file is valid
    if ! haproxy -c -f "$conf";
    then
        cp "$backup" "$conf"
        sum="$old_sum"
        continue
    fi

    # if file has been successfully fetched, 
    # has a different checksum and is valid,
    # reload the worker
    kill -s SIGUSR2 1

done
