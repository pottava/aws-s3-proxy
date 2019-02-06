package main

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type config struct { // nolint
	awsRegion        string // AWS_REGION
	awsAPIEndpoint   string // AWS_API_ENDPOINT
	s3Bucket         string // AWS_S3_BUCKET
	s3KeyPrefix      string // AWS_S3_KEY_PREFIX
	indexDocument    string // INDEX_DOCUMENT
	directoryListing bool   // DIRECTORY_LISTINGS
	dirListingFormat string // DIRECTORY_LISTINGS_FORMAT
	httpCacheControl string // HTTP_CACHE_CONTROL (max-age=86400, no-cache ...)
	httpExpires      string // HTTP_EXPIRES (Thu, 01 Dec 1994 16:00:00 GMT ...)
	basicAuthUser    string // BASIC_AUTH_USER
	basicAuthPass    string // BASIC_AUTH_PASS
	port             string // APP_PORT
	host             string // APP_HOST
	accessLog        bool   // ACCESS_LOG
	sslCert          string // SSL_CERT_PATH
	sslKey           string // SSL_KEY_PATH
	stripPath        string // STRIP_PATH
	contentEncoding  bool   // CONTENT_ENCODING
	corsAllowOrigin  string // CORS_ALLOW_ORIGIN
	corsAllowMethods string // CORS_ALLOW_METHODS
	corsAllowHeaders string // CORS_ALLOW_HEADERS
	corsMaxAge       int64  // CORS_MAX_AGE
	healthCheckPath  string // HEALTHCHECK_PATH
	allPagesInDir    bool   // GET_ALL_PAGES_IN_DIR
}

type symlink struct {
	URL string
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
			fmt.Fprintf(w, "version: %s (built at %s)\n", version, date)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	})

	// Listen & Serve
	addr := net.JoinHostPort(c.host, c.port)
	log.Printf("[service] listening on %s", addr)
	if (len(c.sslCert) > 0) && (len(c.sslKey) > 0) {
		log.Fatal(http.ListenAndServeTLS(addr, c.sslCert, c.sslKey, nil))
	} else {
		log.Fatal(http.ListenAndServe(addr, nil))
	}
}

