#!/bin/sh

set -ex

fetch_config()
{
    # fetch config
    if ! curl -s -f "$1" > new.cfg;
    then
        return 1
    fi
    # compare checksums
    if [ "$(md5sum haproxy.cfg | cut -d ' ' -f 1)" = "$(md5sum new.cfg | cut -d ' ' -f 1)" ]
    then 
        return 2
    fi
    # validate
    if ! haproxy -c -f new.cfg;
    then
        return 3
    fi
    # use valid file once all checks are passed
    cp new.cfg haproxy.cfg
    return 0
}

scrape_config()
{
    while :
    do
        # sleep until next iteration
        sleep "$2"
        if fetch_config "$1";
        then
            kill -s SIGUSR2 1
        fi
        # reload worker
    done
}

# make a backup
cp haproxy.cfg previous.cfg

# sleep a bit to wait for manager
sleep "${STARTUP_DELAY:=5}"

# fetch the first config or use the backup
fetch_config "${MANAGER_ENDPOINT:=http://manager:8080}" || true

# run task in background  every minute to update config and restart if needed proxy
scrape_config "$MANAGER_ENDPOINT" "${SCRAPE_INTERVAL:=60}" &

# exec original entrypoint to make it pid 1
exec docker-entrypoint.sh "$@"
