package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/go-openapi/swag"
	"github.com/labstack/echo/v4"

	"github.com/packethost/aws-s3-proxy/internal/config"
	"github.com/packethost/aws-s3-proxy/internal/service"
)

// AwsS3Get handles download requests
func AwsS3Get(e echo.Context) error {
	c := config.Config
	req := e.Request()
	res := e.Response()

	// Strip the prefix, if it's present.
	path := req.URL.Path
	if len(c.StripPath) > 0 {
		path = strings.TrimPrefix(path, c.StripPath)
	}

	// If there is a health check path defined, and if this path matches it,
	// then return 200 OK and return.
	// Note: we want to apply the health check *after* the prefix is stripped.
	if len(c.HealthCheckPath) > 0 && path == c.HealthCheckPath {
		res.WriteHeader(http.StatusOK)
		return nil
	}
	// Range header
	var rangeHeader *string
	if candidate := req.Header.Get("Range"); !swag.IsZero(candidate) {
		rangeHeader = aws.String(candidate)
	}

	// client := service.NewClient(r.Context(), aws.String(config.Config.AwsRegion))

	// Replace path with symlink.json
	idx := strings.Index(path, "symlink.json")
	if idx > -1 {
		replaced, err := replacePathWithSymlink(req.Context(), c.S3Bucket, c.S3KeyPrefix+path[:idx+12])
		if err != nil {
			e.Error(err)

			return err
		}

		path = aws.StringValue(replaced) + path[idx+12:]
	}

	// Ends with / -> listing or index.html
	if strings.HasSuffix(path, "/") {
		if c.DirectoryListing {
			return s3listFiles(e, c.S3Bucket, c.S3KeyPrefix+path)
		}

		path += c.IndexDocument
	}
	// Get a S3 object
	obj, err := service.S3get(req.Context(), c.S3Bucket, c.S3KeyPrefix+path, rangeHeader)
	if err != nil {
		e.Error(err)

		return err
	}

	setHeadersFromAwsResponse(res, obj, c.HTTPCacheControl, c.HTTPExpires)

	return e.Stream(http.StatusOK, echo.MIMEOctetStream, obj.Body)
}

// AwsS3Put handles upload requests
func AwsS3Put(e echo.Context) error {
	c := config.Config
	req := e.Request()
	res := e.Response()

	// Strip the prefix, if it's present.
	path := req.URL.Path
	if len(c.StripPath) > 0 {
		path = strings.TrimPrefix(path, c.StripPath)
	}

	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		e.Error(err)
		return err
	}
	defer req.Body.Close()
	// Put a S3 object
	obj, err := service.S3put(req.Context(), c.S3Bucket, c.S3KeyPrefix+path, b)
	if err != nil {
		e.Error(err)

		return err
	}

	res.WriteHeader(http.StatusAccepted)
	setStrHeader(res, "ETag", obj.ETag)
	setStrHeader(res, "VersionID", obj.VersionID)
	setStrHeader(res, "UploadID", &obj.UploadID)
	setStrHeader(res, "Location", &obj.Location)

	return nil
}

func replacePathWithSymlink(ctx context.Context, bucket, symlinkPath string) (*string, error) {
	obj, err := service.S3get(ctx, bucket, symlinkPath, nil)
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

func s3listFiles(e echo.Context, bucket, prefix string) error {
	prefix = strings.TrimPrefix(prefix, "/")

	result, err := service.S3listObjects(e.Request().Context(), bucket, prefix)
	if err != nil {
		e.Error(err)

		return err
	}

	files, _ := convertToMaps(result, prefix)

	// Output as a HTML
	if strings.EqualFold(config.Config.DirListingFormat, "html") {
		return e.HTML(http.StatusOK, strings.Join(files, "\n"))
	}

	// Output as a JSON
	bytes, err := json.Marshal(files)
	if err != nil {
		e.Error(err)
		return err
	}

	e.JSONBlob(http.StatusOK, bytes)

	return nil
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
