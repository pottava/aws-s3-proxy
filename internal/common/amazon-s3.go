package common

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pottava/aws-s3-proxy/internal/config"
)

// S3get returns a specified object from Amazon S3
func S3get(ctx context.Context, sess *session.Session, bucket, key string, rangeHeader *string,
) (*s3.GetObjectOutput, error) {
	req := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Range:  rangeHeader,
	}
	return s3.New(sess).GetObjectWithContext(ctx, req)
}

// S3listObjects returns a list of s3 objects
func S3listObjects(ctx context.Context, sess *session.Session, bucket, prefix string,
) (*s3.ListObjectsOutput, error) {
	req := &s3.ListObjectsInput{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	}
	// List 1000 records
	if !config.Config.AllPagesInDir {
		return s3.New(sess).ListObjects(req)
	}
	// List all objects with pagenation
	result := &s3.ListObjectsOutput{
		CommonPrefixes: []*s3.CommonPrefix{},
		Contents:       []*s3.Object{},
		Prefix:         aws.String(prefix),
	}
	err := s3.New(sess).ListObjectsPages(req,
		func(page *s3.ListObjectsOutput, lastPage bool) bool {
			result.CommonPrefixes = append(result.CommonPrefixes, page.CommonPrefixes...)
			result.Contents = append(result.Contents, page.Contents...)
			return len(page.CommonPrefixes) == 1000
		})
	return result, err
}
