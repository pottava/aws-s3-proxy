# Reverse proxy for AWS S3 w/ basic authentication

## Description

This is a reverse proxy for AWS S3, which is able to provide basic authentication as well.  
You don't need to configure a Bucket for `Website Hosting`.

http://this-proxy.com/access/ -> s3://bucket/access/index.html

## Usage

### Set environment variables

Environment Variables     | Description                                       | Required | Default
------------------------- | ------------------------------------------------- | -------- | -----------------
AWS_REGION                | The AWS `region` where the S3 bucket exists.      |          | us-east-1
AWS_ACCESS_KEY_ID         | AWS `access key` for API access.                  |          | EC2 Instance Role
AWS_SECRET_ACCESS_KEY     | AWS `secret key` for API access.                  |          | EC2 Instance Role
AWS_API_ENDPOINT          | The endpoint for AWS API for local development.   |          | -
S3_PROXY_BASIC_AUTH_PASS  | Password for basic authentication.                |          | -

### Set CLI options

```bash
$ aws-s3-proxy serve -h
serve the s3 proxy

Usage:
  aws-s3-proxy serve [flags]

Flags:
      --access-log                         toggle access log
      --aws-api-endpoint string            AWS API Endpoint
      --aws-region string                  AWS region for s3, default AWS env vars will override (default "us-east-1")
      --basic-auth-user string             username for basic auth
      --config string                      config file (default is $HOME/.aws-s3-proxy.yaml)
      --content-access                     toggle content encoding (default true)
      --cors-allow-headers string          CORS:Comma-delimited list of the supported request headers
      --cors-allow-methods string          CORS: comma-delimited list of the allowed - https://www.w3.org/Protocols/rfc2616/rfc2616-sec9.html
      --cors-allow-origin string           CORS: a URI that may access the resource
      --cors-max-age int                   cors max age in seconds (default 600)
      --directory-listing                  toggle directory listing
      --directory-listing-format           toggle directory listing spider formatted
      --disable-compression                toggle compression (default true)
      --disable-upstream-ssl               toggle tls for the aws-sdk
      --get-all-pages-in-dir               toggle getting all pages in directories
      --guess-bucket-timeout int           timeout, in seconds, for guessing bucket region (default 10)
      --healthcheck-path string            path for healthcheck
  -h, --help                               help for serve
      --http-cache-control Cache-Control   overrides S3's HTTP Cache-Control header
      --http-expires Expires               overrides S3's HTTP Expires header
      --idle-connection-timeout int        idle connection timeout in seconds (default 10)
      --index-document string              the index document for static website (default "index.html")
      --insecure-tls                       toggle insecure tls
      --list-port string                   port to listen on (default "21080")
      --listen-address string              host address to listen on (default "::1")
      --max-idle-connections int           max idle connections (default 150)
      --ssl-cert-path string               path to ssl cert
      --ssl-key-path string                path to ssl key
      --strip-path string                  strip path prefix
      --upstream-bucket string             upstream s3 bucket
      --upstream-key-prefix string         upstream s3 path/key prefix
```

## Copyright and license

Code released under the [MIT license](https://github.com/packethost/aws-s3-proxy/blob/master/LICENSE).
