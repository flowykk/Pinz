package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const recommendationSnapshotKeyPrefix = "recsnapshot:"

type RecommendationSnapshot struct {
	UserID     string   `json:"user_id"`
	RegionID   int      `json:"region_id"`
	RegionName string   `json:"region_name"`
	RegionType string   `json:"region_type"`
	Category   string   `json:"category"`
	Season     string   `json:"season"`
	PinIDs     []string `json:"pin_ids"`
	CreatedAt  int64    `json:"created_at"`
}

type RecommendationSnapshotRepository struct {
	client *redis.Client
}

func NewRecommendationSnapshotRepository(client *redis.Client) *RecommendationSnapshotRepository {
	return &RecommendationSnapshotRepository{client: client}
}

func (r *RecommendationSnapshotRepository) Save(ctx context.Context, token string, snap *RecommendationSnapshot, ttl time.Duration) error {
	if r == nil || r.client == nil {
		return nil
	}
	data, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, recommendationSnapshotKeyPrefix+token, data, ttl).Err()
}

func (r *RecommendationSnapshotRepository) Get(ctx context.Context, token string) (*RecommendationSnapshot, bool, error) {
	if r == nil || r.client == nil {
		return nil, false, nil
	}
	raw, err := r.client.Get(ctx, recommendationSnapshotKeyPrefix+token).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var snap RecommendationSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return nil, false, err
	}
	return &snap, true, nil
}

func (r *RecommendationSnapshotRepository) Delete(ctx context.Context, token string) error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Del(ctx, recommendationSnapshotKeyPrefix+token).Err()
}
