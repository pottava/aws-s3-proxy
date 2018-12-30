# AWS S3 Proxy v1.4
# docker run -d -p 8080:80 -e AWS_REGION -e AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY -e AWS_S3_BUCKET pottava/s3-proxy

FROM golang:1.11.4-alpine3.8 AS build-env
RUN apk --no-cache add gcc musl-dev git
RUN go get -u github.com/golang/dep/...
WORKDIR /go/src/github.com/golang/dep
RUN git checkout v0.5.0 > /dev/null 2>&1
RUN go install github.com/golang/dep/...
RUN go get -u github.com/pottava/aws-s3-proxy
WORKDIR /go/src/github.com/pottava/aws-s3-proxy
RUN git checkout v1.4.1 > /dev/null 2>&1
RUN dep ensure
RUN go build -a -installsuffix cgo -ldflags "-s -w"

FROM alpine:3.8

ENV AWS_REGION=us-east-1 \
    APP_PORT=80 \
    ACCESS_LOG=false \
    CONTENT_ENCODING=true

RUN apk add --no-cache ca-certificates
COPY --from=build-env /go/src/github.com/pottava/aws-s3-proxy/aws-s3-proxy /aws-s3-proxy
ENTRYPOINT ["/aws-s3-proxy"]
