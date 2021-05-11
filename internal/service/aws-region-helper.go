package service

import (
	"context"

	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/packethost/aws-s3-proxy/internal/config"
)

// GuessBucketRegion returns a region of the bucket
func GuessBucketRegion(bucket string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.Config.GuessBucketTimeout)
	defer cancel()

	return s3manager.GetBucketRegion(ctx, awsSession(nil), bucket, "us-east-1")
}
