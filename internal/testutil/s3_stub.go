package testutil

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
)

type S3ServiceStubRemovedRecord struct {
	Bucket string
	Key    string
}

type S3ServiceStubPresignCall struct {
	Metadata    map[string]string
	Key         string
	ContentType string
	Size        int64
}

type S3ServiceStub struct {
	RemovedRecords []S3ServiceStubRemovedRecord
	PresignedCalls []S3ServiceStubPresignCall
}

func (s *S3ServiceStub) EnsureBucket(_ context.Context, _ string) error { return nil }
func (s *S3ServiceStub) GetObject(_ context.Context, _ string, _ string) (*minio.Object, error) {
	return nil, io.EOF
}

func (s *S3ServiceStub) StatObject(_ context.Context, _ string, _ string) (minio.ObjectInfo, error) {
	return minio.ObjectInfo{}, nil
}

func (s *S3ServiceStub) RemoveObject(_ context.Context, bucketName string, key string) error {
	s.RemovedRecords = append(s.RemovedRecords, S3ServiceStubRemovedRecord{Bucket: bucketName, Key: key})
	return nil
}

func (s *S3ServiceStub) PutObject(_ context.Context, _ string, _ string, _ io.Reader, _ int64, _ string, _ map[string]string) (minio.UploadInfo, error) {
	return minio.UploadInfo{}, nil
}
func (s *S3ServiceStub) SetBucketWebhookCreatedEvents(_ context.Context) error { return nil }
func (s *S3ServiceStub) PresignPost(_ context.Context, key string, contentType string, size int64, metadata map[string]string, _ time.Duration) (string, map[string]string, error) {
	s.PresignedCalls = append(s.PresignedCalls, S3ServiceStubPresignCall{Key: key, ContentType: contentType, Size: size, Metadata: metadata})
	return "/api/v1/files", map[string]string{"key": key}, nil
}
