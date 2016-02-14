package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type config struct {
	awsRegion     string // AWS_REGION
	s3Bucket      string // AWS_S3_BUCKET
	basicAuthUser string // BASIC_AUTH_USER
	basicAuthPass string // BASIC_AUTH_PASS
	port          string // APP_PORT
	accessLog     bool   // ACCESS_LOG
	sslCert       string // SSL_CERT_PATH
	sslKey        string // SSL_KEY_PATH
}

var (
	version string
	date    string
	c       *config
)

func main() {
	c = configFromEnvironmentVariables()

	http.Handle("/", wrapper(awss3))

	http.HandleFunc("/--version", func(w http.ResponseWriter, r *http.Request) {
		if len(version) > 0 && len(date) > 0 {
			fmt.Fprintf(w, "version: %s (built at %s)", version, date)
		}
		w.WriteHeader(http.StatusOK)
	})

	// Listen & Serve
	log.Printf("[service] listening on port %s", c.port)
	if (len(c.sslCert) > 0) && (len(c.sslKey) > 0) {
		log.Fatal(http.ListenAndServeTLS(":"+c.port, c.sslCert, c.sslKey, nil))
	} else {
		log.Fatal(http.ListenAndServe(":"+c.port, nil))
	}
}

func configFromEnvironmentVariables() *config {
	if len(os.Getenv("AWS_ACCESS_KEY_ID")) == 0 {
		log.Print("Not defined environment variable: AWS_ACCESS_KEY_ID")
	}
	if len(os.Getenv("AWS_SECRET_ACCESS_KEY")) == 0 {
		log.Print("Not defined environment variable: AWS_SECRET_ACCESS_KEY")
	}
	if len(os.Getenv("AWS_REGION")) == 0 {
		log.Fatal("Missing required environment variable: AWS_REGION")
	}
	if len(os.Getenv("AWS_S3_BUCKET")) == 0 {
		log.Fatal("Missing required environment variable: AWS_S3_BUCKET")
	}
	port := os.Getenv("APP_PORT")
	if len(port) == 0 {
		port = "80"
	}
	accessLog := false
	if b, err := strconv.ParseBool(os.Getenv("ACCESS_LOG")); err == nil {
		accessLog = b
	}
	conf := &config{
		awsRegion:     os.Getenv("AWS_REGION"),
		s3Bucket:      os.Getenv("AWS_S3_BUCKET"),
		basicAuthUser: os.Getenv("BASIC_AUTH_USER"),
		basicAuthPass: os.Getenv("BASIC_AUTH_PASS"),
		port:          port,
		accessLog:     accessLog,
		sslCert:       os.Getenv("SSL_CERT_PATH"),
		sslKey:        os.Getenv("SSL_KEY_PATH"),
	}
	// Proxy
	log.Printf("[config] Proxy to %v", conf.s3Bucket)

	// TLS pem files
	if (len(conf.sslCert) > 0) && (len(conf.sslKey) > 0) {
		log.Print("[config] TLS enabled.")
	}
	// Basic authentication
	if (len(conf.basicAuthUser) > 0) && (len(conf.basicAuthPass) > 0) {
		log.Printf("[config] Basic authentication: %s", conf.basicAuthUser)
	}
	return conf
}

func wrapper(f func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (len(c.basicAuthUser) > 0) && (len(c.basicAuthPass) > 0) && !auth(r, c.basicAuthUser, c.basicAuthPass) {
			w.Header().Set("WWW-Authenticate", `Basic realm="REALM"`)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		f(w, r)
		if c.accessLog {
			log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		}
	})
}

func auth(r *http.Request, user, pass string) bool {
	if username, password, ok := r.BasicAuth(); ok {
		return username == user && password == pass
	}
	return false
}

func awss3(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if strings.HasSuffix(path, "/") {
		path += "index.html"
	}
	obj, err := s3get(c.s3Bucket, path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	setStrHeader(w, "Cache-Control", obj.CacheControl)
	setStrHeader(w, "Content-Disposition", obj.ContentDisposition)
	setStrHeader(w, "Content-Encoding", obj.ContentEncoding)
	setStrHeader(w, "Content-Language", obj.ContentLanguage)
	setIntHeader(w, "Content-Length", obj.ContentLength)
	setStrHeader(w, "Content-Range", obj.ContentRange)
	setStrHeader(w, "Content-Type", obj.ContentType)
	setStrHeader(w, "ETag", obj.ETag)
	setStrHeader(w, "Expires", obj.Expires)
	setTimeHeader(w, "Last-Modified", obj.LastModified)
	io.Copy(w, obj.Body)
}

func s3get(backet, key string) (*s3.GetObjectOutput, error) {
	req := &s3.GetObjectInput{
		Bucket: aws.String(backet),
		Key:    aws.String(key),
	}
	return s3.New(session.New(aws.NewConfig().WithRegion(c.awsRegion))).GetObject(req)
}

func setStrHeader(w http.ResponseWriter, key string, value *string) {
	if value == nil || len(*value) == 0 {
		return
	}
	w.Header().Add(key, *value)
}

func setIntHeader(w http.ResponseWriter, key string, value *int64) {
	if value == nil || *value == 0 {
		return
	}
	w.Header().Add(key, strconv.FormatInt(*value, 10))
}

func setTimeHeader(w http.ResponseWriter, key string, value *time.Time) {
	if value == nil || reflect.DeepEqual(*value, time.Time{}) {
		return
	}
	w.Header().Add(key, value.UTC().Format(http.TimeFormat))
}
