FROM alpine:3.5

LABEL com.github.pottava.application="aws-s3-proxy" \
      com.github.pottava.usage="docker run -d -p 8080:80 -e AWS_REGION -e AWS_S3_BUCKET -e BASIC_AUTH_USER -e BASIC_AUTH_PASS pottava/s3-proxy" \
      com.github.pottava.license="MIT"

ENV GOPATH=/go PATH=/go/bin:$PATH GODEBUG=netdns=go APP_PORT=80

RUN apk add --no-cache ca-certificates \
      && apk --no-cache add --virtual build-dependencies go git \
      && mkdir -p /go/src /go/bin \
      && chmod -R 777 /go \
      && go get -u github.com/pottava/aws-s3-proxy \
      && apk del --purge build-dependencies \
      && rm -rf /go/pkg /go/src

EXPOSE 80

ENTRYPOINT ["/go/bin/aws-s3-proxy"]
