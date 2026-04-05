package services

import "context"

// MediaURLResolver presigned PUT/GET and delete for object storage.
type MediaURLResolver interface {
	PresignedUploadURL(ctx context.Context, s3Key, contentType string) (string, error)
	ReadURL(ctx context.Context, s3Key string) (string, error)
	DeleteObject(ctx context.Context, s3Key string) error
}
