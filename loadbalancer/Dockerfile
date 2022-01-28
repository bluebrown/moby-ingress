FROM haproxy:2.5-alpine3.15

WORKDIR /usr/local/etc/haproxy/

USER root

RUN apk add --update curl socat && rm -rf /var/cache/apk/* \
  && mkdir -p /var/run/haproxy \
  && chown -R haproxy:haproxy /var/run/haproxy \
  && chown -R haproxy:haproxy /usr/local/etc/haproxy

ENTRYPOINT [ "entrypoint.sh" ]

CMD ["haproxy", "-f", "/usr/local/etc/haproxy/haproxy.cfg", "-S", "/run/haproxy/haproxy-master.sock"]

COPY --chown=haproxy:haproxy *.cfg /usr/local/etc/haproxy/

COPY --chown=haproxy:haproxy *.sh  /usr/local/bin/

USER haproxy
