FROM alpine:3.11 as base

RUN apk --no-cache --update upgrade && apk --no-cache add ca-certificates && \
 mkdir /tmp/tmp && chmod 1777 /tmp/tmp

FROM scratch
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=base /tmp/tmp /tmp
COPY ./artifacts/svc /svc
COPY ./sha /
COPY ./version /

EXPOSE 8080 8888

CMD ["./svc"]
