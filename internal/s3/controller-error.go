package s3

import (
	"log"
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

		log.Print("unknown s3 error")

		return http.StatusInternalServerError, aerr.Error()
	}

	log.Print("unknown http error")

	return http.StatusInternalServerError, err.Error()
}
