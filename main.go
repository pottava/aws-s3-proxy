package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/go-openapi/swag"
	"github.com/pottava/aws-s3-proxy/internal/common"
	"github.com/pottava/aws-s3-proxy/internal/config"
	"github.com/pottava/aws-s3-proxy/internal/controllers"
)

var (
	version string
	date    string
)

func main() {
	ctx := context.Background()
	if swag.IsZero(config.Config.AwsRegion) {
		config.Config.AwsRegion = "us-east-1"
		if region, err := common.GuessBucketRegion(ctx, config.Config.S3Bucket); err == nil {
			config.Config.AwsRegion = region
		}
	}
	http.Handle("/", common.WrapHandler(controllers.AwsS3))

	http.HandleFunc("/--version", func(w http.ResponseWriter, r *http.Request) {
		if len(version) > 0 && len(date) > 0 {
			fmt.Fprintf(w, "version: %s (built at %s)\n", version, date)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	})

	// Listen & Serve
	addr := net.JoinHostPort(config.Config.Host, config.Config.Port)
	log.Printf("[service] listening on %s", addr)

	if (len(config.Config.SslCert) > 0) && (len(config.Config.SslKey) > 0) {
		log.Fatal(http.ListenAndServeTLS(
			addr, config.Config.SslCert, config.Config.SslKey, nil,
		))
	} else {
		log.Fatal(http.ListenAndServe(addr, nil))
	}
}
