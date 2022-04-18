FROM golang:1.18.1-alpine3.15 as builder
RUN apk add --no-cache \
    xz-dev \
    musl-dev \
    gcc
WORKDIR /go/src/github.com/mendersoftware/workflows
COPY ./ .
RUN env CGO_ENABLED=1 go build

FROM alpine:3.15.0
RUN apk add --no-cache ca-certificates xz
COPY --from=builder /go/src/github.com/mendersoftware/workflows/workflows /usr/bin

RUN mkdir -p /etc/workflows
COPY ./config.yaml /etc/workflows
ENTRYPOINT ["/usr/bin/workflows", "--config", "/etc/workflows/config.yaml"]

EXPOSE 8080
