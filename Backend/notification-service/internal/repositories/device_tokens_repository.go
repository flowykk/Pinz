package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"pinz/backend/notification-service/internal/db/sqlcdb"
	"pinz/backend/notification-service/internal/models"
)

type DeviceTokensRepository struct {
	q *sqlcdb.Queries
}

func NewDeviceTokensRepository(db *sql.DB) *DeviceTokensRepository {
	return &DeviceTokensRepository{q: sqlcdb.New(db)}
}

func (r *DeviceTokensRepository) Upsert(ctx context.Context, userID, apnsToken string) (string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("parse user_id: %w", err)
	}
	id, err := r.q.DeviceTokenUpsert(ctx, sqlcdb.DeviceTokenUpsertParams{UserID: uid, ApnsToken: apnsToken})
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func (r *DeviceTokensRepository) Delete(ctx context.Context, userID, apnsToken string) (int64, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return 0, fmt.Errorf("parse user_id: %w", err)
	}
	return r.q.DeviceTokenDelete(ctx, sqlcdb.DeviceTokenDeleteParams{UserID: uid, ApnsToken: apnsToken})
}

func (r *DeviceTokensRepository) ListByUser(ctx context.Context, userID string) ([]models.DeviceToken, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("parse user_id: %w", err)
	}
	rows, err := r.q.DeviceTokenListByUser(ctx, uid)
	if err != nil {
		return nil, err
	}
	out := make([]models.DeviceToken, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.DeviceToken{
			ID: row.ID.String(),
			UserID: row.UserID.String(),
			APNSToken: row.ApnsToken,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *DeviceTokensRepository) ListByUsers(ctx context.Context, userIDs []string) ([]models.DeviceToken, error) {
	uids := make([]uuid.UUID, 0, len(userIDs))
	for _, s := range userIDs {
		u, err := uuid.Parse(s)
		if err != nil {
			continue
		}
		uids = append(uids, u)
	}
	if len(uids) == 0 {
		return nil, nil
	}
	rows, err := r.q.DeviceTokenListByUsers(ctx, uids)
	if err != nil {
		return nil, err
	}
	out := make([]models.DeviceToken, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.DeviceToken{
			ID: row.ID.String(),
			UserID: row.UserID.String(),
			APNSToken: row.ApnsToken,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *DeviceTokensRepository) DeleteByToken(ctx context.Context, apnsToken string) (int64, error) {
	return r.q.DeviceTokenDeleteByToken(ctx, apnsToken)
}
