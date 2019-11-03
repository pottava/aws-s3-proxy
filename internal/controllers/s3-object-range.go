package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/service/s3"
)

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
	if len(s) > 1 {
		return strings.TrimSpace(s[1])
	}
	return ""
}
