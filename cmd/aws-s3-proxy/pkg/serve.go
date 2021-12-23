package cmd

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/go-openapi/swag"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go.uber.org/automaxprocs/maxprocs"
	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"

	"github.com/packethost/aws-s3-proxy/internal/config"
	"github.com/packethost/aws-s3-proxy/internal/service"
)

var tracer = otel.Tracer("artifacts-store")

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "serve the s3 proxy",
	Run: func(cmd *cobra.Command, args []string) {
		serve(cmd.Context())
	},
}

func newExporter(ctx context.Context) (*otlptrace.Exporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint("api.honeycomb.io:443"),
		otlptracegrpc.WithHeaders(map[string]string{
			"x-honeycomb-team":    os.Getenv("HONEYCOMB_API_KEY"),
			"x-honeycomb-dataset": "playground",
		}),
		otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")),
	}

	client := otlptracegrpc.NewClient(opts...)
	return otlptrace.New(ctx, client)
}

func newTraceProvider(exp *otlptrace.Exporter) *sdktrace.TracerProvider {
	// The service.name attribute is required.
	resource :=
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("artifacts-store"),
		)

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource),
	)
}

func init() {
	rootCmd.AddCommand(serveCmd)

	var (
		idleConnTimeout        int
		guessBucketTimeout     int
		directoryListingFormat bool
	)

	// Basic configs
	serveCmd.Flags().Bool("access-log", false, "toggle access log")
	viperBindFlag("accesslog", serveCmd.Flags().Lookup("access-log"))

	serveCmd.Flags().Bool("get-all-pages-in-dir", false, "toggle getting all pages in directories")
	viperBindFlag("allpagesindir", serveCmd.Flags().Lookup("get-all-pages-in-dir"))

	serveCmd.Flags().Bool("content-encoding", true, "toggle content encoding")
	viperBindFlag("contentencoding", serveCmd.Flags().Lookup("content-encoding"))

	serveCmd.Flags().Bool("directory-listing", false, "toggle directory listing")
	viperBindFlag("directorylisting", serveCmd.Flags().Lookup("directory-listing"))

	serveCmd.Flags().Bool("disable-compression", true, "toggle compression")
	viperBindFlag("disablecompression", serveCmd.Flags().Lookup("disable-compression"))

	serveCmd.Flags().Bool("disable-upstream-ssl", false, "toggle tls for the aws-sdk")
	viperBindFlag("disableupstreamssl", serveCmd.Flags().Lookup("disable-upstream-ssl"))

	serveCmd.Flags().Bool("enable-upload", false, "toggle upload, requires auth")
	viperBindFlag("enableupload", serveCmd.Flags().Lookup("enable-upload"))

	serveCmd.Flags().Bool("insecure-tls", false, "toggle insecure tls")
	viperBindFlag("insecuretls", serveCmd.Flags().Lookup("insecure-tls"))

	serveCmd.Flags().Int64("cors-max-age", 600, "CORS: max age in seconds") // nolint:gomnd
	viperBindFlag("corsmaxage", serveCmd.Flags().Lookup("cors-max-age"))

	serveCmd.Flags().Int("max-idle-connections", 150, "max idle connections") // nolint:gomnd
	viperBindFlag("maxidleconnections", serveCmd.Flags().Lookup("max-idle-connections"))

	serveCmd.Flags().String("basic-auth-user", "", "username for basic auth")
	viperBindFlag("basicauthuser", serveCmd.Flags().Lookup("basic-auth-user"))

	serveCmd.Flags().String("cors-allow-headers", "", "CORS: Comma-delimited list of the supported request headers")
	viperBindFlag("corsallowheaders", serveCmd.Flags().Lookup("cors-allow-headers"))

	serveCmd.Flags().String("cors-allow-methods", "", "CORS: comma-delimited list of the allowed - https://www.w3.org/Protocols/rfc2616/rfc2616-sec9.html")
	viperBindFlag("corsallowmethods", serveCmd.Flags().Lookup("cors-allow-methods"))

	serveCmd.Flags().String("cors-allow-origin", "", "CORS: a URI that may access the resource")
	viperBindFlag("corsalloworigin", serveCmd.Flags().Lookup("cors-allow-origin"))

	serveCmd.Flags().String("facility", "", "Location where the service is running")
	viperBindFlag("facility", serveCmd.Flags().Lookup("facility"))

	serveCmd.Flags().String("healthcheck-path", "", "path for healthcheck")
	viperBindFlag("healthcheckpath", serveCmd.Flags().Lookup("healthcheck-path"))

	serveCmd.Flags().String("listen-address", "::1", "host address to listen on")
	viperBindFlag("listenaddress", serveCmd.Flags().Lookup("listen-address"))

	serveCmd.Flags().String("listen-port", "21080", "port to listen on")
	viperBindFlag("listenport", serveCmd.Flags().Lookup("listen-port"))

	serveCmd.Flags().String("http-cache-control", "", "overrides S3's HTTP `Cache-Control` header")
	viperBindFlag("httpcachecontrol", serveCmd.Flags().Lookup("http-cache-control"))

	serveCmd.Flags().String("http-expires", "", "overrides S3's HTTP `Expires` header")
	viperBindFlag("httpexpires", serveCmd.Flags().Lookup("http-expires"))

	serveCmd.Flags().String("index-document", "index.html", "the index document for static website")
	viperBindFlag("indexdocument", serveCmd.Flags().Lookup("index-document"))

	serveCmd.Flags().String("upstream-bucket", "", "upstream s3 bucket")
	viperBindFlag("s3bucket", serveCmd.Flags().Lookup("upstream-bucket"))

	serveCmd.Flags().String("upstream-key-prefix", "", "upstream s3 path/key prefix")
	viperBindFlag("s3prefix", serveCmd.Flags().Lookup("upstream-key-prefix"))

	serveCmd.Flags().String("ssl-cert-path", "", "path to ssl cert")
	viperBindFlag("sslcert", serveCmd.Flags().Lookup("ssl-cert-path"))

	serveCmd.Flags().String("ssl-key-path", "", "path to ssl key")
	viperBindFlag("sslkey", serveCmd.Flags().Lookup("ssl-key-path"))

	serveCmd.Flags().String("strip-path", "", "strip path prefix")
	viperBindFlag("strippath", serveCmd.Flags().Lookup("strip-path"))

	if err := serveCmd.MarkFlagRequired("upstream-bucket"); err != nil {
		logger.Fatal(err)
	}

	if len(os.Getenv("S3_PROXY_BASIC_AUTH_PASS")) != 0 {
		viper.Set("basicauthpass", os.Getenv("S3_PROXY_BASIC_AUTH_PASS"))
	}

	// Configs that need transformation
	serveCmd.Flags().BoolVar(&directoryListingFormat, "directory-listing-format", false, "toggle directory listing spider formatted")

	if directoryListingFormat {
		viper.Set("directorylistingformat", "html")
	}

	serveCmd.Flags().IntP("idle-connection-timeout", "", 10, "idle connection timeout in seconds") // nolint:gomnd
	viper.Set("idleconntimeout", time.Duration(idleConnTimeout)*time.Second)

	serveCmd.Flags().IntP("guess-bucket-timeout", "", 10, "timeout, in seconds, for guessing bucket region") // nolint:gomnd
	viper.Set("guessbuckettimeout", time.Duration(guessBucketTimeout)*time.Second)

	// Configs with default AWS overrides
	serveCmd.Flags().StringP("aws-api-endpoint", "", "", "AWS API Endpoint")
	viperBindFlag("awsapiendpoint", serveCmd.Flags().Lookup("aws-api-endpoint"))

	if len(os.Getenv("AWS_API_ENDPOINT")) != 0 {
		viper.Set("awsapiendpoint", os.Getenv("AWS_API_ENDPOINT"))
	}

	serveCmd.Flags().StringP("aws-region", "", "us-east-1", "AWS region for s3, default AWS env vars will override")
	viperBindFlag("awsregion", serveCmd.Flags().Lookup("aws-region"))

	if len(os.Getenv("AWS_REGION")) != 0 {
		viper.Set("awsregion", os.Getenv("AWS_REGION"))
	} else if len(os.Getenv("AWS_DEFAULT_REGION")) != 0 {
		viper.Set("awsregion", os.Getenv("AWS_DEFAULT_REGION"))
	}
}

