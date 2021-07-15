#!/bin/sh

set -e


scrape_config()
{
    # take the inital checksum
    sum="$(md5sum haproxy.cfg)"

    while :
        do
            # sleep until next iteration
            sleep "$2"

            # try to fetch the config
            if ! curl -s -f "$1" > haproxy.cfg;
            then
                cp previous.cfg haproxy.cfg
                continue
            fi

            # compare check sums
            old_sum="$sum"
            sum="$(md5sum haproxy.cfg)"
            if [ "$old_sum" = "$sum" ]
            then 
                continue
            fi

            # test if file is valid
            if ! haproxy -c -f haproxy.cfg;
            then
                cp previous.cfg haproxy.cfg
                sum="$old_sum"
                continue
            fi

            # if file has been successfully fetched, 
            # has a different checksum and is valid,
            # reload the worker
            kill -s SIGUSR2 1
        done
}

# make a backup
cp haproxy.cfg previous.cfg

# sleep a bit to wait for manager
sleep "${STARTUP_DELAY:=5}"

# fetch the first config or use the backup
if curl -s -f "${MANAGER_ENDPOINT:=http://manager:8080}" > haproxy.cfg;
then
    if ! haproxy -c -f haproxy.cfg;
        then
            cp previous.cfg haproxy.cfg
fi
else
    cp previous.cfg haproxy.cfg
fi



# run task in background  every minute to update config and restart if needed proxy
scrape_config "$MANAGER_ENDPOINT" "${SCRAPE_INTERVAL:=60}" &

# exec original entrypoint to make it pid 1
exec haproxy -W -db -f haproxy.cfg

