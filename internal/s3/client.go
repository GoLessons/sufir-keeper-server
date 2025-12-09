package s3

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	minio     *minio.Client
	endpoint  string
	accessKey string
	secretKey string
	bucket    string
}

func NewClient(endpoint, accessKey, secretKey, bucket string) (*Client, error) {
	ep := strings.TrimSpace(endpoint)
	ak := strings.TrimSpace(accessKey)
	sk := strings.TrimSpace(secretKey)
	b := strings.TrimSpace(bucket)
	if ep == "" || ak == "" || sk == "" || b == "" {
		return nil, errors.New("invalid s3 config")
	}
	u, err := url.Parse(ep)
	if err != nil {
		return nil, err
	}

	secure := u.Scheme == "https"
	cli, err := minio.New(u.Host, &minio.Options{Creds: credentials.NewStaticV4(ak, sk, ""), Secure: secure})
	if err != nil {
		return nil, err
	}

	return &Client{endpoint: ep, accessKey: ak, secretKey: sk, bucket: b, minio: cli}, nil
}

func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.minio.BucketExists(ctx, c.bucket)
	if err != nil {
		return err
	}
	if !exists {
		if err := c.minio.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) GetObject(ctx context.Context, key string) (*minio.Object, error) {
	return c.minio.GetObject(ctx, c.bucket, strings.TrimSpace(key), minio.GetObjectOptions{})
}

func (c *Client) StatObject(ctx context.Context, key string) (minio.ObjectInfo, error) {
	return c.minio.StatObject(ctx, c.bucket, strings.TrimSpace(key), minio.StatObjectOptions{})
}

func (c *Client) RemoveObject(ctx context.Context, key string) error {
	return c.minio.RemoveObject(ctx, c.bucket, strings.TrimSpace(key), minio.RemoveObjectOptions{})
}

func (c *Client) PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string, metadata map[string]string) (minio.UploadInfo, error) {
	opts := minio.PutObjectOptions{ContentType: strings.TrimSpace(contentType)}
	if len(metadata) > 0 {
		opts.UserMetadata = metadata
	}
	return c.minio.PutObject(ctx, c.bucket, strings.TrimSpace(key), reader, size, opts)
}

// SetBucketWebhookCreatedEvents intentionally removed: bucket notifications are configured by container init.
