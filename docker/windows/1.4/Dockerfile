FROM mcr.microsoft.com/windows/nanoserver:10.0.14393.2485

ENV AWS_REGION=us-east-1 \
    APP_PORT=80 \
    ACCESS_LOG=false \
    CONTENT_ENCODING=true

ADD https://github.com/pottava/aws-s3-proxy/releases/download/v1.4.1/windows_amd64.exe proxy.exe
ENTRYPOINT ["C:\\proxy.exe"]
