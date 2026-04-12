// Package s3: presigned PUT/GET and delete against an S3-compatible API (e.g. Yandex Object Storage).
package s3

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	bucket     string
	presignTTL time.Duration
	api        *awss3.Client
	presign    *awss3.PresignClient
}

// redactAccessKeyID logs only a short prefix of the access key id (never the secret).
func redactAccessKeyID(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:4] + "…"
}

func NewClient(ctx context.Context, endpoint, bucket, region, accessKey, secretKey string, presignTTL time.Duration) (*Client, error) {
	if bucket == "" {
		return nil, fmt.Errorf("s3: bucket is required")
	}
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("s3: access key and secret key are required")
	}
	if presignTTL <= 0 {
		presignTTL = 15 * time.Minute
	}
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		slog.Error("s3: AWS SDK config load failed", "endpoint", endpoint, "region", region, "err", err)
		return nil, fmt.Errorf("s3: load aws config: %w", err)
	}
	api := awss3.NewFromConfig(cfg, func(o *awss3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true // path-style for Yandex Object Storage
	})
	slog.Info("s3: client initialized",
		"endpoint", endpoint,
		"bucket", bucket,
		"region", region,
		"presign_ttl", presignTTL.String(),
		"access_key_id_prefix", redactAccessKeyID(accessKey),
	)
	return &Client{
		bucket:     bucket,
		presignTTL: presignTTL,
		api:        api,
		presign:    awss3.NewPresignClient(api),
	}, nil
}

func (c *Client) PresignedUploadURL(ctx context.Context, s3Key, contentType string) (string, error) {
	in := &awss3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(s3Key),
	}
	if contentType != "" {
		in.ContentType = aws.String(contentType)
	}
	out, err := c.presign.PresignPutObject(ctx, in, func(opts *awss3.PresignOptions) {
		opts.Expires = c.presignTTL
	})
	if err != nil {
		slog.Error("s3: presign PUT failed", "bucket", c.bucket, "key", s3Key, "content_type", contentType, "err", err)
		return "", fmt.Errorf("s3: presign put: %w", err)
	}
	return out.URL, nil
}

func (c *Client) ReadURL(ctx context.Context, s3Key string) (string, error) {
	out, err := c.presign.PresignGetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(s3Key),
	}, func(opts *awss3.PresignOptions) {
		opts.Expires = c.presignTTL
	})
	if err != nil {
		slog.Error("s3: presign GET failed", "bucket", c.bucket, "key", s3Key, "err", err)
		return "", fmt.Errorf("s3: presign get: %w", err)
	}
	return out.URL, nil
}

func (c *Client) DeleteObject(ctx context.Context, s3Key string) error {
	_, err := c.api.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		slog.Error("s3: delete object failed", "bucket", c.bucket, "key", s3Key, "err", err)
		return fmt.Errorf("s3: delete object: %w", err)
	}
	return nil
}
