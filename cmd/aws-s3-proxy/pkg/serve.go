package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"

	"go.uber.org/automaxprocs/maxprocs"

	"github.com/packethost/aws-s3-proxy/internal/config"
	echozaplogger "github.com/packethost/aws-s3-proxy/internal/middleware/echo-zap-logger"
	"github.com/packethost/aws-s3-proxy/internal/s3"
)

var (
	maxIdleConns     = 150
	idleTimeout      = 10
	exitDelayTimeout = 600
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "serve the s3 proxy",
	Run: func(cmd *cobra.Command, args []string) {
		serve(cmd.Context())
	},
}

func httpFlags() {
	serveCmd.Flags().String("facility", "", "Location where the service is running")
	viperBindFlag("httpopts.facility", serveCmd.Flags().Lookup("facility"))

	serveCmd.Flags().String("http-cache-control", "", "overrides S3's HTTP `Cache-Control` header")
	viperBindFlag("httpopts.httpcachecontrol", serveCmd.Flags().Lookup("http-cache-control"))

	serveCmd.Flags().String("http-expires", "", "overrides S3's HTTP `Expires` header")
	viperBindFlag("httpopts.httpexpires", serveCmd.Flags().Lookup("http-expires"))

	serveCmd.Flags().String("healthcheck-path", "", "path for healthcheck")
	viperBindFlag("httpopts.healthcheckpath", serveCmd.Flags().Lookup("healthcheck-path"))
}

// set flags used for the http router
func serverFlags() {
	serveCmd.Flags().String("listen-address", "::1", "host address to listen on")
	viperBindFlag("serveropts.listenaddress", serveCmd.Flags().Lookup("listen-address"))

	serveCmd.Flags().String("listen-port", "21080", "port to listen on")
	viperBindFlag("serveropts.listenport", serveCmd.Flags().Lookup("listen-port"))
}

func s3Flags() {
	// Common flags
	stores := []string{"primary-store", "secondary-store"}
	boolFlags := []struct {
		long         string
		describe     string
		defaultValue bool
		required     bool
	}{
		{
			long:     "disable-compression",
			describe: "toggle compressions",
		},
		{
			long:     "disable-bucket-ssl",
			describe: "toggle tls for the aws-sdk",
		},
		{
			long:     "insecure-tls",
			describe: "toogle tls verify",
		},
	}
	intFlags := []struct {
		long         string
		describe     string
		defaultValue int
		required     bool
	}{
		{
			long:         "max-idle-connections",
			describe:     "max idle connections",
			defaultValue: maxIdleConns,
		},
		{
			long:         "idle-connection-timeout",
			describe:     "idle connection timeout in seconds",
			defaultValue: idleTimeout,
		},
	}
	stringFlags := []struct {
		long         string
		describe     string
		defaultValue string
		required     bool
	}{
		{
			long:     "access-key",
			describe: "s3 access-key",
		},
		{
			long:     "secret-key",
			describe: "s3 secret-access-key",
		},
		{
			long:     "bucket",
			describe: "bucket name",
			required: true,
		},
		{
			long:     "endpoint",
			describe: "endpoint URL (hostname only or fully qualified URI)",
		},
		{
			long:     "region",
			describe: "region for bucket",
		},
	}

	for _, store := range stores {
		for _, boolFlag := range boolFlags {
			// concatenated flag name
			f := fmt.Sprintf("%s-%s", store, boolFlag.long)

			// config json path name
			cfgPath := fmt.Sprintf("%s.%s",
				strings.ReplaceAll(store, "-", ""),
				strings.ReplaceAll(boolFlag.long, "-", ""),
			)

			serveCmd.Flags().Bool(f, boolFlag.defaultValue, boolFlag.describe)

			if boolFlag.required {
				if err := serveCmd.MarkFlagRequired(f); err != nil {
					logger.Fatal(err)
				}
			}

			viperBindFlag(cfgPath, serveCmd.Flags().Lookup(f))
		}

		for _, intFlag := range intFlags {
			f := fmt.Sprintf("%s-%s", store, intFlag.long)
			cfgPath := fmt.Sprintf("%s.%s",
				strings.ReplaceAll(store, "-", ""),
				strings.ReplaceAll(intFlag.long, "-", ""),
			)

			serveCmd.Flags().Int(f, intFlag.defaultValue, intFlag.describe)

			if intFlag.required {
				if err := serveCmd.MarkFlagRequired(f); err != nil {
					logger.Fatal(err)
				}
			}

			viperBindFlag(cfgPath, serveCmd.Flags().Lookup(f))
		}

		for _, stringFlag := range stringFlags {
			f := fmt.Sprintf("%s-%s", store, stringFlag.long)

			cfgPath := fmt.Sprintf("%s.%s",
				strings.ReplaceAll(store, "-", ""),
				strings.ReplaceAll(stringFlag.long, "-", ""),
			)

			serveCmd.Flags().String(f, stringFlag.defaultValue, stringFlag.describe)

			if stringFlag.required {
				if err := serveCmd.MarkFlagRequired(f); err != nil {
					logger.Fatal(err)
				}
			}

			viperBindFlag(cfgPath, serveCmd.Flags().Lookup(f))
		}
	}

	// Secondary bucket flags
	serveCmd.Flags().Bool("secondary-fall-back", false, "toggle read from secondary")
	viperBindFlag("readthrough.enabled", serveCmd.Flags().Lookup("secondary-fall-back"))
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Set up router options from flags
	serverFlags()

	// Set flags for the router
	httpFlags()

	// S3 store configs
	s3Flags()
}

