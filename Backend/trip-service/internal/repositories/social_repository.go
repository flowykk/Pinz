package repositories

import "database/sql"

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
	// Get current reaction if any
	var oldReaction string
	_ = tx.QueryRow("SELECT reaction FROM social WHERE user_id = $1 AND trip_id = $2", userID, tripID).Scan(&oldReaction)
	// Upsert
	_, err = tx.Exec(`
		INSERT INTO social (user_id, trip_id, reaction) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, trip_id) DO UPDATE SET reaction = $3`,
		userID, tripID, reaction)
	if err != nil {
		return err
	}
	// Update trip counters only when reaction changed: decrement old, increment new
	if oldReaction != reaction {
		if oldReaction == reactionLike {
			_, _ = tx.Exec("UPDATE trips SET likes_count = GREATEST(0, likes_count - 1), updated_at = NOW() WHERE id = $1", tripID)
		} else if oldReaction == reactionDislike {
			_, _ = tx.Exec("UPDATE trips SET dislikes_count = GREATEST(0, dislikes_count - 1), updated_at = NOW() WHERE id = $1", tripID)
		}
		if reaction == reactionLike {
			_, _ = tx.Exec("UPDATE trips SET likes_count = likes_count + 1, updated_at = NOW() WHERE id = $1", tripID)
		} else {
			_, _ = tx.Exec("UPDATE trips SET dislikes_count = dislikes_count + 1, updated_at = NOW() WHERE id = $1", tripID)
		}
	}
	return tx.Commit()
}

// GetReaction returns the user's reaction for the trip ("", "Like", or "Dislike").
func (r *SocialRepository) GetReaction(userID, tripID string) (string, error) {
	var reaction string
	err := r.db.QueryRow("SELECT reaction FROM social WHERE user_id = $1 AND trip_id = $2", userID, tripID).Scan(&reaction)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return reaction, err
}
