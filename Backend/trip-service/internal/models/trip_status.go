package models

// Trip status values. Stored as TEXT in trips.status.
const (
	TripStatusDraft = "DRAFT"
	TripStatusUploading = "UPLOADING"
	TripStatusDraftGroupingReview = "DRAFT_GROUPING_REVIEW"
	TripStatusProcessing = "PROCESSING"
	TripStatusDraftFinalReview = "DRAFT_FINAL_REVIEW"
	TripStatusReady = "READY"

	TripStatusAddMediaUploading = "ADD_MEDIA_UPLOADING"
	TripStatusAddMediaGroupingReview = "ADD_MEDIA_GROUPING_REVIEW"
	TripStatusAddMediaProcessing = "ADD_MEDIA_PROCESSING"
	TripStatusAddMediaDraftFinalReview = "ADD_MEDIA_DRAFT_FINAL_REVIEW"
)

// Close reasons for add_media_sessions.
const (
	AddMediaSessionCloseReasonConfirmed = "confirmed"
	AddMediaSessionCloseReasonCancelled = "cancelled"
	AddMediaSessionCloseReasonAbandoned = "abandoned"
	AddMediaSessionCloseReasonLegacy = "legacy"
)
