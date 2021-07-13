#!/bin/sh

set -ex

./background.sh &

curl -s -f "$MANAGER_ENDPOINT" > haproxy.cfg  
# compare cache vs new
exec docker-entrypoint.sh haproxy -f /usr/local/etc/haproxy/haproxy.cfg

