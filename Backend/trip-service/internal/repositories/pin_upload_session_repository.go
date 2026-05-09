package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"pinz/backend/trip-service/internal/db/sqlcdb"
	"pinz/backend/trip-service/internal/models"
)

var (
	ErrPinUploadSessionActive     = errors.New("another pin upload session is already active")
	ErrPinUploadSessionNotFound   = errors.New("pin upload session not found or closed")
	ErrPinUploadSessionWrongState = errors.New("pin upload session is not in the expected processing state")
)

type PinUploadSessionRepository struct {
	q  *sqlcdb.Queries
	db *sql.DB
}

func NewPinUploadSessionRepository(db *sql.DB) *PinUploadSessionRepository {
	return &PinUploadSessionRepository{q: sqlcdb.New(db), db: db}
}

// Create: targetPinID nil → creation (UNIQUE per trip), иначе addition (UNIQUE per pin).
func (r *PinUploadSessionRepository) Create(ctx context.Context, tripID string, targetPinID *string, userID string) (string, error) {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return "", err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", err
	}
	if targetPinID == nil {
		sid, err := r.q.PinUploadSessionCreateForCreation(ctx, sqlcdb.PinUploadSessionCreateForCreationParams{
			TripID:          tid,
			InitiatorUserID: uid,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", ErrPinUploadSessionActive
			}
			return "", err
		}
		return sid.String(), nil
	}
	pid, err := uuid.Parse(*targetPinID)
	if err != nil {
		return "", err
	}
	sid, err := r.q.PinUploadSessionCreateForAddition(ctx, sqlcdb.PinUploadSessionCreateForAdditionParams{
		TripID:          tid,
		TargetPinID:     uuid.NullUUID{UUID: pid, Valid: true},
		InitiatorUserID: uid,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrPinUploadSessionActive
		}
		return "", err
	}
	return sid.String(), nil
}

func (r *PinUploadSessionRepository) GetByID(ctx context.Context, sessionID string) (*models.PinUploadSession, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.PinUploadSessionGet(ctx, sid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPinUploadSessionNotFound
		}
		return nil, err
	}
	return rowToPinUploadSession(row), nil
}

func (r *PinUploadSessionRepository) GetActiveCreationForTrip(ctx context.Context, tripID string) (*models.PinUploadSession, error) {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.PinUploadSessionGetActiveCreation(ctx, tid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPinUploadSessionNotFound
		}
		return nil, err
	}
	return rowToPinUploadSession(row), nil
}

func (r *PinUploadSessionRepository) GetActiveAdditionForPin(ctx context.Context, pinID string) (*models.PinUploadSession, error) {
	pid, err := uuid.Parse(pinID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.PinUploadSessionGetActiveAddition(ctx, uuid.NullUUID{UUID: pid, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPinUploadSessionNotFound
		}
		return nil, err
	}
	return rowToPinUploadSession(row), nil
}

func (r *PinUploadSessionRepository) Touch(ctx context.Context, sessionID string) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}
	return r.q.PinUploadSessionTouch(ctx, sid)
}

func (r *PinUploadSessionRepository) SetDraftSnapshot(ctx context.Context, sessionID string, snapshot []byte) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}
	var raw pqtype.NullRawMessage
	if snapshot != nil {
		raw = pqtype.NullRawMessage{RawMessage: json.RawMessage(snapshot), Valid: true}
	}
	return r.q.PinUploadSessionSetSnapshot(ctx, sqlcdb.PinUploadSessionSetSnapshotParams{
		SessionID:     sid,
		DraftSnapshot: raw,
	})
}

// SetProcessingStatus — CAS; на mismatch → ErrPinUploadSessionWrongState.
func (r *PinUploadSessionRepository) SetProcessingStatus(ctx context.Context, sessionID, expected, next string) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}
	_, err = r.q.PinUploadSessionSetProcessingStatus(ctx, sqlcdb.PinUploadSessionSetProcessingStatusParams{
		SessionID:      sid,
		ExpectedStatus: expected,
		NextStatus:     next,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPinUploadSessionWrongState
		}
		return err
	}
	return nil
}

func (r *PinUploadSessionRepository) Close(ctx context.Context, sessionID, reason string) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}
	_, err = r.q.PinUploadSessionClose(ctx, sqlcdb.PinUploadSessionCloseParams{
		SessionID:   sid,
		CloseReason: sql.NullString{String: reason, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPinUploadSessionNotFound
		}
		return err
	}
	return nil
}

// AbandonedPinUploadSession — для cron-cleanup.
type AbandonedPinUploadSession struct {
	SessionID   string
	TripID      string
	TargetPinID *string
	InitiatorID string
}

func (r *PinUploadSessionRepository) ListAbandoned(ctx context.Context, threshold time.Time) ([]AbandonedPinUploadSession, error) {
	rows, err := r.q.PinUploadSessionListAbandoned(ctx, threshold)
	if err != nil {
		return nil, err
	}
	out := make([]AbandonedPinUploadSession, 0, len(rows))
	for _, row := range rows {
		entry := AbandonedPinUploadSession{
			SessionID:   row.SessionID.String(),
			TripID:      row.TripID.String(),
			InitiatorID: row.InitiatorUserID.String(),
		}
		if row.TargetPinID.Valid {
			s := row.TargetPinID.UUID.String()
			entry.TargetPinID = &s
		}
		out = append(out, entry)
	}
	return out, nil
}

// DeleteClosedOlderThan физически удаляет закрытые сессии старше threshold (cron-janitor).
func (r *PinUploadSessionRepository) DeleteClosedOlderThan(ctx context.Context, threshold time.Time) (int64, error) {
	return r.q.PinUploadSessionDeleteClosedOlderThan(ctx, sql.NullTime{Time: threshold, Valid: true})
}

func rawMessageOrNil(raw pqtype.NullRawMessage) []byte {
	if raw.Valid {
		return raw.RawMessage
	}
	return nil
}

func nullTimePtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

func nullStringPtr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	v := s.String
	return &v
}

func rowToPinUploadSession(row sqlcdb.PinUploadSession) *models.PinUploadSession {
	out := &models.PinUploadSession{
		SessionID:        row.SessionID.String(),
		TripID:           row.TripID.String(),
		InitiatorUserID:  row.InitiatorUserID.String(),
		DraftSnapshot:    rawMessageOrNil(row.DraftSnapshot),
		ProcessingStatus: row.ProcessingStatus,
		CreatedAt:        row.CreatedAt,
		LastActivityAt:   row.LastActivityAt,
		ClosedAt:         nullTimePtr(row.ClosedAt),
		CloseReason:      nullStringPtr(row.CloseReason),
	}
	if row.TargetPinID.Valid {
		s := row.TargetPinID.UUID.String()
		out.TargetPinID = &s
	}
	return out
}
