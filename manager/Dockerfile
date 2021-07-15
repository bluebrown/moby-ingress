FROM golang
WORKDIR /src
COPY go.mod .
COPY go.sum .
COPY *.go /src/
RUN go build -o app .
ENTRYPOINT [ "./app" ] 
COPY haproxy.cfg.template .