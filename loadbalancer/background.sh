#!/bin/sh

set -ex

while :
do
    sleep 60
    curl -s -f "$MANAGER_ENDPOINT" > haproxy.cfg 
    kill -s SIGUSR2 1
done