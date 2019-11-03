package common

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// GuessBucketRegion returns a region of the bucket
func GuessBucketRegion(ctx context.Context, bucket string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return s3manager.GetBucketRegion(ctx, AwsSession(nil), bucket, "us-east-1")
}
