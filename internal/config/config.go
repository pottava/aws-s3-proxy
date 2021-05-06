package config

import (
	"time"
)

// Config represents its configurations
var (
	Config *config
)

func init() {
	Setup()
}

type config struct { // nolint
	AccessLog          bool          // ACCESS_LOG
	AllPagesInDir      bool          // GET_ALL_PAGES_IN_DIR
	AwsAPIEndpoint     string        // AWS_API_ENDPOINT
	AwsRegion          string        // AWS_REGION
	BasicAuthPass      string        // BASIC_AUTH_PASS
	BasicAuthUser      string        // BASIC_AUTH_USER
	ContentEncoding    bool          // CONTENT_ENCODING
	CorsAllowHeaders   string        // CORS_ALLOW_HEADERS
	CorsAllowMethods   string        // CORS_ALLOW_METHODS
	CorsAllowOrigin    string        // CORS_ALLOW_ORIGIN
	CorsMaxAge         int64         // CORS_MAX_AGE
	DirectoryListing   bool          // DIRECTORY_LISTINGS
	DirListingFormat   string        // DIRECTORY_LISTINGS_FORMAT
	DisableCompression bool          // DISABLE_COMPRESSION
	DisableUpsteamSSL  bool          // Disables SSL in the aws-sdk
	GuessBucketTimeout time.Duration // Used by region helper
	HealthCheckPath    string        // HEALTHCHECK_PATH
	Host               string        // APP_HOST
	HTTPCacheControl   string        // HTTP_CACHE_CONTROL (max-age=86400, no-cache ...)
	HTTPExpires        string        // HTTP_EXPIRES (Thu, 01 Dec 1994 16:00:00 GMT ...)
	IdleConnTimeout    time.Duration // IDLE_CONNECTION_TIMEOUT
	IndexDocument      string        // INDEX_DOCUMENT
	InsecureTLS        bool          // Disables TLS validation on request endpoints.
	JwtSecretKey       string        // JWT_SECRET_KEY
	MaxIdleConns       int           // MAX_IDLE_CONNECTIONS
	Port               string        // APP_PORT
	S3Bucket           string        // AWS_S3_BUCKET
	S3KeyPrefix        string        // AWS_S3_KEY_PREFIX
	SslCert            string        // SSL_CERT_PATH
	SslKey             string        // SSL_KEY_PATH
	StripPath          string        // STRIP_PATH
}

// Setup configurations with environment variables
func Setup() {
	Config = &config{}
}