func toHTTPError(err error) (int, string) {
	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case s3.ErrCodeNoSuchBucket, s3.ErrCodeNoSuchKey:
			return http.StatusNotFound, aerr.Error()
		}

		log.Print("unknown s3 error")

		return http.StatusInternalServerError, aerr.Error()
	}

	log.Print("unknown http error")

	return http.StatusInternalServerError, err.Error()
}

func getS3File(ctx echo.Context) error {
	c := config.Config

	if len(c.Facility) > 0 {
		ctx.Response().Writer.Header().Add("Facility", c.Facility)
	}
	// Strip the prefix, if it's present.
	path := ctx.Request().URL.Path
	if len(c.StripPath) > 0 {
		path = strings.TrimPrefix(path, c.StripPath)
	}

	// Range header
	var rangeHeader *string
	if candidate := ctx.Request().Header.Get("Range"); !swag.IsZero(candidate) {
		rangeHeader = aws.String(candidate)
	}

	client := service.NewClient(ctx.Request().Context(), aws.String(config.Config.AwsRegion))

	obj, err := client.S3get(c.S3Bucket, c.S3KeyPrefix+path, rangeHeader)
	if err != nil {
		code, message := toHTTPError(err)
		log.Printf("error getting s3 object:[%d] %s", code, message)

		return ctx.String(code, message)
	}

	return ctx.Stream(http.StatusOK, echo.MIMEOctetStream, obj.Body)
}

