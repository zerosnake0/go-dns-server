FROM golang:1.14.4-alpine3.12

WORKDIR /go/src/github.com/zerosnake0/go-dns-server

COPY . .

RUN go build -o /app/dns main.go

EXPOSE 53

ENTRYPOINT ["/app/dns"]