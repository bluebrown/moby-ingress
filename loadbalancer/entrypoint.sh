#!/bin/sh

set -e

fetch_config() 
{
    curl -s -f "$1" > haproxy.cfg
    md5sum haproxy.cfg
}

scrape_config()
{
    while :
        do
            sleep 60
            old_sum="$(md5sum haproxy.cfg)"
            sum="$(fetch_config "$1")"
            if [ "$old_sum" = "$sum" ]
            then
                echo "config has not changed"
            else
                echo "config has changed, reloading worker"
                kill -s SIGUSR2 1
            fi
        done
}

# run task in background to update config and restart proxy
scrape_config "$MANAGER_ENDPOINT" &

# get inital config
fetch_config "$MANAGER_ENDPOINT"

# exec original entrypoint to make it pid 1
exec docker-entrypoint.sh haproxy -f /usr/local/etc/haproxy/haproxy.cfg