func configFromEnvironmentVariables() *config {
	if len(os.Getenv("AWS_ACCESS_KEY_ID")) == 0 {
		log.Print("Not defined environment variable: AWS_ACCESS_KEY_ID")
	}
	if len(os.Getenv("AWS_SECRET_ACCESS_KEY")) == 0 {
		log.Print("Not defined environment variable: AWS_SECRET_ACCESS_KEY")
	}
	if len(os.Getenv("AWS_S3_BUCKET")) == 0 {
		log.Fatal("Missing required environment variable: AWS_S3_BUCKET")
	}
	region := os.Getenv("AWS_REGION")
	if len(region) == 0 {
		region = os.Getenv("AWS_DEFAULT_REGION")
		if len(region) == 0 {
			region = "us-east-1"
		}
	}
	endpoint := os.Getenv("AWS_API_ENDPOINT")
	if len(endpoint) == 0 {
		endpoint = ""
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
	contentEncoging := false
	if b, err := strconv.ParseBool(os.Getenv("CONTENT_ENCODING")); err == nil {
		contentEncoging = b
	}
	corsMaxAge := int64(600)
	if i, err := strconv.ParseInt(os.Getenv("CORS_MAX_AGE"), 10, 64); err == nil {
		corsMaxAge = i
	}
	allPagesInDir := false
	if b, err := strconv.ParseBool(os.Getenv("GET_ALL_PAGES_IN_DIR")); err == nil {
		allPagesInDir = b
	}
	conf := &config{
		awsRegion:        region,
		awsAPIEndpoint:   endpoint,
		s3Bucket:         os.Getenv("AWS_S3_BUCKET"),
		s3KeyPrefix:      os.Getenv("AWS_S3_KEY_PREFIX"),
		indexDocument:    indexDocument,
		directoryListing: directoryListings,
		dirListingFormat: os.Getenv("DIRECTORY_LISTINGS_FORMAT"),
		httpCacheControl: os.Getenv("HTTP_CACHE_CONTROL"),
		httpExpires:      os.Getenv("HTTP_EXPIRES"),
		basicAuthUser:    os.Getenv("BASIC_AUTH_USER"),
		basicAuthPass:    os.Getenv("BASIC_AUTH_PASS"),
		port:             port,
		host:             os.Getenv("APP_HOST"),
		accessLog:        accessLog,
		sslCert:          os.Getenv("SSL_CERT_PATH"),
		sslKey:           os.Getenv("SSL_KEY_PATH"),
		stripPath:        os.Getenv("STRIP_PATH"),
		contentEncoding:  contentEncoging,
		corsAllowOrigin:  os.Getenv("CORS_ALLOW_ORIGIN"),
		corsAllowMethods: os.Getenv("CORS_ALLOW_METHODS"),
		corsAllowHeaders: os.Getenv("CORS_ALLOW_HEADERS"),
		corsMaxAge:       corsMaxAge,
		healthCheckPath:  os.Getenv("HEALTHCHECK_PATH"),
		allPagesInDir:    allPagesInDir,
	}
	// Proxy
	log.Printf("[config] Proxy to %v", conf.s3Bucket)
	log.Printf("[config] AWS Region: %v", conf.awsRegion)

	// TLS pem files
	if (len(conf.sslCert) > 0) && (len(conf.sslKey) > 0) {
		log.Print("[config] TLS enabled.")
	}
	// Basic authentication
	if (len(conf.basicAuthUser) > 0) && (len(conf.basicAuthPass) > 0) {
		log.Printf("[config] Basic authentication: %s", conf.basicAuthUser)
	}
	// CORS
	if (len(conf.corsAllowOrigin) > 0) && (conf.corsMaxAge > 0) {
		log.Printf("[config] CORS enabled: %s", conf.corsAllowOrigin)
	}
	return conf
}

type custom struct {
	io.Writer
	http.ResponseWriter
	status int
}

func (r *custom) Write(b []byte) (int, error) {
	if r.Header().Get("Content-Type") == "" {
		r.Header().Set("Content-Type", http.DetectContentType(b))
	}
	return r.Writer.Write(b)
}

func (r *custom) WriteHeader(status int) {
	r.ResponseWriter.WriteHeader(status)
	r.status = status
}

func wrapper(f func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (len(c.corsAllowOrigin) > 0) && (len(c.corsAllowMethods) > 0) && (len(c.corsAllowHeaders) > 0) && (c.corsMaxAge > 0) {
			w.Header().Set("Access-Control-Allow-Origin", c.corsAllowOrigin)
			w.Header().Set("Access-Control-Allow-Methods", c.corsAllowMethods)
			w.Header().Set("Access-Control-Allow-Headers", c.corsAllowHeaders)
			w.Header().Set("Access-Control-Max-Age", strconv.FormatInt(c.corsMaxAge, 10))
		}
		if (len(c.basicAuthUser) > 0) && (len(c.basicAuthPass) > 0) && !auth(r) && !isHealthCheckPath(r) {
			w.Header().Set("WWW-Authenticate", `Basic realm="REALM"`)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		proc := time.Now()
		addr := r.RemoteAddr
		if ip, found := header(r, "X-Forwarded-For"); found {
			addr = ip
		}
		ioWriter := w.(io.Writer)
		if encodings, found := header(r, "Accept-Encoding"); found && c.contentEncoding {
			for _, encoding := range splitCsvLine(encodings) {
				if encoding == "gzip" {
					w.Header().Set("Content-Encoding", "gzip")
					g := gzip.NewWriter(w)
					defer g.Close()
					ioWriter = g
					break
				}
				if encoding == "deflate" {
					w.Header().Set("Content-Encoding", "deflate")
					z := zlib.NewWriter(w)
					defer z.Close()
					ioWriter = z
					break
				}
			}
		}
		writer := &custom{Writer: ioWriter, ResponseWriter: w, status: http.StatusOK}
		f(writer, r)

		if c.accessLog {
			log.Printf("[%s] %.3f %d %s %s",
				addr, time.Since(proc).Seconds(),
				writer.status, r.Method, r.URL)
		}
	})
}

func auth(r *http.Request) bool {
	if username, password, ok := r.BasicAuth(); ok {
		return username == c.basicAuthUser &&
			password == c.basicAuthPass
	}
	return false
}

func isHealthCheckPath(r *http.Request) bool {
	path := r.URL.Path
	if len(c.healthCheckPath) > 0 && path == c.healthCheckPath {
		return true
	}
	return false
}

func header(r *http.Request, key string) (string, bool) {
	if r.Header == nil {
		return "", false
	}
	if candidate := r.Header[key]; len(candidate) > 0 {
		return candidate[0], true
	}
	return "", false
}

func splitCsvLine(data string) []string {
	splitted := strings.SplitN(data, ",", -1)
	parsed := make([]string, len(splitted))
	for i, val := range splitted {
		parsed[i] = strings.TrimSpace(val)
	}
	return parsed
}

