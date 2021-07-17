docker run -p 9999:9999 -v $PWD/hapi/haproxy.cfg:/haproxy.cfg --rm --name hp haproxy:alpine -S ipv4@0.0.0.0:9999 -f /haproxy.cfg
