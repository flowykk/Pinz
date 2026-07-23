package s3

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

const (
	defaultYandexEndpoint = "https://storage.yandexcloud.net"
	defaultRegion = "ru-central1"
	defaultPresignTTL = 15 * time.Minute
)

// NewFromEnv loads S3 config from env; (nil, nil) if S3_BUCKET is unset.
// Needs S3_ACCESS_KEY and S3_SECRET_KEY when bucket is set. Optional: S3_ENDPOINT, S3_REGION, S3_PRESIGN_TTL.
func NewFromEnv(ctx context.Context) (*Client, error) {
	bucket := strings.TrimSpace(os.Getenv("S3_BUCKET"))
	if bucket == "" {
		return nil, nil
	}
	accessKey := strings.TrimSpace(os.Getenv("S3_ACCESS_KEY"))
	secretKey := strings.TrimSpace(os.Getenv("S3_SECRET_KEY"))
	if accessKey == "" || secretKey == "" {
		slog.Error("s3: S3_BUCKET is set but credentials are missing", "bucket", bucket)
		return nil, fmt.Errorf("S3_BUCKET is set but S3_ACCESS_KEY or S3_SECRET_KEY is empty")
	}
	endpoint := strings.TrimSpace(os.Getenv("S3_ENDPOINT"))
	if endpoint == "" {
		endpoint = defaultYandexEndpoint
	}
	region := strings.TrimSpace(os.Getenv("S3_REGION"))
	if region == "" {
		region = defaultRegion
	}
	ttl := defaultPresignTTL
	if v := strings.TrimSpace(os.Getenv("S3_PRESIGN_TTL")); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			slog.Error("s3: invalid S3_PRESIGN_TTL", "value", v, "err", err)
			return nil, fmt.Errorf("S3_PRESIGN_TTL: %w", err)
		}
		ttl = d
	}
	client, err := NewClient(ctx, endpoint, bucket, region, accessKey, secretKey, ttl)
	if err != nil {
		slog.Error("s3: failed to create client from environment", "endpoint", endpoint, "bucket", bucket, "region", region, "err", err)
	}
	return client, err
}
