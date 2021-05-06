package cmd

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/packethost/aws-s3-proxy/internal/config"
	"github.com/packethost/aws-s3-proxy/internal/controllers"
	common "github.com/packethost/aws-s3-proxy/internal/http"
)

var cfgFile string
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "serve the s3 proxy",
	Run: func(cmd *cobra.Command, args []string) {
		serve()
	},
}

func init() {
	var (
		idleConnTimeout        int
		guessBucketTimeout     int
		directoryListingFormat bool
	)

	c := config.Config
	// Basic configs
	cobra.OnInitialize(initConfig)
	serveCmd.PersistentFlags().BoolVar(&c.AccessLog, "access-log", false, "toggle access log")
	serveCmd.PersistentFlags().BoolVar(&c.AllPagesInDir, "get-all-pages-in-dir", false, "toggle getting all pages in directories")
	serveCmd.PersistentFlags().BoolVar(&c.ContentEncoding, "content-access", true, "toggle content encoding")
	serveCmd.PersistentFlags().BoolVar(&c.DirectoryListing, "directory-listing", false, "toggle directory listing")
	serveCmd.PersistentFlags().BoolVar(&c.DisableCompression, "disable-compression", true, "toggle compression")
	serveCmd.PersistentFlags().BoolVar(&c.DisableUpsteamSSL, "disable-upstream-ssl", false, "toggle tls for the aws-sdk")
	serveCmd.PersistentFlags().StringVar(&c.HTTPCacheControl, "http-cache-control", "", "overrides S3's HTTP `Cache-Control` header")
	serveCmd.PersistentFlags().StringVar(&c.HTTPExpires, "http-expires", "", "overrides S3's HTTP `Expires` header")
	serveCmd.PersistentFlags().StringVar(&c.BasicAuthUser, "basic-auth-user", "", "username for basic auth")
	serveCmd.PersistentFlags().StringVar(&c.SslCert, "ssl-cert-path", "", "path to ssl cert")
	serveCmd.PersistentFlags().StringVar(&c.SslKey, "ssl-key-path", "", "path to ssl key")
	serveCmd.PersistentFlags().StringVar(&c.StripPath, "strip-path", "", "strip path prefix")
	serveCmd.PersistentFlags().StringVar(&c.CorsAllowOrigin, "cors-allow-origin", "", "CORS: a URI that may access the resource")
	serveCmd.PersistentFlags().StringVar(&c.CorsAllowMethods, "cors-allow-methods", "", "CORS: comma-delimited list of the allowed - https://www.w3.org/Protocols/rfc2616/rfc2616-sec9.html")
	serveCmd.PersistentFlags().StringVar(&c.CorsAllowHeaders, "cors-allow-headers", "", "CORS:Comma-delimited list of the supported request headers")
	serveCmd.PersistentFlags().BoolVar(&c.InsecureTLS, "insecure-tls", false, "toggle insecure tls")
	serveCmd.PersistentFlags().StringVar(&c.HealthCheckPath, "healthcheck-path", "", "path for healthcheck")
	serveCmd.PersistentFlags().Int64Var(&c.CorsMaxAge, "cors-max-age", 600, "CORS: max age in seconds")
	serveCmd.PersistentFlags().IntVar(&c.MaxIdleConns, "max-idle-connections", 150, "max idle connections")
	serveCmd.PersistentFlags().StringVar(&c.Host, "listen-address", "::1", "host address to listen on")
	serveCmd.PersistentFlags().StringVar(&c.IndexDocument, "index-document", "index.html", "the index document for static website")
	serveCmd.PersistentFlags().StringVar(&c.Port, "list-port", "21080", "port to listen on")
	serveCmd.PersistentFlags().StringVar(&c.S3Bucket, "upstream-bucket", "", "upstream s3 bucket")
	serveCmd.PersistentFlags().StringVar(&c.S3KeyPrefix, "upstream-key-prefix", "", "upstream s3 path/key prefix")
	serveCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.aws-s3-proxy.yaml)")

	c.BasicAuthPass = viper.GetString("basic-auth-pass")

	// Configs that need transformation
	serveCmd.PersistentFlags().BoolVar(&directoryListingFormat, "directory-listing-format", false, "toggle directory listing spider formatted")

	if directoryListingFormat {
		c.DirListingFormat = "html"
	}

	serveCmd.PersistentFlags().IntVar(&idleConnTimeout, "idle-connection-timeout", 10, "idle connection timeout in seconds")
	c.IdleConnTimeout = time.Duration(idleConnTimeout) * time.Second

	serveCmd.PersistentFlags().IntVar(&guessBucketTimeout, "guess-bucket-timeout", 10, "timeout, in seconds, for guessing bucket region")
	c.GuessBucketTimeout = time.Duration(guessBucketTimeout) * time.Second

	// Configs with default AWS overrides
	serveCmd.PersistentFlags().StringVar(&c.AwsAPIEndpoint, "aws-api-endpoint", "", "AWS API Endpoint")

	if len(os.Getenv("AWS_API_ENDPOINT")) != 0 {
		c.AwsAPIEndpoint = os.Getenv("AWS_API_ENDPOINT")
	}

	serveCmd.PersistentFlags().StringVar(&c.AwsRegion, "aws-region", "us-east-1", "AWS region for s3, default AWS env vars will override")

	if len(os.Getenv("AWS_REGION")) != 0 {
		c.AwsRegion = os.Getenv("AWS_REGION")
	} else if len(os.Getenv("AWS_DEFAULT_REGION")) != 0 {
		c.AwsRegion = os.Getenv("AWS_DEFAULT_REGION")
	}

	rootCmd.AddCommand(serveCmd)
}

func serve() {
	http.Handle("/", common.WrapHandler(controllers.AwsS3))

	// Listen & Serve
	addr := net.JoinHostPort(config.Config.Host, config.Config.Port)
	log.Printf("[service] listening on %s", addr)
	log.Printf("[config] Proxy to %v", config.Config.S3Bucket)
	log.Printf("[config] AWS Region: %v", config.Config.AwsRegion)

	log.Fatal(http.ListenAndServe(addr, nil))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".decuddle" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".s3-proxy")
	}

	// Check for ENV variables set
	// All ENV vars will be prefixed with "S3_PROXY_"
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("s3-proxy")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
