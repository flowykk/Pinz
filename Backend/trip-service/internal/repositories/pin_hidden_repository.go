package repositories

import (
	"database/sql"
)

type PinHiddenRepository struct {
	db *sql.DB
}

func NewPinHiddenRepository(db *sql.DB) *PinHiddenRepository {
	return &PinHiddenRepository{db: db}
}

func (r *PinHiddenRepository) HidePinForUser(pinID, userID string) error {
	_, err := r.db.Exec(
		`INSERT INTO pin_hidden_by_user (pin_id, user_id) VALUES ($1, $2) ON CONFLICT (pin_id, user_id) DO NOTHING`,
		pinID, userID,
	)
	return err
}

func (r *PinHiddenRepository) ListHiddenPinIDsForUser(tripID, userID string) ([]string, error) {
	rows, err := r.db.Query(
		`SELECT ph.pin_id FROM pin_hidden_by_user ph INNER JOIN pins p ON p.id = ph.pin_id WHERE p.trip_id = $1 AND ph.user_id = $2`,
		tripID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
