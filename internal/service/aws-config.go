package service

import (
	"crypto/tls"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/packethost/aws-s3-proxy/internal/config"
)

func awsSession(region *string) *session.Session {
	cfg := &aws.Config{
		DisableSSL: aws.Bool(config.Config.DisableUpstreamSSL),
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				Proxy:              http.ProxyFromEnvironment,
				MaxIdleConns:       config.Config.MaxIdleConns,
				IdleConnTimeout:    config.Config.IdleConnTimeout,
				DisableCompression: config.Config.DisableCompression,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: config.Config.InsecureTLS,
				},
			},
		},
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
