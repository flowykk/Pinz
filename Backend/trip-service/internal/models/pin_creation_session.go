package models

import "time"

// PinCreationSession — состояние сессии создания одиночного пина в READY-трипе
// (ТЗ 4.1, 4.6-4.11). Сессия per-trip без коллаборатив-роли: один инициатор
// ведёт от Start до Finalize или Cancel.
//
// DraftSnapshot хранит результат Process (хеш-дедуп, NSFW, similar groups +
// suggested поля для нового пина: имя/категория/теги/координаты/start-end)
// между этапами Process и Finalize, чтобы клиент мог получить review-снимок
// повторно через GetPinCreationReview.
type PinCreationSession struct {
	SessionID string
	TripID string
	InitiatorUserID string
	DraftSnapshot []byte
	CreatedAt time.Time
	LastActivityAt time.Time
	ClosedAt *time.Time
	CloseReason *string
}

const (
	PinCreationSessionCloseReasonConfirmed = "confirmed"
	PinCreationSessionCloseReasonCancelled = "cancelled"
)
