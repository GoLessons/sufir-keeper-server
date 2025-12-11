package s3

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/notification"
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

func (c *Client) EnsureBucket(ctx context.Context, bucketName string) error {
	if bucketName == "" {
		bucketName = c.bucket
	}
	exists, err := c.minio.BucketExists(ctx, bucketName)
	if err != nil {
		return err
	}
	if !exists {
		if err := c.minio.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) GetObject(ctx context.Context, bucketName string, key string) (*minio.Object, error) {
	if bucketName == "" {
		bucketName = c.bucket
	}
	return c.minio.GetObject(ctx, bucketName, strings.TrimSpace(key), minio.GetObjectOptions{})
}

func (c *Client) StatObject(ctx context.Context, bucketName string, key string) (minio.ObjectInfo, error) {
	if bucketName == "" {
		bucketName = c.bucket
	}
	return c.minio.StatObject(ctx, bucketName, strings.TrimSpace(key), minio.StatObjectOptions{})
}

func (c *Client) RemoveObject(ctx context.Context, bucketName string, key string) error {
	if bucketName == "" {
		bucketName = c.bucket
	}
	return c.minio.RemoveObject(ctx, bucketName, strings.TrimSpace(key), minio.RemoveObjectOptions{})
}

func (c *Client) PutObject(ctx context.Context, bucketName string, key string, reader io.Reader, size int64, contentType string, metadata map[string]string) (minio.UploadInfo, error) {
	if bucketName == "" {
		bucketName = c.bucket
	}
	opts := minio.PutObjectOptions{ContentType: strings.TrimSpace(contentType)}
	if len(metadata) > 0 {
		opts.UserMetadata = metadata
	}
	return c.minio.PutObject(ctx, bucketName, strings.TrimSpace(key), reader, size, opts)
}

func (c *Client) SetBucketWebhookCreatedEvents(ctx context.Context) error {
	arn := notification.NewArn("minio", "sqs", "", "1", "webhook")
	qcfg := notification.NewConfig(arn)
	qcfg.AddEvents(
		notification.EventType("s3:ObjectCreated:Put"),
		notification.EventType("s3:ObjectCreated:Post"),
		notification.EventType("s3:ObjectCreated:Copy"),
		notification.EventType("s3:ObjectCreated:CompleteMultipartUpload"),
	)
	cfg := notification.Configuration{}
	cfg.AddQueue(qcfg)
	return c.minio.SetBucketNotification(ctx, c.bucket, cfg)
}

func (c *Client) PresignPost(ctx context.Context, key string, contentType string, size int64, metadata map[string]string, expires time.Duration) (string, map[string]string, error) {
	pol := minio.NewPostPolicy()
	if err := pol.SetBucket(c.bucket); err != nil {
		return "", nil, err
	}
	if err := pol.SetKey(strings.TrimSpace(key)); err != nil {
		return "", nil, err
	}
	if err := pol.SetExpires(time.Now().UTC().Add(expires)); err != nil {
		return "", nil, err
	}
	ct := strings.TrimSpace(contentType)
	if ct != "" {
		if err := pol.SetContentType(ct); err != nil {
			return "", nil, err
		}
	}
	if size > 0 {
		if err := pol.SetContentLengthRange(1, size); err != nil {
			return "", nil, err
		}
	}
	if len(metadata) > 0 {
		for k, v := range metadata {
			if err := pol.SetUserMetadata(strings.TrimSpace(k), strings.TrimSpace(v)); err != nil {
				return "", nil, err
			}
		}
	}
	url, formData, err := c.minio.PresignedPostPolicy(ctx, pol)
	if err != nil {
		return "", nil, err
	}
	return url.String(), formData, nil
}
