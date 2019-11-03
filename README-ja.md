# Basic 認証も可能な AWS S3 へのリバースプロキシ

[![pottava/s3-proxy](http://dockeri.co/image/pottava/s3-proxy)](https://hub.docker.com/r/pottava/s3-proxy/)


## 概要

指定した S3 バケット にリバースプロキシする Web サービスです。  
API 経由でアクセスするため、バケットに静的 Web サイトホスティングの設定は不要です。  
オプションでフロントに Basic 認証がかけられます。

http://this-proxy.com/access/ -> s3://bucket/access/index.html


## 使い方

### 1. 環境変数をセットします

環境変数                   | 説明                                             | 必須    | 初期値
------------------------- | ----------------------------------------------- | ------ | ---
AWS_S3_BUCKET             | プロキシ先の S3 バケット                           | *       |
AWS_S3_KEY_PREFIX         | S3 オブジェクトにプリフィクス文字列があるなら指定       |        | -
AWS_REGION                | バケットの存在する AWS リージョン                    |        | us-east-1
AWS_ACCESS_KEY_ID         | API を使うための AWS アクセスキー                   |        | EC2 インスタンスロール
AWS_SECRET_ACCESS_KEY     | API を使うための AWS シークレットキー                |        | EC2 インスタンスロール
AWS_API_ENDPOINT          | API 接続先エンドポイント（通常指定する必要なし）       |          | -
INDEX_DOCUMENT            | インデックスドキュメントの名前                       |          | index.html
DIRECTORY_LISTINGS        | / で終わる URL の場合、ファイル一覧を返す             |          | false
DIRECTORY_LISTINGS_FORMAT | `html` がセットされていたらファイル一覧を HTML で返す |       | -
HTTP_CACHE_CONTROL        | S3 の `Cache-Control` 属性を上書きして返します      |        | S3 オブジェクト属性値
HTTP_EXPIRES              | S3 の `Expires` 属性を上書きして返します            |        | S3 オブジェクト属性値
BASIC_AUTH_USER           | Basic 認証をかけるなら、その `ユーザ名`              |        | -
BASIC_AUTH_PASS           | Basic 認証をかけるなら、その `パスワード`            |        | -
SSL_CERT_PATH             | TLS を有効にしたいなら、その `cert.pem` へのパス     |        | -
SSL_KEY_PATH              | TLS を有効にしたいなら、その `key.pem` へのパス      |        | -
CORS_ALLOW_ORIGIN  | CORS を有効にしたいなら、リソースへのアクセスを許可する URI |        | -
CORS_ALLOW_METHODS | CORS を有効にしたいなら、許可する [HTTP request methods](https://www.w3.org/Protocols/rfc2616/rfc2616-sec9.html)のカンマ区切りのリスト |        | -
CORS_ALLOW_HEADERS | CORS を有効にしたいなら、サポートするヘッダーのカンマ区切りのリスト |        | -
CORS_MAX_AGE       | CORS における preflight リクエスト結果のキャッシュ上限時間(秒) |        | 600
APP_PORT                  | このサービスが待機する `ポート番号`                  |        | 80
ACCESS_LOG                | 標準出力へアクセスログを送る                        |        | false
STRIP_PATH                | 指定した Prefix を S3 のパスから削除                |         | -
CONTENT_ENCODING          | リクエストが許可していればレスポンスを圧縮します       |        | true
HEALTHCHECK_PATH          | 指定すると Basic 認証設定の有無などに依らず 200 OK を返します |   | -
GET_ALL_PAGES_IN_DIR      | 指定ディレクトリの全てのオブジェクトを返す             |          | false
MAX_IDLE_CONNECTIONS      | S3 への利用が終わったコネクションの最大保持数          |       | 150
IDLE_CONNECTION_TIMEOUT   | S3 への接続タイムアウト                            |          | 10
DISABLE_COMPRESSION       | S3 との間の Content-Encoding を無効にします         |          | true
INSECURE_TLS              | TLS 証明書の正当性チェックをスキップします             |          | false

### 2. アプリを起動します

`docker run -d -p 8080:80 -e AWS_REGION -e AWS_S3_BUCKET pottava/s3-proxy`

* Basic 認証をつけるなら:

`docker run -d -p 8080:80 -e AWS_REGION -e AWS_S3_BUCKET -e BASIC_AUTH_USER -e BASIC_AUTH_PASS pottava/s3-proxy`

* TLS を有効にしたいなら:

`docker run -d -p 8080:80 -e AWS_REGION -e AWS_S3_BUCKET -e SSL_CERT_PATH -e SSL_KEY_PATH pottava/s3-proxy`

* CORS を有効にしたいなら:

`docker run -d -p 8080:80 -e CORS_ALLOW_ORIGIN -e CORS_ALLOW_METHODS -e CORS_ALLOW_HEADERS -e CORS_MAX_AGE pottava/s3-proxy`

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
