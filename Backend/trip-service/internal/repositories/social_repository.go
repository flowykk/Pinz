package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"pinz/backend/trip-service/internal/db/sqlcdb"
)

const reactionLike = "like"
const reactionDislike = "dislike"

type SocialRepository struct {
	db *sql.DB
}

func NewSocialRepository(db *sql.DB) *SocialRepository {
	return &SocialRepository{db: db}
}

// SetReaction sets or replaces user's reaction (like or dislike). Updates trip likes_count/dislikes_count.
// Возвращает предыдущую реакцию (oldReaction: "", "like", "dislike"), чтобы вызывающий сервис
// мог корректно опубликовать LIKE_ADDED/LIKE_REMOVED/DISLIKE_* события для statistics.
func (r *SocialRepository) SetReaction(userID, tripID, reaction string) (oldReaction string, err error) {
	if reaction != reactionLike && reaction != reactionDislike {
		return "", nil
	}
	tx, err := r.db.Begin()
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()

	q := sqlcdb.New(tx)
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", err
	}
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return "", err
	}

	oldReaction, err = q.SocialGetReaction(context.Background(), sqlcdb.SocialGetReactionParams{UserID: uid, TripID: tid})
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return "", err
		}
		oldReaction = ""
	}

	if err := q.SocialUpsert(context.Background(), sqlcdb.SocialUpsertParams{UserID: uid, TripID: tid, Reaction: reaction}); err != nil {
		return oldReaction, err
	}

	if oldReaction != reaction {
		if oldReaction == reactionLike {
			_ = q.TripDecrementLikes(context.Background(), tid)
		} else if oldReaction == reactionDislike {
			_ = q.TripDecrementDislikes(context.Background(), tid)
		}
		if reaction == reactionLike {
			_ = q.TripIncrementLikes(context.Background(), tid)
		} else {
			_ = q.TripIncrementDislikes(context.Background(), tid)
		}
	}
	return oldReaction, tx.Commit()
}

// GetReactionsByUserAndTrips bulk-фетч реакций пользователя по списку trip_id.
// Возвращает map[tripID]reaction только для трипов, по которым у пользователя есть запись.
// Невалидные UUID в tripIDs пропускаются. На пустом входе — пустая мапа без ошибки.
func (r *SocialRepository) GetReactionsByUserAndTrips(userID string, tripIDs []string) (map[string]string, error) {
	out := make(map[string]string, len(tripIDs))
	if len(tripIDs) == 0 {
		return out, nil
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return out, err
	}
	tids := make([]uuid.UUID, 0, len(tripIDs))
	for _, id := range tripIDs {
		tid, perr := uuid.Parse(id)
		if perr != nil {
			continue
		}
		tids = append(tids, tid)
	}
	if len(tids) == 0 {
		return out, nil
	}
	q := sqlcdb.New(r.db)
	rows, err := q.SocialGetReactionsByUserAndTrips(context.Background(), sqlcdb.SocialGetReactionsByUserAndTripsParams{
		UserID: uid,
		Column2: tids,
	})
	if err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.TripID.String()] = row.Reaction
	}
	return out, nil
}

// GetReaction returns the user's reaction for the trip ("", "like", or "dislike").
func (r *SocialRepository) GetReaction(userID, tripID string) (string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", err
	}
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return "", err
	}
	q := sqlcdb.New(r.db)
	reaction, err := q.SocialGetReaction(context.Background(), sqlcdb.SocialGetReactionParams{UserID: uid, TripID: tid})
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return reaction, err
}
