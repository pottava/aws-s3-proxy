package config

import (
	"log"
	"os"
	"strconv"
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
	AwsRegion          string        // AWS_REGION
	AwsAPIEndpoint     string        // AWS_API_ENDPOINT
	S3Bucket           string        // AWS_S3_BUCKET
	S3KeyPrefix        string        // AWS_S3_KEY_PREFIX
	IndexDocument      string        // INDEX_DOCUMENT
	DirectoryListing   bool          // DIRECTORY_LISTINGS
	DirListingFormat   string        // DIRECTORY_LISTINGS_FORMAT
	HTTPCacheControl   string        // HTTP_CACHE_CONTROL (max-age=86400, no-cache ...)
	HTTPExpires        string        // HTTP_EXPIRES (Thu, 01 Dec 1994 16:00:00 GMT ...)
	BasicAuthUser      string        // BASIC_AUTH_USER
	BasicAuthPass      string        // BASIC_AUTH_PASS
	Port               string        // APP_PORT
	Host               string        // APP_HOST
	AccessLog          bool          // ACCESS_LOG
	SslCert            string        // SSL_CERT_PATH
	SslKey             string        // SSL_KEY_PATH
	StripPath          string        // STRIP_PATH
	ContentEncoding    bool          // CONTENT_ENCODING
	CorsAllowOrigin    string        // CORS_ALLOW_ORIGIN
	CorsAllowMethods   string        // CORS_ALLOW_METHODS
	CorsAllowHeaders   string        // CORS_ALLOW_HEADERS
	CorsMaxAge         int64         // CORS_MAX_AGE
	HealthCheckPath    string        // HEALTHCHECK_PATH
	AllPagesInDir      bool          // GET_ALL_PAGES_IN_DIR
	MaxIdleConns       int           // MAX_IDLE_CONNECTIONS
	IdleConnTimeout    time.Duration // IDLE_CONNECTION_TIMEOUT
	DisableCompression bool          // DISABLE_COMPRESSION
	InsecureTLS        bool          // Disables TLS validation on request endpoints.
	JwtSecretKey       string        // JWT_SECRET_KEY
	ReverseSorting     bool          // REVERSE_SORTING
}

// Setup configurations with environment variables
func Setup() {
	region := os.Getenv("AWS_REGION")
	if len(region) == 0 {
		region = os.Getenv("AWS_DEFAULT_REGION")
	}
	port := os.Getenv("APP_PORT")
	if len(port) == 0 {
		port = "80"
	}
	indexDocument := os.Getenv("INDEX_DOCUMENT")
	if len(indexDocument) == 0 {
		indexDocument = "index.html"
	}
	directoryListings := false
	if b, err := strconv.ParseBool(os.Getenv("DIRECTORY_LISTINGS")); err == nil {
		directoryListings = b
	}
	accessLog := false
	if b, err := strconv.ParseBool(os.Getenv("ACCESS_LOG")); err == nil {
		accessLog = b
	}
	contentEncoding := true
	if b, err := strconv.ParseBool(os.Getenv("CONTENT_ENCODING")); err == nil {
		contentEncoding = b
	}
	corsMaxAge := int64(600)
	if i, err := strconv.ParseInt(os.Getenv("CORS_MAX_AGE"), 10, 64); err == nil {
		corsMaxAge = i
	}
	allPagesInDir := false
	if b, err := strconv.ParseBool(os.Getenv("GET_ALL_PAGES_IN_DIR")); err == nil {
		allPagesInDir = b
	}
	maxIdleConns := 150
	if b, err := strconv.ParseInt(os.Getenv("MAX_IDLE_CONNECTIONS"), 10, 16); err == nil {
		maxIdleConns = int(b)
	}
	idleConnTimeout := time.Duration(10) * time.Second
	if b, err := strconv.ParseInt(os.Getenv("IDLE_CONNECTION_TIMEOUT"), 10, 64); err == nil {
		idleConnTimeout = time.Duration(b) * time.Second
	}
	disableCompression := true
	if b, err := strconv.ParseBool(os.Getenv("DISABLE_COMPRESSION")); err == nil {
		disableCompression = b
	}
	insecureTLS := false
	if b, err := strconv.ParseBool(os.Getenv("INSECURE_TLS")); err == nil {
		insecureTLS = b
	}
	reverseSorting := false
	if b, err := strconv.ParseBool(os.Getenv("REVERSE_SORTING")); err == nil {
		reverseSorting = b
	}
	Config = &config{
		AwsRegion:          region,
		AwsAPIEndpoint:     os.Getenv("AWS_API_ENDPOINT"),
		S3Bucket:           os.Getenv("AWS_S3_BUCKET"),
		S3KeyPrefix:        os.Getenv("AWS_S3_KEY_PREFIX"),
		IndexDocument:      indexDocument,
		DirectoryListing:   directoryListings,
		DirListingFormat:   os.Getenv("DIRECTORY_LISTINGS_FORMAT"),
		HTTPCacheControl:   os.Getenv("HTTP_CACHE_CONTROL"),
		HTTPExpires:        os.Getenv("HTTP_EXPIRES"),
		BasicAuthUser:      os.Getenv("BASIC_AUTH_USER"),
		BasicAuthPass:      os.Getenv("BASIC_AUTH_PASS"),
		Port:               port,
		Host:               os.Getenv("APP_HOST"),
		AccessLog:          accessLog,
		SslCert:            os.Getenv("SSL_CERT_PATH"),
		SslKey:             os.Getenv("SSL_KEY_PATH"),
		StripPath:          os.Getenv("STRIP_PATH"),
		ContentEncoding:    contentEncoding,
		CorsAllowOrigin:    os.Getenv("CORS_ALLOW_ORIGIN"),
		CorsAllowMethods:   os.Getenv("CORS_ALLOW_METHODS"),
		CorsAllowHeaders:   os.Getenv("CORS_ALLOW_HEADERS"),
		CorsMaxAge:         corsMaxAge,
		HealthCheckPath:    os.Getenv("HEALTHCHECK_PATH"),
		AllPagesInDir:      allPagesInDir,
		MaxIdleConns:       maxIdleConns,
		IdleConnTimeout:    idleConnTimeout,
		DisableCompression: disableCompression,
		InsecureTLS:        insecureTLS,
		JwtSecretKey:       os.Getenv("JWT_SECRET_KEY"),
		ReverseSorting:     reverseSorting,
	}
	// Proxy
	log.Printf("[config] Proxy to %v", Config.S3Bucket)
	log.Printf("[config] AWS Region: %v", Config.AwsRegion)

	// TLS pem files
	if (len(Config.SslCert) > 0) && (len(Config.SslKey) > 0) {
		log.Print("[config] TLS enabled.")
	}
	// Basic authentication
	if (len(Config.BasicAuthUser) > 0) && (len(Config.BasicAuthPass) > 0) {
		log.Printf("[config] Basic authentication: %s", Config.BasicAuthUser)
	}
	// CORS
	if (len(Config.CorsAllowOrigin) > 0) && (Config.CorsMaxAge > 0) {
		log.Printf("[config] CORS enabled: %s", Config.CorsAllowOrigin)
	}
}
