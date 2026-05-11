package repositories

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	pinzotel "pinz/backend/pkg/otel"
)

type StreamQueryer struct {
	Client *redis.Client
}

func (s StreamQueryer) XLen(ctx context.Context, stream string) (int64, error) {
	if s.Client == nil {
		return 0, nil
	}
	return s.Client.XLen(ctx, stream).Result()
}

func (s StreamQueryer) XPending(ctx context.Context, stream, group string) (pinzotel.StreamPending, error) {
	if s.Client == nil {
		return pinzotel.StreamPending{}, nil
	}
	p, err := s.Client.XPending(ctx, stream, group).Result()
	if err != nil {
		if strings.Contains(err.Error(), "NOGROUP") {
			return pinzotel.StreamPending{}, nil
		}
		return pinzotel.StreamPending{}, err
	}
	if p == nil {
		return pinzotel.StreamPending{}, nil
	}
	out := pinzotel.StreamPending{Count: p.Count}
	if p.Count == 0 {
		return out, nil
	}
	ext, err := s.Client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: stream, Group: group, Start: "-", End: "+", Count: 1,
	}).Result()
	if err == nil && len(ext) > 0 {
		out.MaxIdle = time.Duration(ext[0].Idle) * time.Millisecond
	}
	return out, nil
}
