package service

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// GuessBucketRegion returns a region of the bucket
func GuessBucketRegion(bucket string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s3manager.GetBucketRegion(ctx, awsSession(nil), bucket, "us-east-1")
}
