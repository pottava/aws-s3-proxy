package s3

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/packethost/aws-s3-proxy/internal/config"
)

// Download wraps the  AWS s3 Get object
type Download struct {
	Output *s3.GetObjectOutput
}

// Upload wraps the s3 upload object
type Upload struct {
	Output *s3manager.UploadOutput
}

// Get returns a specified object from Amazon S3
func get(ctx context.Context, bucket *config.Bucket, key, rangeHeader *string) (*Download, error) {
	if bucket.Session == nil {
		config.Logger.Panic("bad s3 client")
	}

	req := &s3.GetObjectInput{
		Bucket: &bucket.Bucket,
		Key:    key,
		Range:  rangeHeader,
	}

	c := s3.New(bucket.Session)
	get, err := c.GetObjectWithContext(ctx, req)

	return &Download{
		Output: get,
	}, err
}

// Put uploads a file to the bucket
func put(ctx context.Context, bucket *config.Bucket, key *string, r io.Reader) (*Upload, error) {
	up := &s3manager.UploadInput{
		Bucket: &bucket.Bucket,
		ACL:    aws.String("public-read"),
		Key:    key,
		Body:   r,
	}

	put, err := s3manager.NewUploader(bucket.Session).UploadWithContext(ctx, up)

	return &Upload{
		Output: put,
	}, err
}
