package services

import (
	"context"
	"time"
)

// MediaURLResolver presigned PUT/GET and delete for object storage.
type MediaURLResolver interface {
	PresignedUploadURL(ctx context.Context, s3Key, contentType string) (string, error)
	ReadURL(ctx context.Context, s3Key string) (string, error)
	// ttl<=0 → дефолтный TTL клиента.
	ReadURLWithTTL(ctx context.Context, s3Key string, ttl time.Duration) (string, error)
	DeleteObject(ctx context.Context, s3Key string) error
}
