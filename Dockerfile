FROM golang:1.14-alpine3.11 as builder

WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w' -o httelper *.go

FROM alpine:3.11

COPY --from=builder /build/httelper /usr/bin
ENV GIN_MODE=release
ENTRYPOINT [ "/usr/bin/httelper" ]
