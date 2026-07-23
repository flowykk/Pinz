package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"pinz/backend/auth-service/internal/db/sqlcdb"
)

type CredentialRepository struct {
	q *sqlcdb.Queries
}

func NewCredentialRepository(db *sql.DB) *CredentialRepository {
	return &CredentialRepository{q: sqlcdb.New(db)}
}

// CreateCredential persists a new WebAuthn credential for the given user.
func (r *CredentialRepository) CreateCredential(userID string, cred *webauthn.Credential) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	data, err := json.Marshal(cred)
	if err != nil {
		return err
	}
	return r.q.CreateCredential(context.Background(), sqlcdb.CreateCredentialParams{
		UserID: uid,
		CredentialID: cred.ID,
		CredentialJson: data,
	})
}

// GetCredentialsByUserID returns all WebAuthn credentials registered for a user.
func (r *CredentialRepository) GetCredentialsByUserID(userID string) ([]webauthn.Credential, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	raws, err := r.q.GetCredentialJSONByUserID(context.Background(), uid)
	if err != nil {
		return nil, err
	}
	var creds []webauthn.Credential
	for _, data := range raws {
		var c webauthn.Credential
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	return creds, nil
}

// UpdateCredential updates the stored credential data (e.g. updated sign counter).
func (r *CredentialRepository) UpdateCredential(cred *webauthn.Credential) error {
	data, err := json.Marshal(cred)
	if err != nil {
		return err
	}
	n, err := r.q.UpdateCredentialJSON(context.Background(), sqlcdb.UpdateCredentialJSONParams{
		CredentialJson: data,
		CredentialID: cred.ID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("credential not found")
	}
	return nil
}
