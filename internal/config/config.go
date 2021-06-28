package config

import (
	"log"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Config represents its configurations
var Config *config

type config struct { // nolint
	AccessLog          bool               // ACCESS_LOG
	AllPagesInDir      bool               // GET_ALL_PAGES_IN_DIR
	AwsAPIEndpoint     string             // AWS_API_ENDPOINT
	AwsRegion          string             // AWS_REGION
	BasicAuthPass      string             // BASIC_AUTH_PASS
	BasicAuthUser      string             // BASIC_AUTH_USER
	ContentEncoding    bool               // CONTENT_ENCODING
	CorsAllowHeaders   string             // CORS_ALLOW_HEADERS
	CorsAllowMethods   string             // CORS_ALLOW_METHODS
	CorsAllowOrigin    string             // CORS_ALLOW_ORIGIN
	CorsMaxAge         int64              // CORS_MAX_AGE
	DirectoryListing   bool               // DIRECTORY_LISTINGS
	DirListingFormat   string             // DIRECTORY_LISTINGS_FORMAT
	DisableCompression bool               // DISABLE_COMPRESSION
	DisableUpstreamSSL bool               // Disables SSL in the aws-sdk
	EnableUpload       bool               // Toggles upload
	Facility           string             // Location the service is running in
	GuessBucketTimeout time.Duration      // Used by region helper
	HealthCheckPath    string             // HEALTHCHECK_PATH
	HTTPCacheControl   string             // HTTP_CACHE_CONTROL (max-age=86400, no-cache ...)
	HTTPExpires        string             // HTTP_EXPIRES (Thu, 01 Dec 1994 16:00:00 GMT ...)
	IdleConnTimeout    time.Duration      // IDLE_CONNECTION_TIMEOUT
	IndexDocument      string             // INDEX_DOCUMENT
	InsecureTLS        bool               // Disables TLS validation on request endpoints.
	JwtSecretKey       string             // JWT_SECRET_KEY
	ListenAddress      string             //
	ListenPort         string             // APP_PORT
	Logger             *zap.SugaredLogger // A logger
	MaxIdleConns       int                // MAX_IDLE_CONNECTIONS
	S3Bucket           string             // AWS_S3_BUCKET
	S3KeyPrefix        string             // AWS_S3_KEY_PREFIX
	SslCert            string             // SSL_CERT_PATH
	SslKey             string             // SSL_KEY_PATH
	StripPath          string             // STRIP_PATH
}

// Load configurations and map to the config struct
func Load(l *zap.SugaredLogger) {
	if err := viper.Unmarshal(&Config); err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}

	Config.Logger = l
}
