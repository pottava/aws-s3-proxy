package service

import (
	"bytes"
	"context"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/packethost/aws-s3-proxy/internal/config"
)

// S3get returns a specified object from Amazon S3
func S3get(ctx context.Context, bucket, key string, rangeHeader *string) (*s3.GetObjectOutput, error) {
	req := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Range:  rangeHeader,
	}

	return s3.New(c.Session).GetObjectWithContext(ctx, req)
}

// S3put uploads a file to the bucket
func S3put(ctx context.Context, bucket, key string, b []byte) (*s3manager.UploadOutput, error) {
	up := &s3manager.UploadInput{
		Bucket:          aws.String(bucket),
		ContentEncoding: aws.String(http.DetectContentType(b)),
		ACL:             aws.String("public-read"),
		Key:             aws.String(key),
		Body:            bytes.NewReader(b),
	}

	return s3manager.NewUploader(c.Session).UploadWithContext(ctx, up)
}

// S3listObjects returns a list of s3 objects
func S3listObjects(ctx context.Context, bucket, prefix string) (*s3.ListObjectsOutput, error) {
	req := &s3.ListObjectsInput{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	}

	// List 1000 records
	if !config.Config.AllPagesInDir {
		return s3.New(c.Session).ListObjectsWithContext(ctx, req)
	}

	// List all objects with pagenation
	result := &s3.ListObjectsOutput{
		CommonPrefixes: []*s3.CommonPrefix{},
		Contents:       []*s3.Object{},
		Prefix:         aws.String(prefix),
	}

	err := s3.New(c.Session).ListObjectsPagesWithContext(ctx, req,
		func(page *s3.ListObjectsOutput, lastPage bool) bool {
			result.CommonPrefixes = append(result.CommonPrefixes, page.CommonPrefixes...)
			result.Contents = append(result.Contents, page.Contents...)
			return len(page.Contents) == 1000 // nolint:gomnd
		})

	return result, err
}
