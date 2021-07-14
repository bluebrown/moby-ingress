FROM golang
WORKDIR /src
COPY go.mod .
COPY go.sum .
COPY main.go .
RUN ls
RUN go build -o app .
ENTRYPOINT [ "./app" ] 
COPY haproxy.cfg.template .