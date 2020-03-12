package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/go-openapi/swag"
	"github.com/pottava/aws-s3-proxy/internal/config"
	"github.com/pottava/aws-s3-proxy/internal/controllers"
	common "github.com/pottava/aws-s3-proxy/internal/http"
	"github.com/pottava/aws-s3-proxy/internal/service"
)

var (
	ver    = "dev"
	commit string
	date   string
)

func main() {
	validateAwsConfigurations()

	httpMux := http.NewServeMux()

	httpMux.Handle("/", common.WrapHandler(controllers.AwsS3))

	httpMux.HandleFunc("/--version", func(w http.ResponseWriter, r *http.Request) {
		if len(commit) > 0 && len(date) > 0 {
			fmt.Fprintf(w, "%s-%s (built at %s)\n", ver, commit, date)
			return
		}
		fmt.Fprintln(w, ver)
	})

	// Listen & Serve
	addr := net.JoinHostPort(config.Config.Host, config.Config.Port)
	log.Printf("[service] listening on %s", addr)

	if (len(config.Config.SslCert) > 0) && (len(config.Config.SslKey) > 0) {
		log.Fatal(http.ListenAndServeTLS(
			addr, config.Config.SslCert, config.Config.SslKey, &slashFix{httpMux},
		))
	} else {
		log.Fatal(http.ListenAndServe(addr, &slashFix{httpMux}))
	}
}

func validateAwsConfigurations() {
	if len(os.Getenv("AWS_ACCESS_KEY_ID")) == 0 {
		log.Print("Not defined environment variable: AWS_ACCESS_KEY_ID")
	}
	if len(os.Getenv("AWS_SECRET_ACCESS_KEY")) == 0 {
		log.Print("Not defined environment variable: AWS_SECRET_ACCESS_KEY")
	}
	if len(os.Getenv("AWS_S3_BUCKET")) == 0 {
		log.Fatal("Missing required environment variable: AWS_S3_BUCKET")
	}
	if swag.IsZero(config.Config.AwsRegion) {
		config.Config.AwsRegion = "us-east-1"
		if region, err := service.GuessBucketRegion(config.Config.S3Bucket); err == nil {
			config.Config.AwsRegion = region
		}
	}
}

type slashFix struct {
	mux http.Handler
}

func (h *slashFix) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var pathBuilder strings.Builder
	slash := false
	for _, c := range r.URL.Path {
		if c == '/' {
			if !slash {
				pathBuilder.WriteRune(c)
			}
			slash = true
		} else {
			pathBuilder.WriteRune(c)
			slash = false
		}
	}
	r.URL.Path = pathBuilder.String()
	h.mux.ServeHTTP(w, r)
}
