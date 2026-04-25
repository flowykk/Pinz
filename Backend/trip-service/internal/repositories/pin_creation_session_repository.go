package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"pinz/backend/trip-service/internal/db/sqlcdb"
	"pinz/backend/trip-service/internal/models"
)

// ErrPinCreationSessionActive — для трипа уже идёт активная сессия создания пина.
// Сервис интерпретирует как FailedPrecondition.
var ErrPinCreationSessionActive = errors.New("another pin creation session is already active for this trip")

// ErrPinCreationSessionNotFound — сессия не существует или уже закрыта.
var ErrPinCreationSessionNotFound = errors.New("pin creation session not found or closed")

type PinCreationSessionRepository struct {
	q *sqlcdb.Queries
}

func NewPinCreationSessionRepository(db *sql.DB) *PinCreationSessionRepository {
	return &PinCreationSessionRepository{q: sqlcdb.New(db)}
}

// Create — новая сессия. UNIQUE-индекс idx_pin_creation_sessions_active_per_trip
// + ON CONFLICT DO NOTHING → sql.ErrNoRows при race с уже активной сессией.
func (r *PinCreationSessionRepository) Create(ctx context.Context, tripID, userID string) (string, error) {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return "", err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", err
	}
	sid, err := r.q.PinCreationSessionCreate(ctx, sqlcdb.PinCreationSessionCreateParams{
		TripID:          tid,
		InitiatorUserID: uid,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrPinCreationSessionActive
		}
		return "", err
	}
	return sid.String(), nil
}

func (r *PinCreationSessionRepository) GetByID(ctx context.Context, sessionID string) (*models.PinCreationSession, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.PinCreationSessionGet(ctx, sid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPinCreationSessionNotFound
		}
		return nil, err
	}
	return rowToPinCreationSession(row), nil
}

func (r *PinCreationSessionRepository) GetActiveForTrip(ctx context.Context, tripID string) (*models.PinCreationSession, error) {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.PinCreationSessionGetActiveForTrip(ctx, tid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPinCreationSessionNotFound
		}
		return nil, err
	}
	return rowToPinCreationSession(row), nil
}

func (r *PinCreationSessionRepository) Touch(ctx context.Context, sessionID string) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}
	return r.q.PinCreationSessionTouch(ctx, sid)
}

func (r *PinCreationSessionRepository) SetDraftSnapshot(ctx context.Context, sessionID string, snapshot []byte) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}
	var raw pqtype.NullRawMessage
	if snapshot != nil {
		raw = pqtype.NullRawMessage{RawMessage: json.RawMessage(snapshot), Valid: true}
	}
	return r.q.PinCreationSessionSetSnapshot(ctx, sqlcdb.PinCreationSessionSetSnapshotParams{
		SessionID:     sid,
		DraftSnapshot: raw,
	})
}

// Close идемпотентно закрывает сессию (UPDATE ... WHERE closed_at IS NULL RETURNING).
// Если уже закрыта — ErrPinCreationSessionNotFound.
func (r *PinCreationSessionRepository) Close(ctx context.Context, sessionID, reason string) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}
	_, err = r.q.PinCreationSessionClose(ctx, sqlcdb.PinCreationSessionCloseParams{
		SessionID:   sid,
		CloseReason: sql.NullString{String: reason, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPinCreationSessionNotFound
		}
		return err
	}
	return nil
}

func rowToPinCreationSession(row sqlcdb.PinCreationSession) *models.PinCreationSession {
	out := &models.PinCreationSession{
		SessionID:       row.SessionID.String(),
		TripID:          row.TripID.String(),
		InitiatorUserID: row.InitiatorUserID.String(),
		CreatedAt:       row.CreatedAt,
		LastActivityAt:  row.LastActivityAt,
	}
	if row.DraftSnapshot.Valid {
		out.DraftSnapshot = row.DraftSnapshot.RawMessage
	}
	if row.ClosedAt.Valid {
		out.ClosedAt = &row.ClosedAt.Time
	}
	if row.CloseReason.Valid {
		out.CloseReason = &row.CloseReason.String
	}
	return out
}
