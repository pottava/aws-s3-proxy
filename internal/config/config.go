package config

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Cfg represents its configurations
var Cfg *Config

// Logger exports a logger to be used by all parts of the application
var Logger *zap.SugaredLogger

// ReadThrough holds info if we are transparently reading back to upstream
type ReadThrough struct {
	Enabled        bool
	CacheToPrimary bool
}

// Bucket has the attributes needed to interact with S3 buckets
type Bucket struct {
	AccessKey string
	// Endpoint        string
	IdleConnTimeout time.Duration
	Bucket          string
	Region          string
	S3Prefix        string
	SecretKey       string

	Session *session.Session

	InsecureTLS        bool
	DisableCompression bool
	DisableBucketSSL   bool

	MaxIdleConns int
}

// HTTPOpts has http options
type HTTPOpts struct {
	Facility         string
	HealthCheckPath  string
	HTTPCacheControl string
	HTTPExpires      string

	ContentEncoding bool
	EnableUpload    bool
}

// ServerOpts has configs for how to bind
type ServerOpts struct {
	ListenAddress string
	ListenPort    string
}

// Config encapsulates other config options
type Config struct {
	HTTPOpts       HTTPOpts
	ServerOpts     ServerOpts
	Logger         zap.SugaredLogger
	SecondaryStore Bucket
	PrimaryStore   Bucket
	ReadThrough    ReadThrough
}

// Load configurations and map to the config struct
func Load(ctx context.Context, l *zap.SugaredLogger) {
	if err := viper.Unmarshal(&Cfg); err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}

	Logger = l

	Cfg.PrimaryStore.BuildS3API()
	Cfg.SecondaryStore.BuildS3API()

	Logger.Info("configuration loaded")
}

// BuildS3API creates a client per bucket
func (b *Bucket) BuildS3API() {
	b.Session = session.Must(session.NewSession(b.buildAwsConfig()))
}

func (b *Bucket) buildAwsConfig() *aws.Config {
	awsCfg := &aws.Config{
		Region:      &b.Region,
		Credentials: credentials.NewStaticCredentials(b.AccessKey, b.SecretKey, ""),
		DisableSSL:  aws.Bool(b.DisableBucketSSL),
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				Proxy:              http.ProxyFromEnvironment,
				MaxIdleConns:       b.MaxIdleConns,
				IdleConnTimeout:    b.IdleConnTimeout,
				DisableCompression: b.DisableCompression,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: b.InsecureTLS,
				},
			},
		},
	}

	if b.Region == "" {
		Cfg.Logger.Panicf("failed to guess region for %s bucket, please specify", b.Bucket)
	}

	// if b.Endpoint != "" {
	// 	awsCfg.Endpoint = &b.Endpoint
	// 	awsCfg.S3ForcePathStyle = aws.Bool(true)
	// }

	return awsCfg
}