func makeRouter() (*echo.Echo, *string) {
	c := config.Cfg
	s := c.ServerOpts
	// A labstack/echo router

	router := echo.New()

	router.Use(
		middleware.RequestID(),
		middleware.Recover(),
		middleware.Decompress(),
		middleware.Gzip(),
		echozaplogger.ZapLogger(logger.Desugar()),
	)

	// Metrics
	p := prometheus.NewPrometheus("echo", nil)
	p.Use(router)

	router.GET("/*", s3.Handler(s3.AwsS3Get))
	router.HEAD("/*", s3.Handler(s3.AwsS3Get))

	addr := net.JoinHostPort(s.ListenAddress, s.ListenPort)

	return router, &addr
}

func serve(ctx context.Context) {
	// Limits GOMAXPROCS in a container
	undo, err := maxprocs.Set(maxprocs.Logger(logger.Infof))
	defer undo()

	if err != nil {
		logger.Fatalf("failed to set GOMAXPROCS: %v", err)
	}

	// This maps the viper values to the Config object
	config.Load(ctx, logger)

	router, addr := makeRouter()

	// Set up signal channel for graceful shut down
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Listen & Serve
	go func() {
		logger.Infof("[service] listening on %s", *addr)

		if config.Cfg.PrimaryStore.Session == nil {
			logger.Error("invalid primary bucket session")

			shutdown <- os.Interrupt
		}

		logger.Infof("[config] Primary bucket '%s'", config.Cfg.PrimaryStore.Bucket)

		if config.Cfg.ReadThrough.Enabled {
			logger.Infof("[config] Secondary bucket '%s'", config.Cfg.SecondaryStore.Bucket)
			
			if config.Cfg.SecondaryStore.Session == nil {
				logger.Error("invalid secoindary bucket session")

				shutdown <- os.Interrupt
			}
		}

		router.Logger.Fatal(router.Start(*addr))
	}()

	<-shutdown
	logger.Info("Shutting down")

	// Create a context to allow the server to provide deadline before shutting down
	ctx, cancel := context.WithTimeout(ctx, time.Duration(exitDelayTimeout)*time.Second)

	defer func() {
		cancel()
	}()

	if err := router.Shutdown(ctx); err != nil {
		logger.Errorf("Failed graceful shutdown", err)
	}
}
