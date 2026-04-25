package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"pinz/backend/trip-service/internal/db/sqlcdb"
	"pinz/backend/trip-service/internal/models"
)

// ErrNoActiveSession — нет активной add-media сессии для трипа.
var ErrNoActiveSession = errors.New("no active add-media session")

type AddMediaSessionRepository struct {
	q *sqlcdb.Queries
}

func NewAddMediaSessionRepository(db *sql.DB) *AddMediaSessionRepository {
	return &AddMediaSessionRepository{q: sqlcdb.New(db)}
}

// Create пытается создать новую сессию. Если уже есть активная (UNIQUE-индекс сработал
// через ON CONFLICT DO NOTHING), возвращает sql.ErrNoRows — сервис должен позвать GetActive
// и вернуть существующую session_id клиенту с joined=true (B1).
func (r *AddMediaSessionRepository) Create(ctx context.Context, tripID string, existingMediaIDs []string) (sessionID string, err error) {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(existingMediaIDs)
	if err != nil {
		return "", err
	}
	sid, err := r.q.AddMediaSessionCreate(ctx, sqlcdb.AddMediaSessionCreateParams{
		TripID: tid,
		ExistingMediaIds: b,
	})
	if err != nil {
		return "", err
	}
	return sid.String(), nil
}

func (r *AddMediaSessionRepository) GetExistingMediaIDs(ctx context.Context, sessionID string) ([]string, string, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, "", err
	}
	row, err := r.q.AddMediaSessionGet(ctx, sid)
	if err != nil {
		return nil, "", err
	}
	var ids []string
	if err := json.Unmarshal(row.ExistingMediaIds, &ids); err != nil {
		return nil, "", err
	}
	return ids, row.TripID.String(), nil
}

func (r *AddMediaSessionRepository) Exists(ctx context.Context, tripID, sessionID string) (bool, error) {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return false, err
	}
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return false, err
	}
	n, err := r.q.AddMediaSessionExists(ctx, sqlcdb.AddMediaSessionExistsParams{TripID: tid, SessionID: sid})
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// GetActive возвращает активную сессию трипа (closed_at IS NULL). ErrNoActiveSession,
// если активной нет.
func (r *AddMediaSessionRepository) GetActive(ctx context.Context, tripID string) (*models.AddMediaSession, error) {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.AddMediaSessionGetActive(ctx, tid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoActiveSession
		}
		return nil, err
	}
	var existingIDs []string
	if len(row.ExistingMediaIds) > 0 {
		if err := json.Unmarshal(row.ExistingMediaIds, &existingIDs); err != nil {
			return nil, err
		}
	}
	s := &models.AddMediaSession{
		SessionID: row.SessionID.String(),
		TripID: row.TripID.String(),
		ExistingMediaIDs: existingIDs,
		LastActivityAt: row.LastActivityAt,
	}
	if row.CurrentInitiatorUserID.Valid {
		v := row.CurrentInitiatorUserID.UUID.String()
		s.CurrentInitiatorUserID = &v
	}
	if row.InitiatorAssignedAt.Valid {
		t := row.InitiatorAssignedAt.Time
		s.InitiatorAssignedAt = &t
	}
	return s, nil
}

// SetInitiator назначает/переназначает ведущего и обновляет last_activity_at (один запрос).
// Вызывается при AddMediaApplyGroupsAndProcess и при неявном перехвате .
func (r *AddMediaSessionRepository) SetInitiator(ctx context.Context, sessionID, userID string, at time.Time) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.q.AddMediaSessionSetInitiator(ctx, sqlcdb.AddMediaSessionSetInitiatorParams{
		SessionID: sid,
		CurrentInitiatorUserID: uuid.NullUUID{UUID: uid, Valid: true},
		InitiatorAssignedAt: sql.NullTime{Time: at, Valid: true},
	})
}

// Touch обновляет last_activity_at (мутирующие ручки add-media ревью).
func (r *AddMediaSessionRepository) Touch(ctx context.Context, sessionID string, at time.Time) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}
	return r.q.AddMediaSessionTouch(ctx, sqlcdb.AddMediaSessionTouchParams{
		SessionID: sid,
		LastActivityAt: at,
	})
}

// Close закрывает сессию (Confirm/Cancel/abandoned). Возвращает trip_id закрытой сессии;
// если 0 rows — сессия уже была закрыта (ErrNoActiveSession).
func (r *AddMediaSessionRepository) Close(ctx context.Context, sessionID, reason string, at time.Time) (tripID string, err error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return "", err
	}
	tid, err := r.q.AddMediaSessionClose(ctx, sqlcdb.AddMediaSessionCloseParams{
		SessionID: sid,
		ClosedAt: sql.NullTime{Time: at, Valid: true},
		CloseReason: sql.NullString{String: reason, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNoActiveSession
		}
		return "", err
	}
	return tid.String(), nil
}

// AbandonedSession — найденная cron-кандидат на авто-закрытие.
type AbandonedSession struct {
	SessionID string
	TripID string
}

// ListAbandoned — сессии без активности дольше threshold. Для cron session_cleanup.
func (r *AddMediaSessionRepository) ListAbandoned(ctx context.Context, threshold time.Time) ([]AbandonedSession, error) {
	rows, err := r.q.AddMediaSessionListAbandoned(ctx, threshold)
	if err != nil {
		return nil, err
	}
	out := make([]AbandonedSession, 0, len(rows))
	for _, r := range rows {
		out = append(out, AbandonedSession{
			SessionID: r.SessionID.String(),
			TripID: r.TripID.String(),
		})
	}
	return out, nil
}
