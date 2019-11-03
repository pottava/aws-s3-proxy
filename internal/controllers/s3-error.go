package controllers

import (
	"net/http"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
)

func toHTTPError(err error) (int, string) {
	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case s3.ErrCodeNoSuchBucket, s3.ErrCodeNoSuchKey:
			return http.StatusNotFound, aerr.Error()
		}
		return http.StatusInternalServerError, aerr.Error()
	}
	return http.StatusInternalServerError, err.Error()
}
