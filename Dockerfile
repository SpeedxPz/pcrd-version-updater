
FROM golang:1.18 AS builder
WORKDIR $GOPATH/src/app/
COPY . .

ENV GO111MODULE=on
RUN go mod download
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o /go/bin/app

FROM alpine:3.15
EXPOSE 80
WORKDIR /go/bin/
COPY --from=builder /go/bin/app /go/bin/app

ENTRYPOINT ["./app"]
