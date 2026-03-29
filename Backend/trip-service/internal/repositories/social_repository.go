package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"pinz/backend/trip-service/internal/db/sqlcdb"
)

const reactionLike = "Like"
const reactionDislike = "Dislike"

type SocialRepository struct {
	db *sql.DB
}

func NewSocialRepository(db *sql.DB) *SocialRepository {
	return &SocialRepository{db: db}
}

// SetReaction sets or replaces user's reaction (Like or Dislike). Updates trip likes_count/dislikes_count.
func (r *SocialRepository) SetReaction(userID, tripID, reaction string) error {
	if reaction != reactionLike && reaction != reactionDislike {
		return nil
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	q := sqlcdb.New(tx)
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return err
	}

	oldReaction, err := q.SocialGetReaction(context.Background(), sqlcdb.SocialGetReactionParams{UserID: uid, TripID: tid})
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		oldReaction = ""
	}

	if err := q.SocialUpsert(context.Background(), sqlcdb.SocialUpsertParams{UserID: uid, TripID: tid, Reaction: reaction}); err != nil {
		return err
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
	return tx.Commit()
}

// GetReaction returns the user's reaction for the trip ("", "Like", or "Dislike").
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
