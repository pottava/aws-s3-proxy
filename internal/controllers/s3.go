package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/go-openapi/swag"
	"github.com/pottava/aws-s3-proxy/internal/config"
	"github.com/pottava/aws-s3-proxy/internal/service"
)

// AwsS3 handles requests for Amazon S3
func AwsS3(w http.ResponseWriter, r *http.Request) {
	c := config.Config

	// Strip the prefix, if it's present.
	path := r.URL.Path
	if len(c.StripPath) > 0 {
		path = strings.TrimPrefix(path, c.StripPath)
	}

	// If there is a health check path defined, and if this path matches it,
	// then return 200 OK and return.
	// Note: we want to apply the health check *after* the prefix is stripped.
	if len(c.HealthCheckPath) > 0 && path == c.HealthCheckPath {
		w.WriteHeader(http.StatusOK)
		return
	}
	// Range header
	var rangeHeader *string
	if candidate := r.Header.Get("Range"); !swag.IsZero(candidate) {
		rangeHeader = aws.String(candidate)
	}

	client := service.NewClient(r.Context(), aws.String(config.Config.AwsRegion))

	// Replace path with symlink.json
	idx := strings.Index(path, "symlink.json")
	if idx > -1 {
		replaced, err := replacePathWithSymlink(client, c.S3Bucket, c.S3KeyPrefix+path[:idx+12])
		if err != nil {
			code, message := toHTTPError(err)
			http.Error(w, message, code)
			return
		}
		path = aws.StringValue(replaced) + path[idx+12:]
	}
	// Ends with / -> listing or index.html
	if strings.HasSuffix(path, "/") {
		if c.DirectoryListing {
			s3listFiles(w, r, client, c.S3Bucket, c.S3KeyPrefix+path)
			return
		}
		path += c.IndexDocument
	}
	// Get a S3 object
	obj, err := client.S3get(c.S3Bucket, c.S3KeyPrefix+path, rangeHeader)
	if err != nil {
		code, message := toHTTPError(err)
		http.Error(w, message, code)
		return
	}
	setHeadersFromAwsResponse(w, obj, c.HTTPCacheControl, c.HTTPExpires)

	io.Copy(w, obj.Body) // nolint
}

func replacePathWithSymlink(client service.AWS, bucket, symlinkPath string) (*string, error) {
	obj, err := client.S3get(bucket, symlinkPath, nil)
	if err != nil {
		return nil, err
	}
	link := struct {
		URL string
	}{}
	buf := new(bytes.Buffer)
	if _, err = buf.ReadFrom(obj.Body); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(buf.Bytes(), &link); err != nil {
		return nil, err
	}
	return aws.String(link.URL), nil
}

func setHeadersFromAwsResponse(w http.ResponseWriter, obj *s3.GetObjectOutput, httpCacheControl, httpExpires string) {

	// Cache-Control
	if len(httpCacheControl) > 0 {
		setStrHeader(w, "Cache-Control", &httpCacheControl)
	} else {
		setStrHeader(w, "Cache-Control", obj.CacheControl)
	}
	// Expires
	if len(httpExpires) > 0 {
		setStrHeader(w, "Expires", &httpExpires)
	} else {
		setStrHeader(w, "Expires", obj.Expires)
	}
	setStrHeader(w, "Content-Disposition", obj.ContentDisposition)
	setStrHeader(w, "Content-Encoding", obj.ContentEncoding)
	setStrHeader(w, "Content-Language", obj.ContentLanguage)

	// Fix https://github.com/pottava/aws-s3-proxy/issues/20
	if len(w.Header().Get("Content-Encoding")) == 0 {
		setIntHeader(w, "Content-Length", obj.ContentLength)
	}
	setStrHeader(w, "Content-Range", obj.ContentRange)
	setStrHeader(w, "Content-Type", obj.ContentType)
	setStrHeader(w, "ETag", obj.ETag)
	setTimeHeader(w, "Last-Modified", obj.LastModified)

	w.WriteHeader(determineHTTPStatus(obj))
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

func s3listFiles(w http.ResponseWriter, r *http.Request, client service.AWS, bucket, prefix string) {
	prefix = strings.TrimPrefix(prefix, "/")

	result, err := client.S3listObjects(bucket, prefix)
	if err != nil {
		code, message := toHTTPError(err)
		http.Error(w, message, code)
		return
	}
	files, updatedAt := convertToMaps(result, prefix)

	// Output as a HTML
	if strings.EqualFold(config.Config.DirListingFormat, "html") {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintln(w, toHTML(files, updatedAt))
		return
	}
	// Output as a JSON
	bytes, merr := json.Marshal(files)
	if merr != nil {
		http.Error(w, merr.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(w, string(bytes))
}

func convertToMaps(s3output *s3.ListObjectsOutput, prefix string) ([]string, map[string]time.Time) {
	candidates := map[string]bool{}
	updatedAt := map[string]time.Time{}

	// Prefixes
	for _, obj := range s3output.CommonPrefixes {
		candidate := strings.TrimPrefix(aws.StringValue(obj.Prefix), prefix)
		if len(candidate) == 0 {
			continue
		}
		candidates[candidate] = true
	}
	// Contents
	for _, obj := range s3output.Contents {
		candidate := strings.TrimPrefix(aws.StringValue(obj.Key), prefix)
		if len(candidate) == 0 {
			continue
		}
		candidates[candidate] = true
		updatedAt[candidate] = *obj.LastModified
	}
	// Sort file names
	files := []string{}
	for file := range candidates {
		files = append(files, file)
	}
	sort.Sort(s3objects(files))

	return files, updatedAt
}

func toHTML(files []string, updatedAt map[string]time.Time) string {
	html := "<!DOCTYPE html><html><body><ul>"
	for _, file := range files {
		html += "<li><a href=\"" + file + "\">" + file + "</a>"
		if timestamp, ok := updatedAt[file]; ok {
			html += " " + timestamp.Format(time.RFC3339)
		}
		html += "</li>"
	}
	return html + "</ul></body></html>"
}
