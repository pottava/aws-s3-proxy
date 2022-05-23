package service

import (
	"context"
	"net/http"

	"runtime"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

// DefaultDownloadPartSize is the default range of bytes to get at a time when
// using Download().
const DefaultDownloadPartSize = 1024 * 1024 * 50

// DefaultDownloadConcurrency is the default number of goroutines to spin up
// when using Download().

var DefaultDownloadConcurrency = runtime.NumCPU()

// AWS is a service to interact with original AWS services
type AWS interface {
	S3get(bucket, key string, rangeHeader *string) (*s3.GetObjectOutput, error)
	S3Download(f http.ResponseWriter, bucket, key string, rangeHeader *string) error
	S3listObjects(bucket, prefix string) (*s3.ListObjectsOutput, error)
}

type client struct {
	context.Context
	// *session.Session
	S3          s3iface.S3API
	PartSize    int64
	Concurrency int
}

// NewClient returns new AWS client
func NewClient(ctx context.Context, region *string, partSize int64, concurrency int) AWS {
	return &client{Context: ctx,
		S3:          s3.New(awsSession(region)),
		PartSize:    partSize,
		Concurrency: concurrency,
	}
}
