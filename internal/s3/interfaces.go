package s3

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
)

type Service interface {
	EnsureBucket(ctx context.Context, bucketName string) error
	GetObject(ctx context.Context, bucketName string, key string) (*minio.Object, error)
	StatObject(ctx context.Context, bucketName string, key string) (minio.ObjectInfo, error)
	RemoveObject(ctx context.Context, bucketName string, key string) error
	PutObject(ctx context.Context, bucketName string, key string, reader io.Reader, size int64, contentType string, metadata map[string]string) (minio.UploadInfo, error)
	SetBucketWebhookCreatedEvents(ctx context.Context) error
	PresignPost(ctx context.Context, key string, contentType string, size int64, metadata map[string]string, expires time.Duration) (string, map[string]string, error)
}
