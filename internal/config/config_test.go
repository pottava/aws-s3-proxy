package config

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func defaultConfig() *config {
	return &config{
		AwsRegion:          "",
		AwsAPIEndpoint:     "",
		S3Bucket:           "",
		S3KeyPrefix:        "",
		IndexDocument:      "index.html",
		DirectoryListing:   false,
		DirListingFormat:   "",
		HTTPCacheControl:   "",
		HTTPExpires:        "",
		BasicAuthUser:      "",
		BasicAuthPass:      "",
		Port:               "80",
		Host:               "",
		AccessLog:          false,
		SslCert:            "",
		SslKey:             "",
		StripPath:          "",
		ContentEncoding:    true,
		CorsAllowOrigin:    "",
		CorsAllowMethods:   "",
		CorsAllowHeaders:   "",
		CorsMaxAge:         int64(600),
		HealthCheckPath:    "",
		AllPagesInDir:      false,
		MaxIdleConns:       150,
		IdleConnTimeout:    time.Duration(10) * time.Second,
		DisableCompression: true,
		InsecureTLS:        false,
		WhiteListIPRanges:  make([]*net.IPNet, 0),
	}
}

func TestConfigDefaults(t *testing.T) {
	expected := defaultConfig()
	assert.Equal(t, expected, Config)
}

func TestChangeDefaults(t *testing.T) {
	os.Setenv("DIRECTORY_LISTINGS", "1")
	os.Setenv("ACCESS_LOG", "True")
	os.Setenv("CONTENT_ENCODING", "f")
	os.Setenv("CORS_MAX_AGE", "0")
	os.Setenv("GET_ALL_PAGES_IN_DIR", "TRUE")
	os.Setenv("MAX_IDLE_CONNECTIONS", "0")
	os.Setenv("IDLE_CONNECTION_TIMEOUT", "60")
	os.Setenv("DISABLE_COMPRESSION", "FALSE")
	os.Setenv("INSECURE_TLS", "t")
	os.Setenv("WHITELIST_IP_RANGES", "10.0.0.0/24,198.5.5.3")

	Setup()

	expected := defaultConfig()
	expected.DirectoryListing = true
	expected.AccessLog = true
	expected.ContentEncoding = false
	expected.CorsMaxAge = 0
	expected.AllPagesInDir = true
	expected.MaxIdleConns = 0
	expected.IdleConnTimeout = time.Duration(60) * time.Second
	expected.DisableCompression = false
	expected.InsecureTLS = true
	expected.WhiteListIPRanges = make([]*net.IPNet, 0, 2)
	for _, subStr := range []string{"10.0.0.0/24", "198.5.5.3/32"} {
		_, subnet, _ := net.ParseCIDR(subStr)
		expected.WhiteListIPRanges = append(expected.WhiteListIPRanges, subnet)
	}

	assert.Equal(t, expected, Config)
}
