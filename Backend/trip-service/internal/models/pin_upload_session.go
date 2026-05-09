package models

import "time"

// PinUploadSession: TargetPinID nil — создание пина, заполнен — addition в пин.
type PinUploadSession struct {
	SessionID        string
	TripID           string
	TargetPinID      *string
	InitiatorUserID  string
	DraftSnapshot    []byte
	ProcessingStatus string
	CreatedAt        time.Time
	LastActivityAt   time.Time
	ClosedAt         *time.Time
	CloseReason      *string
}

const (
	PinUploadSessionCloseReasonConfirmed = "confirmed"
	PinUploadSessionCloseReasonCancelled = "cancelled"
	PinUploadSessionCloseReasonAbandoned = "abandoned"

	PinUploadProcessingStatusUploading      = "UPLOADING"
	PinUploadProcessingStatusProcessing     = "PROCESSING"
	PinUploadProcessingStatusReadyForReview = "READY_FOR_REVIEW"
)
