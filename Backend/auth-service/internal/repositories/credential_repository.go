package repositories

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/go-webauthn/webauthn/webauthn"
)

type CredentialRepository struct {
	db *sql.DB
}

func NewCredentialRepository(db *sql.DB) *CredentialRepository {
	return &CredentialRepository{db: db}
}

// CreateCredential persists a new WebAuthn credential for the given user.
// The full credential is serialised to JSON and stored alongside its raw ID.
func (r *CredentialRepository) CreateCredential(userID string, cred *webauthn.Credential) error {
	data, err := json.Marshal(cred)
	if err != nil {
		return err
	}
	q := psq.Insert("passkey_credentials").
		Columns("user_id", "credential_id", "credential_data").
		Values(userID, cred.ID, data)
	_, err = q.RunWith(r.db).Exec()
	return err
}

// GetCredentialsByUserID returns all WebAuthn credentials registered for a user.
func (r *CredentialRepository) GetCredentialsByUserID(userID string) ([]webauthn.Credential, error) {
	rows, err := r.db.Query(
		`SELECT credential_data FROM passkey_credentials WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []webauthn.Credential
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var c webauthn.Credential
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	return creds, rows.Err()
}

// UpdateCredential updates the stored credential data (e.g. updated sign counter).
// It matches on the raw credential ID.
func (r *CredentialRepository) UpdateCredential(cred *webauthn.Credential) error {
	data, err := json.Marshal(cred)
	if err != nil {
		return err
	}
	res, err := r.db.Exec(
		`UPDATE passkey_credentials SET credential_data = $1 WHERE credential_id = $2`,
		data, cred.ID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("credential not found")
	}
	return nil
}
