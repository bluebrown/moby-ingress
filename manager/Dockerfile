FROM golang

WORKDIR /src

COPY . . 
RUN go build -o app .
ENTRYPOINT [ "./app" ] 