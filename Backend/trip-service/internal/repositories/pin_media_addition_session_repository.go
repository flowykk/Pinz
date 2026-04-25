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

// ErrPinAdditionSessionActive — для пина уже идёт активная add-media-сессия.
// Сервис интерпретирует как FailedPrecondition, чтобы клиент дождался finalize/cancel.
var ErrPinAdditionSessionActive = errors.New("another pin media addition session is already active")

// ErrPinAdditionSessionNotFound — сессия не существует или уже закрыта.
var ErrPinAdditionSessionNotFound = errors.New("pin media addition session not found or closed")

type PinMediaAdditionSessionRepository struct {
	q *sqlcdb.Queries
}

func NewPinMediaAdditionSessionRepository(db *sql.DB) *PinMediaAdditionSessionRepository {
	return &PinMediaAdditionSessionRepository{q: sqlcdb.New(db)}
}

// Create — создание новой сессии. ON CONFLICT срабатывает на уникальном индексе
// idx_pin_media_addition_sessions_active_per_pin (одна активная сессия на пин)
// и приводит к sql.ErrNoRows — сервис интерпретирует это как ErrPinAdditionSessionActive.
func (r *PinMediaAdditionSessionRepository) Create(ctx context.Context, tripID, pinID, userID string) (string, error) {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return "", err
	}
	pid, err := uuid.Parse(pinID)
	if err != nil {
		return "", err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", err
	}
	sid, err := r.q.PinMediaAdditionSessionCreate(ctx, sqlcdb.PinMediaAdditionSessionCreateParams{
		TripID:          tid,
		PinID:           pid,
		InitiatorUserID: uid,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrPinAdditionSessionActive
		}
		return "", err
	}
	return sid.String(), nil
}

func (r *PinMediaAdditionSessionRepository) GetByID(ctx context.Context, sessionID string) (*models.PinMediaAdditionSession, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.PinMediaAdditionSessionGet(ctx, sid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPinAdditionSessionNotFound
		}
		return nil, err
	}
	return rowToPinMediaAdditionSession(row), nil
}

func (r *PinMediaAdditionSessionRepository) GetActiveForPin(ctx context.Context, pinID string) (*models.PinMediaAdditionSession, error) {
	pid, err := uuid.Parse(pinID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.PinMediaAdditionSessionGetActiveForPin(ctx, pid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPinAdditionSessionNotFound
		}
		return nil, err
	}
	return rowToPinMediaAdditionSession(row), nil
}

func (r *PinMediaAdditionSessionRepository) Touch(ctx context.Context, sessionID string) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}
	return r.q.PinMediaAdditionSessionTouch(ctx, sid)
}

// SetDraftSnapshot сохраняет результат Process (PinAdditionDraft + similar) в JSONB.
// Snapshot читается из GetPinMediaAdditionReview для повторного отображения ревью.
func (r *PinMediaAdditionSessionRepository) SetDraftSnapshot(ctx context.Context, sessionID string, snapshot []byte) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}
	var raw pqtype.NullRawMessage
	if snapshot != nil {
		raw = pqtype.NullRawMessage{RawMessage: json.RawMessage(snapshot), Valid: true}
	}
	return r.q.PinMediaAdditionSessionSetSnapshot(ctx, sqlcdb.PinMediaAdditionSessionSetSnapshotParams{
		SessionID:     sid,
		DraftSnapshot: raw,
	})
}

// Close идемпотентно закрывает сессию (UPDATE ... WHERE closed_at IS NULL RETURNING).
// Возвращает ErrPinAdditionSessionNotFound если сессия уже закрыта или не существует.
func (r *PinMediaAdditionSessionRepository) Close(ctx context.Context, sessionID, reason string) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}
	_, err = r.q.PinMediaAdditionSessionClose(ctx, sqlcdb.PinMediaAdditionSessionCloseParams{
		SessionID:   sid,
		CloseReason: sql.NullString{String: reason, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPinAdditionSessionNotFound
		}
		return err
	}
	return nil
}

func rowToPinMediaAdditionSession(row sqlcdb.PinMediaAdditionSession) *models.PinMediaAdditionSession {
	out := &models.PinMediaAdditionSession{
		SessionID:       row.SessionID.String(),
		TripID:          row.TripID.String(),
		PinID:           row.PinID.String(),
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
