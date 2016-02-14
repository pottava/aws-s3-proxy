# Reverse proxy for AWS S3 w/ basic authentication

![circleci status](https://circleci.com/gh/pottava/aws-s3-proxy.svg?style=shield&circle-token=)

## Description

This is a reverse proxy for AWS S3, which is able to provide basic authentication as well.  
You don't neet to configure a Bucket for `Website Hosting`.  
([日本語はこちら](https://github.com/pottava/aws-s3-proxy/blob/master/README-ja.md))


## Usage

### 1. Set environment variables

Environment Variables     | Description                                       | Required
------------------------- | ------------------------------------------------- | ---------
AWS_S3_BUCKET             | The `S3 bucket` to be proxied with this app.      | *
AWS_REGION                | The AWS `region` where the S3 bucket exists.      | *
AWS_ACCESS_KEY_ID         | AWS `access key` for API access.                  | or EC2 Instance Role
AWS_SECRET_ACCESS_KEY     | AWS `secret key` for API access.                  | 
BASIC_AUTH_USER           | User for basic authentication.                    | 
BASIC_AUTH_PASS           | Password for basic authentication.                | 
APP_PORT                  | The port number to be assigned for listening.     | 
SSL_CERT_PATH             | TLS: cert.pem file path.                          | 
SSL_KEY_PATH              | TLS: key.pem file path.                           | 
ACCESS_LOG                | Send access logs to /dev/stdout. (default: false) | 

### 2. Run the application

`docker run -d -p 8080:80 -e AWS_REGION -e AWS_S3_BUCKET pottava/s3-proxy`

* with basic auth:  

`docker run -d -p 8080:80 -e AWS_REGION -e AWS_S3_BUCKET -e BASIC_AUTH_USER -e BASIC_AUTH_PASS pottava/s3-proxy`

* with TLS:  

`docker run -d -p 8080:80 -e AWS_REGION -e AWS_S3_BUCKET -e SSL_CERT_PATH -e SSL_KEY_PATH pottava/s3-proxy`

* with docker-compose.yml:  

```
proxy:
  image: pottava/s3-proxy
  ports:
    - 8080:80
  links:
    - web
  environment:
    - AWS_REGION=ap-northeast-1
    - AWS_ACCESS_KEY_ID
    - AWS_SECRET_ACCESS_KEY
    - AWS_S3_BUCKET
    - BASIC_AUTH_PASS=password
    - ACCESS_LOG=true
  container_name: proxy
```


## Copyright and license

Code released under the [MIT license](https://github.com/pottava/aws-s3-proxy/blob/master/LICENSE).
