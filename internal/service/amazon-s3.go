package service

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pottava/aws-s3-proxy/internal/config"
)

// S3get returns a specified object from Amazon S3
func (c client) S3get(bucket, key string, rangeHeader *string) (*s3.GetObjectOutput, error) {
	req := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Range:  rangeHeader,
	}
	return s3.New(c.Session).GetObjectWithContext(c.Context, req)
}

// S3listObjects returns a list of s3 objects
func (c client) S3listObjects(bucket, prefix string) (*s3.ListObjectsOutput, error) {
	req := &s3.ListObjectsInput{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	}
	// List 1000 records
	if !config.Config.AllPagesInDir {
		return s3.New(c.Session).ListObjectsWithContext(c.Context, req)
	}
	// List all objects with pagenation
	result := &s3.ListObjectsOutput{
		CommonPrefixes: []*s3.CommonPrefix{},
		Contents:       []*s3.Object{},
		Prefix:         aws.String(prefix),
	}
	err := s3.New(c.Session).ListObjectsPagesWithContext(c.Context, req,
		func(page *s3.ListObjectsOutput, lastPage bool) bool {
			result.CommonPrefixes = append(result.CommonPrefixes, page.CommonPrefixes...)
			result.Contents = append(result.Contents, page.Contents...)
			return len(page.Contents) == 1000
		})
	return result, err
}
