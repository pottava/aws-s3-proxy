package service

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Client ...
var Client AWS

// AWS is a service to interact with original AWS services
type AWS interface {
	S3get(ctx context.Context, bucket, key string, rangeHeader *string) (*s3.GetObjectOutput, error)
	S3listObjects(bucket, prefix string) (*s3.ListObjectsOutput, error)
	S3put(bucket, key string, b []byte) (*s3manager.UploadOutput, error)
}

type client struct {
	context.Context
	*session.Session
}

// NewClient returns new AWS client
func NewClient(ctx context.Context, region *string) AWS {
	return client{Context: ctx, Session: awsSession(region)}
}

// InitClient ...
func InitClient(ctx context.Context, region *string) {
	Client = client{Context: ctx, Session: awsSession(region)}
}
