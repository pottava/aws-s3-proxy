FROM golang:1.14-alpine AS builder
RUN apk --no-cache --update upgrade && \
    apk --no-cache add gcc musl-dev git make upx bash ca-certificates && \
    mkdir /tmp/tmp && chmod 1777 /tmp/tmp
COPY . /build

RUN cd /build && make build

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /tmp/tmp /tmp
COPY --from=builder /build/artifacts/svc /svc
COPY --from=builder /build/sha /
COPY --from=builder /build/version /

EXPOSE 8080 8888

CMD ["./svc"]
