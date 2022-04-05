FROM golang:1.17.8-alpine3.15 as builder
RUN apk add --no-cache \
    xz-dev \
    musl-dev \
    gcc \
    ca-certificates
WORKDIR /go/src/github.com/mendersoftware/workflows
COPY ./ .
RUN env CGO_ENABLED=0 go build

FROM scratch
EXPOSE 8080
WORKDIR /etc/workflows
COPY ./config.yaml .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/github.com/mendersoftware/workflows/workflows /usr/bin/

ENTRYPOINT ["/usr/bin/workflows", "--config", "/etc/workflows/config.yaml"]

EXPOSE 8080
