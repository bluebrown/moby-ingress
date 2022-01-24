FROM haproxy:alpine

USER root
RUN apk add --update curl socat && rm -rf /var/cache/apk/*

ENTRYPOINT [ "entrypoint.sh" ]
CMD ["haproxy", "-f", "/usr/local/etc/haproxy/haproxy.cfg", "-S", "/run/haproxy-master.sock"]

WORKDIR /usr/local/etc/haproxy/

COPY *.cfg /usr/local/etc/haproxy/
COPY *.sh  /usr/local/bin/

# TODO: use a non-root user
