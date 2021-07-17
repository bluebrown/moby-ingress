FROM golang as builder

WORKDIR /src/
COPY main.go .
COPY go.mod .
COPY go.sum .
RUN go build -ldflags '-linkmode external -w -extldflags "-static"' -o conf-listener .

FROM haproxy:alpine
USER root
WORKDIR /usr/local/etc/haproxy/
COPY --from=builder /src/conf-listener /usr/local/bin/
COPY *.cfg /usr/local/etc/haproxy/
CMD ["haproxy", "-f", "haproxy.cfg", "-f", "postload.cfg", "-S", "ipv4@0.0.0.0:9080"]
