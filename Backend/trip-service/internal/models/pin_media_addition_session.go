package models

import "time"

// PinMediaAdditionSession — состояние сессии добавления медиа в существующий
// пин (ТЗ 4.2.2 + 4.12-4.14). Сессия per-pin, без коллаборатив-роли: один
// инициатор ведёт от Start до Finalize или Cancel.
//
// DraftSnapshot хранит результат Process (хеш-дедуп, NSFW, similar-группы,
// pin issues) между этапами Process и Finalize, чтобы клиент мог получить
// review-снимок повторно через GetPinMediaAdditionReview.
type PinMediaAdditionSession struct {
	SessionID string
	TripID string
	PinID string
	InitiatorUserID string
	DraftSnapshot []byte
	CreatedAt time.Time
	LastActivityAt time.Time
	ClosedAt *time.Time
	CloseReason *string
}

const (
	PinMediaAdditionSessionCloseReasonConfirmed = "confirmed"
	PinMediaAdditionSessionCloseReasonCancelled = "cancelled"
)
