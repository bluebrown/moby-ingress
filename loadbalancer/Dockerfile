FROM haproxy:alpine

USER root
RUN apk add --update curl socat && rm -rf /var/cache/apk/*

WORKDIR /usr/local/etc/haproxy/
COPY . . 

ENTRYPOINT [ "./entrypoint.sh" ]
CMD ["haproxy", "-f", "/usr/local/etc/haproxy/haproxy.cfg", "-S", "/run/haproxy-master.sock"]
