#!/bin/sh

set -e

# make a backup
cp haproxy.cfg previous.cfg

# sleep a bit to wait for manager
sleep "${STARTUP_DELAY:=5}"

# fetch the first config or use the backup
if curl -s -f "${MANAGER_ENDPOINT:=http://manager:8080}" > haproxy.cfg;
then
   haproxy -c -f haproxy.cfg || cp previous.cfg haproxy.cfg
else
    cp previous.cfg haproxy.cfg
fi

# exec original entrypoint to make it pid 1
exec docker-entrypoint.sh "$@"
