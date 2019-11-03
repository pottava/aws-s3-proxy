package controllers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/assert"
)

func TestToHTTPError(t *testing.T) {
	expectedCode := http.StatusInternalServerError
	expectedMsg := "test"

	code, msg := toHTTPError(errors.New(expectedMsg))

	assert.Equal(t, expectedCode, code)
	assert.Equal(t, expectedMsg, msg)
}

func TestToHTTPNoSuchBucketError(t *testing.T) {
	expectedCode := http.StatusNotFound
	expectedMsg := "NoSuchBucket: 2\ncaused by: 1"

	code, msg := toHTTPError(awserr.New(
		s3.ErrCodeNoSuchBucket,
		"2",
		errors.New("1"),
	))
	assert.Equal(t, expectedCode, code)
	assert.Equal(t, expectedMsg, msg)
}

func TestToHTTPNoSuchKeyError(t *testing.T) {
	expectedCode := http.StatusNotFound
	expectedMsg := "NoSuchKey: 2\ncaused by: 1"

	code, msg := toHTTPError(awserr.New(
		s3.ErrCodeNoSuchKey,
		"2",
		errors.New("1"),
	))
	assert.Equal(t, expectedCode, code)
	assert.Equal(t, expectedMsg, msg)
}

func TestToHTTPNoSuchUploadError(t *testing.T) {
	expectedCode := http.StatusInternalServerError
	expectedMsg := "NoSuchUpload: 2\ncaused by: 1"

	code, msg := toHTTPError(awserr.New(
		s3.ErrCodeNoSuchUpload,
		"2",
		errors.New("1"),
	))
	assert.Equal(t, expectedCode, code)
	assert.Equal(t, expectedMsg, msg)
}
