package repositories

import (
	"database/sql"
	"encoding/json"
	"errors"

	sq "github.com/Masterminds/squirrel"

	"pinz/backend/trip-service/internal/models"
)

type MediaBattleRepository struct {
	db *sql.DB
}

func NewMediaBattleRepository(db *sql.DB) *MediaBattleRepository {
	return &MediaBattleRepository{db: db}
}

func (r *MediaBattleRepository) Create(b *models.MediaBattle) error {
	idsJSON, err := json.Marshal(b.MediaIDs)
	if err != nil {
		return err
	}
	q := psq.Insert("media_battles").
		Columns("trip_id", "user_id", "media_ids").
		Values(b.TripID, b.UserID, idsJSON).
		Suffix("RETURNING id, created_at")
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return err
	}
	return r.db.QueryRow(sqlStr, args...).Scan(&b.ID, &b.CreatedAt)
}

func (r *MediaBattleRepository) GetByID(id string) (*models.MediaBattle, error) {
	sqlStr := `SELECT id, trip_id, user_id, media_ids, winner_media_id, created_at, finished_at
		FROM media_battles WHERE id = $1`
	var b models.MediaBattle
	var idsJSON []byte
	var winner sql.NullString
	var finishedAt sql.NullTime
	err := r.db.QueryRow(sqlStr, id).Scan(&b.ID, &b.TripID, &b.UserID, &idsJSON, &winner, &b.CreatedAt, &finishedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	if err := json.Unmarshal(idsJSON, &b.MediaIDs); err != nil {
		return nil, err
	}
	if winner.Valid {
		b.WinnerMediaID = &winner.String
	}
	if finishedAt.Valid {
		t := finishedAt.Time
		b.FinishedAt = &t
	}
	return &b, nil
}

// SetWinner помечает батл завершённым и фиксирует победителя. Идемпотентно защищает от повторного инкремента:
// если finished_at уже установлен — RowsAffected==0, возвращаем sql.ErrNoRows, вызывающий код трактует это как
// "батл уже завершён" (см. services/battle.go). Использует NOW для finished_at.
func (r *MediaBattleRepository) SetWinner(battleID, winnerMediaID string) error {
	res, err := psq.Update("media_battles").
		Set("winner_media_id", winnerMediaID).
		Set("finished_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": battleID}).
		Where(sq.Eq{"finished_at": nil}).
		RunWith(r.db).Exec()
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