func awss3(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	rangeHeader := r.Header.Get("Range")

	// Strip the prefix, if it's present.
	if len(c.stripPath) > 0 {
		path = strings.TrimPrefix(path, c.stripPath)
	}

	// If there is a health check path defined, and if this path matches it,
	// then return 200 OK and return.
	// Note: we want to apply the health check *after* the prefix is stripped.
	if len(c.healthCheckPath) > 0 && path == c.healthCheckPath {
		w.WriteHeader(http.StatusOK)
		return
	}

	idx := strings.Index(path, "symlink.json")
	if idx > -1 {
		result, err := s3get(c.s3Bucket, c.s3KeyPrefix+path[:idx+12], rangeHeader)
		if err != nil {
			code, message := awsError(err)
			http.Error(w, message, code)
			return
		}
		var link symlink
		buf := new(bytes.Buffer)
		buf.ReadFrom(result.Body) // nolint
		err = json.Unmarshal(buf.Bytes(), &link)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		path = link.URL + path[idx+12:]
	}
	if strings.HasSuffix(path, "/") {
		if c.directoryListing {
			s3listFiles(w, r, c.s3Bucket, c.s3KeyPrefix+path)
			return
		}
		path += c.indexDocument
	}
	obj, err := s3get(c.s3Bucket, c.s3KeyPrefix+path, rangeHeader)
	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case s3.ErrCodeNoSuchKey:
			path += "/" + c.indexDocument
			obj, err = s3get(c.s3Bucket, c.s3KeyPrefix+path, rangeHeader)
		}
	}

	if err != nil {
		code, message := awsError(err)
		http.Error(w, message, code)
		return
	}
	setHeadersFromAwsResponse(w, obj)

	io.Copy(w, obj.Body) // nolint
}

func setHeadersFromAwsResponse(w http.ResponseWriter, obj *s3.GetObjectOutput) {

	// Cache-Control
	if len(c.httpCacheControl) > 0 {
		setStrHeader(w, "Cache-Control", &c.httpCacheControl)
	} else {
		setStrHeader(w, "Cache-Control", obj.CacheControl)
	}

	// Expires
	if len(c.httpExpires) > 0 {
		setStrHeader(w, "Expires", &c.httpExpires)
	} else {
		setStrHeader(w, "Expires", obj.Expires)
	}

	setStrHeader(w, "Content-Disposition", obj.ContentDisposition)
	setStrHeader(w, "Content-Encoding", obj.ContentEncoding)
	setStrHeader(w, "Content-Language", obj.ContentLanguage)
	setIntHeader(w, "Content-Length", obj.ContentLength)
	setStrHeader(w, "Content-Range", obj.ContentRange)
	setStrHeader(w, "Content-Type", obj.ContentType)
	setStrHeader(w, "ETag", obj.ETag)
	setTimeHeader(w, "Last-Modified", obj.LastModified)

	httpStatus := determineHTTPStatus(obj)

	w.WriteHeader(httpStatus)
}

func determineHTTPStatus(obj *s3.GetObjectOutput) int {

	httpStatus := http.StatusOK

	contentRangeIsGiven := obj.ContentRange != nil && len(*obj.ContentRange) > 0

	if contentRangeIsGiven {
		httpStatus = http.StatusPartialContent

		if totalFileSizeEqualToContentRange(obj) {
			httpStatus = http.StatusOK
		}

	}
	return httpStatus
}

func totalFileSizeEqualToContentRange(obj *s3.GetObjectOutput) bool {
	totalSizeIsEqualToContentRange := false
	if totalSize, err := strconv.ParseInt(getFileSizeAsString(obj), 10, 64); err == nil {
		if totalSize == (*obj.ContentLength) {
			totalSizeIsEqualToContentRange = true
		}
	}
	return totalSizeIsEqualToContentRange
}

/**
See https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Range
*/
func getFileSizeAsString(obj *s3.GetObjectOutput) string {
	s := strings.Split(*obj.ContentRange, "/")
	totalSizeString := s[1]
	totalSizeString = strings.TrimSpace(totalSizeString)
	return totalSizeString
}

func s3get(backet, key, rangeHeader string) (*s3.GetObjectOutput, error) {
	var rangeHeaderAwsString *string

	if len(rangeHeader) > 0 {
		rangeHeaderAwsString = aws.String(rangeHeader)
	}

	req := &s3.GetObjectInput{
		Bucket: aws.String(backet),
		Key:    aws.String(key),
		Range:  rangeHeaderAwsString,
	}
	return s3.New(awsSession()).GetObject(req)
}

