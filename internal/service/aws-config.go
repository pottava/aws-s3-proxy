package service

import (
	"crypto/tls"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pottava/aws-s3-proxy/internal/config"
)

func awsSession(region *string) *session.Session {
	cfg := &aws.Config{
		HTTPClient: configureClient(),
	}
	if region != nil {
		cfg.Region = region
	}
	if len(config.Config.AwsAPIEndpoint) > 0 {
		cfg.Endpoint = aws.String(config.Config.AwsAPIEndpoint)
		cfg.S3ForcePathStyle = aws.Bool(true)
	}
	return session.Must(session.NewSession(cfg))
}

func configureClient() *http.Client {
	tlsCfg := &tls.Config{}
	if config.Config.InsecureTLS {
		tlsCfg.InsecureSkipVerify = true
	}
	transport := &http.Transport{
		Proxy:              http.ProxyFromEnvironment,
		MaxIdleConns:       config.Config.MaxIdleConns,
		IdleConnTimeout:    config.Config.IdleConnTimeout,
		DisableCompression: config.Config.DisableCompression,
		TLSClientConfig:    tlsCfg,
	}
	return &http.Client{
		Transport: transport,
	}
}
