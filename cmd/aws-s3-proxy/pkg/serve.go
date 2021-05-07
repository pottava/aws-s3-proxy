package cmd

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/automaxprocs/maxprocs"

	"github.com/packethost/aws-s3-proxy/internal/config"
	"github.com/packethost/aws-s3-proxy/internal/controllers"
	common "github.com/packethost/aws-s3-proxy/internal/http"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "serve the s3 proxy",
	Run: func(cmd *cobra.Command, args []string) {
		serve()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	var (
		idleConnTimeout        int
		guessBucketTimeout     int
		directoryListingFormat bool
	)

	// Basic configs
	serveCmd.Flags().BoolP("access-log", "", false, "toggle access log")
	viperBindFlag("accesslog", serveCmd.Flags().Lookup("access-log"))

	serveCmd.Flags().BoolP("get-all-pages-in-dir", "", false, "toggle getting all pages in directories")
	viperBindFlag("allpagesindir", serveCmd.Flags().Lookup("get-all-pages-in-dir"))

	serveCmd.Flags().BoolP("content-encoding", "", true, "toggle content encoding")
	viperBindFlag("contentencoding", serveCmd.Flags().Lookup("content-encoding"))

	serveCmd.Flags().BoolP("directory-listing", "", false, "toggle directory listing")
	viperBindFlag("directorylisting", serveCmd.Flags().Lookup("directory-listing"))

	serveCmd.Flags().BoolP("disable-compression", "", true, "toggle compression")
	viperBindFlag("disablecompression", serveCmd.Flags().Lookup("disable-compression"))

	serveCmd.Flags().BoolP("disable-upstream-ssl", "", false, "toggle tls for the aws-sdk")
	viperBindFlag("disableupstreamssl", serveCmd.Flags().Lookup("disable-upstream-ssl"))

	serveCmd.Flags().BoolP("insecure-tls", "", false, "toggle insecure tls")
	viperBindFlag("insecuretls", serveCmd.Flags().Lookup("insecure-tls"))

	serveCmd.Flags().Int64P("cors-max-age", "", 600, "CORS: max age in seconds")
	viperBindFlag("corsmaxage", serveCmd.Flags().Lookup("cors-max-age"))

	serveCmd.Flags().IntP("max-idle-connections", "", 150, "max idle connections")
	viperBindFlag("maxidleconnections", serveCmd.Flags().Lookup("max-idle-connections"))

	serveCmd.Flags().StringP("basic-auth-user", "", "", "username for basic auth")
	viperBindFlag("basicauthuser", serveCmd.Flags().Lookup("basic-auth-user"))

	serveCmd.Flags().StringP("cors-allow-headers", "", "", "CORS: Comma-delimited list of the supported request headers")
	viperBindFlag("corsallowheaders", serveCmd.Flags().Lookup("cors-allow-headers"))

	serveCmd.Flags().StringP("cors-allow-methods", "", "", "CORS: comma-delimited list of the allowed - https://www.w3.org/Protocols/rfc2616/rfc2616-sec9.html")
	viperBindFlag("corsallowmethods", serveCmd.Flags().Lookup("cors-allow-methods"))

	serveCmd.Flags().StringP("cors-allow-origin", "", "", "CORS: a URI that may access the resource")
	viperBindFlag("corsalloworigin", serveCmd.Flags().Lookup("cors-allow-origin"))

	serveCmd.Flags().StringP("healthcheck-path", "", "", "path for healthcheck")
	viperBindFlag("healthcheckpath", serveCmd.Flags().Lookup("healthcheck-path"))

	serveCmd.Flags().StringP("listen-address", "", "::1", "host address to listen on")
	viperBindFlag("listenaddress", serveCmd.Flags().Lookup("listen-address"))

	serveCmd.Flags().StringP("listen-port", "", "21080", "port to listen on")
	viperBindFlag("listenport", serveCmd.Flags().Lookup("listen-port"))

	serveCmd.Flags().StringP("http-cache-control", "", "", "overrides S3's HTTP `Cache-Control` header")
	viperBindFlag("httpcachecontrol", serveCmd.Flags().Lookup("http-cache-control"))

	serveCmd.Flags().StringP("http-expires", "", "", "overrides S3's HTTP `Expires` header")
	viperBindFlag("httpexpires", serveCmd.Flags().Lookup("http-expires"))

	serveCmd.Flags().StringP("index-document", "", "index.html", "the index document for static website")
	viperBindFlag("indexdocument", serveCmd.Flags().Lookup("index-document"))

	serveCmd.Flags().StringP("upstream-bucket", "", "", "upstream s3 bucket")
	viperBindFlag("s3bucket", serveCmd.Flags().Lookup("upstream-bucket"))

	serveCmd.Flags().StringP("upstream-key-prefix", "", "", "upstream s3 path/key prefix")
	viperBindFlag("s3prefix", serveCmd.Flags().Lookup("upstream-key-prefix"))

	serveCmd.Flags().StringP("ssl-cert-path", "", "", "path to ssl cert")
	viperBindFlag("sslcert", serveCmd.Flags().Lookup("ssl-cert-path"))

	serveCmd.Flags().StringP("ssl-key-path", "", "", "path to ssl key")
	viperBindFlag("sslkey", serveCmd.Flags().Lookup("ssl-key-path"))

	serveCmd.Flags().StringP("strip-path", "", "", "strip path prefix")
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

	serveCmd.Flags().IntP("idle-connection-timeout", "", 10, "idle connection timeout in seconds")
	viper.Set("idleconntimeout", time.Duration(idleConnTimeout)*time.Second)

	serveCmd.Flags().IntP("guess-bucket-timeout", "", 10, "timeout, in seconds, for guessing bucket region")
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

func serve() {
	// Limits GOMAXPROCS in a container
	undo, err := maxprocs.Set(maxprocs.Logger(logger.Infof))
	defer undo()

	if err != nil {
		logger.Fatalf("failed to set GOMAXPROCS: %v", err)
	}

	// This maps the viper values to the Config object
	config.Load()

	// A gorilla/mux router is used to allow for more control
	router := mux.NewRouter()
	router.PathPrefix("/").Handler(common.WrapHandler(controllers.AwsS3)).Methods("GET")

	server := &http.Server{
		Handler: router,
		Addr:    net.JoinHostPort(config.Config.ListenAddress, config.Config.ListenPort),
	}

	// Set up signal channel for graceful shut down
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Listen & Serve
	go func() {
		logger.Infof("[service] listening on %s", server.Addr)
		logger.Infof("[config] Proxy to %v", config.Config.S3Bucket)
		logger.Infof("[config] AWS Region: %v", config.Config.AwsRegion)

		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("failed starting the server: %s", err.Error())
		}
	}()

	<-shutdown
	logger.Info("Shutting down")

	// Create a context to allow the server to provide deadline before shutting down
	// TODO: decide real time so clients don't get interrupted
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(600)*time.Second) // nolint:gomnd

	defer func() {
		cancel()
	}()

	if err := server.Shutdown(ctx); err != nil {
		logger.Errorf("Failed graceful shutdown", err)
	}
}
