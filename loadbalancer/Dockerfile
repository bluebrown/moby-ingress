FROM haproxy:alpine

USER root
RUN apk add --update curl socat && rm -rf /var/cache/apk/*

WORKDIR /usr/local/etc/haproxy/

ENTRYPOINT [ "entrypoint.sh" ]
CMD ["haproxy", "-f", "/usr/local/etc/haproxy/program.cfg",  "-f", "/usr/local/etc/haproxy/haproxy.cfg", "-S", "/run/haproxy-master.sock"]

COPY *.cfg /usr/local/etc/haproxy/ 
COPY *.sh /usr/local/bin/