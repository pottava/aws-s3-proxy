# Basic 認証も可能な AWS S3 へのリバースプロキシ


## 概要

指定した S3 バケット にリバースプロキシする Web サービスです。  
API 経由でアクセスするため、バケットに静的 Web サイトホスティングの設定は不要です。  
オプションでフロントに Basic 認証がかけられます。


## 使い方

### 1. 環境変数をセットします

環境変数                   | 説明                                             | 必須
------------------------- | ----------------------------------------------- | ---------
AWS_S3_BUCKET             | プロキシ先の S3 バケット                           | *
AWS_REGION                | バケットの存在する AWS リージョン                    | *
AWS_ACCESS_KEY_ID         | API を使うための AWS アクセスキー                   | インスタンスロールでも OK
AWS_SECRET_ACCESS_KEY     | API を使うための AWS シークレットキー                | 
BASIC_AUTH_PASS           | Basic 認証をかけるなら、その `パスワード`            | 
APP_PORT                  | このサービスが待機する `ポート番号` （デフォルト 80番） | 
SSL_CERT_PATH             | TLS を有効にしたいなら、その `cert.pem` へのパス     | 
SSL_KEY_PATH              | TLS を有効にしたいなら、その `key.pem` へのパス      | 
ACCESS_LOG                | 標準出力へアクセスログを送る (初期値: false)          | 

### 2. アプリを起動します

`docker run -d -p 8080:80 -e AWS_REGION -e AWS_S3_BUCKET pottava/s3-proxy`

* Basic 認証をつけるなら:  

`docker run -d -p 8080:80 -e AWS_REGION -e AWS_S3_BUCKET -e BASIC_AUTH_USER -e BASIC_AUTH_PASS pottava/s3-proxy`

* TLS を有効にしたいなら:  

`docker run -d -p 8080:80 -e AWS_REGION -e AWS_S3_BUCKET -e SSL_CERT_PATH -e SSL_KEY_PATH pottava/s3-proxy`

* docker-compose.yml として使うなら:  

```
proxy:
  image: pottava/s3-proxy
  ports:
    - 8080:80
  environment:
    - AWS_REGION=ap-northeast-1
    - AWS_ACCESS_KEY_ID
    - AWS_SECRET_ACCESS_KEY
    - AWS_S3_BUCKET
    - BASIC_AUTH_USER=admin
    - BASIC_AUTH_PASS=password
    - ACCESS_LOG=true
  container_name: proxy
```
