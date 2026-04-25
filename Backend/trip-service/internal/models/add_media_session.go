package models

import "time"

// AddMediaSession — состояние активной сессии добавления медиа .
// Ведущий (CurrentInitiator*) появляется с момента Apply: до этого оба поля nil,
// во время DRAFT_FINAL_REVIEW — заполнены и обновляются при мутации.
type AddMediaSession struct {
	SessionID string
	TripID string
	ExistingMediaIDs []string
	CurrentInitiatorUserID *string
	InitiatorAssignedAt *time.Time
	LastActivityAt time.Time
}
