package repositories

import (
	"context"
	"database/sql"
	"errors"

	"pinz/backend/statistics-service/internal/models"
)

type UserStatsRepository struct {
	db *sql.DB
}

func NewUserStatsRepository(db *sql.DB) *UserStatsRepository {
	return &UserStatsRepository{db: db}
}

func (r *UserStatsRepository) GetByUserID(ctx context.Context, userID string) (*models.UserStats, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT user_id, total_likes, total_dislikes, battles_finished, updated_at
		FROM user_stats WHERE user_id = $1`, userID)
	var s models.UserStats
	if err := row.Scan(&s.UserID, &s.TotalLikes, &s.TotalDislikes, &s.BattlesFinished, &s.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &models.UserStats{UserID: userID}, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *UserStatsRepository) increment(ctx context.Context, userID, column string, delta int32) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_stats (user_id, `+column+`)
		VALUES ($1, GREATEST($2, 0))
		ON CONFLICT (user_id) DO UPDATE SET
			`+column+` = GREATEST(user_stats.`+column+` + $2, 0),
			updated_at = NOW()`, userID, delta)
	return err
}

func (r *UserStatsRepository) IncrementLikes(ctx context.Context, userID string, delta int32) error {
	return r.increment(ctx, userID, "total_likes", delta)
}

func (r *UserStatsRepository) IncrementDislikes(ctx context.Context, userID string, delta int32) error {
	return r.increment(ctx, userID, "total_dislikes", delta)
}

func (r *UserStatsRepository) IncrementBattles(ctx context.Context, userID string, delta int32) error {
	return r.increment(ctx, userID, "battles_finished", delta)
}
