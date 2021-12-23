package service

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/session"
)

var c *client

type client struct {
	context.Context
	*session.Session
}

// InitAWSClient configures the AWS client for the service
func InitAWSClient(ctx context.Context, region *string) {
	c = &client{Context: ctx, Session: awsSession(region)}
}
