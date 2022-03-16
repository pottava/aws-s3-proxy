package s3

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/labstack/echo/v4"

	"github.com/packethost/aws-s3-proxy/internal/config"
)

var getStatus *regexp.Regexp

func init() {
	getStatus = regexp.MustCompile(`status code: (?P<status>\d\d\d),`)
}

func storeObject(e echo.Context, r io.Reader, path *string) error {
	c := config.Cfg

	// tee the stream
	var buf bytes.Buffer
	tee := io.TeeReader(r, &buf)

	err := e.Stream(http.StatusOK, echo.MIMEOctetStream, tee)
	if err != nil {
		e.Error(err)

		return err
	}

	c.Logger.Debugf("writing object %s to local cache", *path)

	_, err = put(context.Background(), &c.PrimaryStore, path, bytes.NewReader(buf.Bytes()))
	if err != nil {
		c.Logger.Error("read through cache save failed")

		return err
	}

	return nil
}

func trySecondary(e echo.Context) error {
	c := config.Cfg
	h := c.HTTPOpts
	req := e.Request()
	res := e.Response()
	path := &req.URL.Path

	// Range header
	var rangeHeader *string
	if candidate := req.Header.Get("Range"); candidate != "" {
		rangeHeader = &candidate
	}

	get, err := get(req.Context(), &c.SecondaryStore, path, rangeHeader)
	if err != nil {
		status := http.StatusInternalServerError

		if getStatus.Match([]byte(err.Error())) {
			statusStr := getStatus.FindStringSubmatch(err.Error())

			status, _ = strconv.Atoi(statusStr[1])
		}

		return e.String(status, err.Error())
	}

	// stream object to client
	setHeadersFromAwsResponse(res, get, h.HTTPCacheControl, h.HTTPExpires)

	if c.ReadThrough.CacheToPrimary {
		return storeObject(e, get.Output.Body, path)
	}

	return e.Stream(http.StatusOK, echo.MIMEOctetStream, get.Output.Body)
}

// AwsS3Get handles download requests
func AwsS3Get(e echo.Context) error {
	c := config.Cfg
	h := c.HTTPOpts
	req := e.Request()
	res := e.Response()
	path := &req.URL.Path

	// Range header
	var rangeHeader *string
	if candidate := req.Header.Get("Range"); candidate != "" {
		rangeHeader = &candidate
	}

	get, err := get(req.Context(), &c.PrimaryStore, path, rangeHeader)
	if err != nil {
		if c.ReadThrough.Enabled {
			e.Logger().Warn("err in primary, trying secondary")
			return trySecondary(e)
		}

		status := http.StatusInternalServerError

		if getStatus.Match([]byte(err.Error())) {
			statusStr := getStatus.FindStringSubmatch(err.Error())

			status, _ = strconv.Atoi(statusStr[1])
		}

		e.Logger().Errorf("status %d", status)

		return e.String(status, err.Error())
	}

	setHeadersFromAwsResponse(res, get, h.HTTPCacheControl, h.HTTPExpires)

	return e.Stream(http.StatusOK, echo.MIMEOctetStream, get.Output.Body)
}

// AwsS3Put handles upload requests
func AwsS3Put(e echo.Context) error {
	c := config.Cfg
	req := e.Request()
	res := e.Response()
	path := &req.URL.Path

	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		e.Error(err)
		return err
	}
	defer req.Body.Close()
	// Put a S3 object
	put, err := put(req.Context(), &c.PrimaryStore, path, bytes.NewReader(b))
	if err != nil {
		e.Error(err)

		return err
	}

	o := put.Output

	res.WriteHeader(http.StatusAccepted)
	setStrHeader(res, "ETag", o.ETag)
	setStrHeader(res, "VersionID", o.VersionID)
	setStrHeader(res, "UploadID", &o.UploadID)
	setStrHeader(res, "Location", &o.Location)

	return nil
}

func setHeadersFromAwsResponse(w http.ResponseWriter, obj *Download, httpCacheControl, httpExpires string) {
	s := obj.Output

	// Cache-Control
	if len(httpCacheControl) > 0 {
		setStrHeader(w, "Cache-Control", &httpCacheControl)
	} else {
		setStrHeader(w, "Cache-Control", s.CacheControl)
	}

	// Expires
	if len(httpExpires) > 0 {
		setStrHeader(w, "Expires", &httpExpires)
	} else {
		setStrHeader(w, "Expires", s.Expires)
	}

	setStrHeader(w, "Content-Disposition", s.ContentDisposition)
	setStrHeader(w, "Content-Encoding", s.ContentEncoding)
	setStrHeader(w, "Content-Language", s.ContentLanguage)

	// Fix https://github.com/pottava/aws-s3-proxy/issues/20
	if len(w.Header().Get("Content-Encoding")) == 0 {
		setIntHeader(w, "Content-Length", s.ContentLength)
	}

	setStrHeader(w, "Content-Range", s.ContentRange)
	setStrHeader(w, "Content-Type", s.ContentType)
	setStrHeader(w, "ETag", s.ETag)
	setTimeHeader(w, "Last-Modified", s.LastModified)

	w.WriteHeader(determineHTTPStatus(s))
}

func setStrHeader(w http.ResponseWriter, key string, value *string) {
	if value != nil {
		if len(*value) > 0 {
			w.Header().Add(key, *value)
		}
	}
}

func setIntHeader(w http.ResponseWriter, key string, value *int64) {
	if value != nil {
		if *value > 0 {
			w.Header().Add(key, strconv.FormatInt(*value, 10)) // nolint: gomnd
		}
	}
}

func setTimeHeader(w http.ResponseWriter, key string, value *time.Time) {
	if value != nil {
		if !reflect.DeepEqual(*value, time.Time{}) {
			w.Header().Add(key, value.UTC().Format(http.TimeFormat))
		}
	}
}

func determineHTTPStatus(obj *s3.GetObjectOutput) int {
	if obj.ContentRange != nil && len(*obj.ContentRange) > 0 {
		if !totalFileSizeEqualToContentRange(obj) {
			return http.StatusPartialContent
		}
	}

	return http.StatusOK
}

func totalFileSizeEqualToContentRange(obj *s3.GetObjectOutput) bool {
	totalSizeIsEqualToContentRange := false

	if totalSize, err := strconv.ParseInt(getFileSizeAsString(obj), 10, 64); err == nil { // nolint
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
	if len(s) > 1 {
		return strings.TrimSpace(s[1])
	}

	return ""
}
