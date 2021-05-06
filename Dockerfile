FROM golang:1.16-alpine AS builder
WORKDIR /root

RUN apk --no-cache add gcc musl-dev git
COPY . /root

ENV APP_VERSION=v2.0.0
RUN ls -h && go mod vendor \
    && CGO_ENABLED=0 GOARCH=amd64 go build \
    -o app cmd/aws-s3-proxy/main.go

FROM alpine:3 AS certs
RUN apk add -U ca-certificates

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /root/app /aws-s3-proxy
ENTRYPOINT ["/aws-s3-proxy"]
