FROM golang as builder
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . /workspace/
RUN mkdir -p /dist
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /dist/standalone ./prog/standalone
RUN mv assets /dist/assets

FROM haproxy:alpine
USER root
WORKDIR /standalone
COPY --from=builder /dist ./
ENTRYPOINT ["./standalone"]
LABEL ingress.role="controller"