func echoRouter() *echo.Echo {
	// A labstack/echo router
	router := echo.New()

	// Middleware
	router.Use(middleware.Logger())
	router.Use(middleware.Recover())

	// Metrics
	p := prometheus.NewPrometheus("echo", nil)
	p.Use(router)

	// Tracing

	router.Use(otelecho.Middleware("artifacts-store"))

	router.GET("/*", getS3File)
	router.HEAD("/*", getS3File)

	return router
}

func serve(ctx context.Context) {
	// Limits GOMAXPROCS in a container
	undo, err := maxprocs.Set(maxprocs.Logger(logger.Infof))
	defer undo()

	if err != nil {
		logger.Fatalf("failed to set GOMAXPROCS: %v", err)
	}

	// This maps the viper values to the Config object
	config.Load(logger)

	// Tracing
	exp, err := newExporter(ctx)
	if err != nil {
		log.Fatalf("failed to initialize exporter: %v", err)
	}
	// Create a new tracer provider with a batch span processor and the otlp exporter.
	tp := newTraceProvider(exp)

	// Handle this error in a sensible manner where possible
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	otel.SetTracerProvider(tp)

	router := echoRouter()
	addr := net.JoinHostPort(config.Config.ListenAddress, config.Config.ListenPort)

	// Set up signal channel for graceful shut down
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Listen & Serve
	go func() {
		logger.Infof("[service] listening on %s", addr)
		logger.Infof("[config] Proxy to %v", config.Config.S3Bucket)
		logger.Infof("[config] AWS Region: %v", config.Config.AwsRegion)

		router.Logger.Fatal(router.Start(addr))
	}()

	<-shutdown
	logger.Info("Shutting down")

	// Create a context to allow the server to provide deadline before shutting down
	// TODO: decide real time so clients don't get interrupted
	ctx, cancel := context.WithTimeout(ctx, time.Duration(600)*time.Second) // nolint:gomnd

	defer func() {
		cancel()
	}()

	if err := router.Shutdown(ctx); err != nil {
		logger.Errorf("Failed graceful shutdown", err)
	}
}
