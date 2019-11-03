# AWS S3 Proxy v2.0
# docker run -d -p 8080:80 -e AWS_REGION -e AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY -e AWS_S3_BUCKET pottava/s3-proxy

FROM golang:1.13.3-alpine3.10 AS build-env
RUN apk --no-cache add gcc musl-dev git
RUN go get -u github.com/pottava/aws-s3-proxy
WORKDIR /go/src/github.com/pottava/aws-s3-proxy
RUN git checkout v2.0.0-rc1 > /dev/null 2>&1
RUN go mod vendor
RUN go build -a -installsuffix cgo -ldflags "-s -w" -o /app

FROM alpine:3.10

ENV AWS_REGION=us-east-1 \
    APP_PORT=80 \
    ACCESS_LOG=false \
    CONTENT_ENCODING=true

RUN apk add --no-cache ca-certificates
COPY --from=build-env /app /aws-s3-proxy
ENTRYPOINT ["/aws-s3-proxy"]
