package s3

import (
	"context"
	"time"
)

type Client struct {
	endpoint  string
	accessKey string
	secretKey string
	bucket    string
}

func NewClient(endpoint, accessKey, secretKey, bucket string) (*Client, error) {
	return &Client{endpoint: endpoint, accessKey: accessKey, secretKey: secretKey, bucket: bucket}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	_ = ctx
	time.Sleep(1 * time.Millisecond)
	return nil
}