func getObjsFromS3(req *s3.ListObjectsInput, key string) (*s3.ListObjectsOutput, error) {
	var result *s3.ListObjectsOutput
	var err error

	if c.allPagesInDir {
		result = &s3.ListObjectsOutput{
			CommonPrefixes: []*s3.CommonPrefix{},
			Contents:       []*s3.Object{},
			Prefix:         aws.String(key),
		}

		err = s3.New(awsSession()).ListObjectsPages(req,
			func(page *s3.ListObjectsOutput, lastPage bool) bool {
				result.CommonPrefixes = append(result.CommonPrefixes, page.CommonPrefixes...)
				result.Contents = append(result.Contents, page.Contents...)
				return len(page.CommonPrefixes) == 1000
			})
	} else {
		result, err = s3.New(awsSession()).ListObjects(req)
	}
	return result, err
}

func s3listFiles(w http.ResponseWriter, r *http.Request, backet, key string) {
	key = strings.TrimPrefix(key, "/")
	req := &s3.ListObjectsInput{
		Bucket:    aws.String(backet),
		Prefix:    aws.String(key),
		Delimiter: aws.String("/"),
	}

	result, err := getObjsFromS3(req, key)

	if err != nil {
		code, message := awsError(err)
		http.Error(w, message, code)
		return
	}
	candidates := map[string]bool{}
	updatedAt := map[string]time.Time{}
	for _, obj := range result.Contents {
		candidate := strings.TrimPrefix(aws.StringValue(obj.Key), key)
		if len(candidate) == 0 {
			continue
		}
		candidates[candidate] = true
		updatedAt[candidate] = *obj.LastModified
	}
	for _, obj := range result.CommonPrefixes {
		candidate := strings.TrimPrefix(aws.StringValue(obj.Prefix), key)
		if len(candidate) == 0 {
			continue
		}
		candidates[candidate] = true
	}
	files := []string{}
	for file := range candidates {
		files = append(files, file)
	}
	sort.Sort(objects(files))

	if strings.ToLower(c.dirListingFormat) == "html" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := "<!DOCTYPE html><html><body><ul>"
		for _, file := range files {
			html += "<li><a href=\"" + file + "\">" + file + "</a>"
			if len(updatedAt) != 0 {
				html += " " + updatedAt[file].String()
			}
			html += "</li>"
		}
		html += "</ul></body></html>"
		fmt.Fprintln(w, html)
		return
	}
	bytes, merr := json.Marshal(files)
	if merr != nil {
		http.Error(w, merr.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(w, string(bytes))
}

func awsSession() *session.Session {
	config := &aws.Config{
		Region: aws.String(c.awsRegion),
	}
	if len(c.awsAPIEndpoint) > 0 {
		config.Endpoint = aws.String(c.awsAPIEndpoint)
		config.S3ForcePathStyle = aws.Bool(true)
	}
	return session.Must(session.NewSession(config))
}

func setStrHeader(w http.ResponseWriter, key string, value *string) {
	if value != nil && len(*value) > 0 {
		w.Header().Add(key, *value)
	}
}

func setIntHeader(w http.ResponseWriter, key string, value *int64) {
	if value != nil && *value > 0 {
		w.Header().Add(key, strconv.FormatInt(*value, 10))
	}
}

func setTimeHeader(w http.ResponseWriter, key string, value *time.Time) {
	if value != nil && !reflect.DeepEqual(*value, time.Time{}) {
		w.Header().Add(key, value.UTC().Format(http.TimeFormat))
	}
}

func awsError(err error) (int, string) {
	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case s3.ErrCodeNoSuchBucket, s3.ErrCodeNoSuchKey:
			return http.StatusNotFound, aerr.Error()
		}
		return http.StatusInternalServerError, aerr.Error()
	}
	return http.StatusInternalServerError, err.Error()
}

type objects []string

func (s objects) Len() int {
	return len(s)
}
func (s objects) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s objects) Less(i, j int) bool {
	if strings.Contains(s[i], "/") {
		if !strings.Contains(s[j], "/") {
			return true
		}
	} else {
		if strings.Contains(s[j], "/") {
			return false
		}
	}
	irs := []rune(s[i])
	jrs := []rune(s[j])

	max := len(irs)
	if max > len(jrs) {
		max = len(jrs)
	}
	for idx := 0; idx < max; idx++ {
		ir := irs[idx]
		jr := jrs[idx]
		irl := unicode.ToLower(ir)
		jrl := unicode.ToLower(jr)

		if irl != jrl {
			return irl < jrl
		}
		if ir != jr {
			return ir < jr
		}
	}
	return false
}
