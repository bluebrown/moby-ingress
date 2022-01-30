FROM golang as builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . /src/
RUN mkdir -p /dist/templates
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /dist/run .
RUN mv assets/haproxy.cfg.template /dist/templates/

FROM busybox
WORKDIR /mananger
COPY --from=builder /dist ./
ENTRYPOINT ["./run"]
LABEL ingress.role="manager"
