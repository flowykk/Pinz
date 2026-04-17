package requests

// CreateTripRequest is the REST body for POST /api/v1/trips (creation/start). Privacy is set by default and changed via PATCH .../privacy + worker.
type CreateTripRequest struct {
	Name          string              `json:"name"`
	Description   string              `json:"description"`
	Category      string              `json:"category"`
	Season        string              `json:"season"`
	FilesToUpload []FileToUploadEntry `json:"files_to_upload,omitempty"`
}

// FileToUploadEntry is one file to upload (client_id + content_type for Presigned URL).
type FileToUploadEntry struct {
	ClientID    string `json:"client_id"`
	ContentType string `json:"content_type"`
}

// UpdateTripRequest is the REST body for PATCH /api/v1/trips/:id
type UpdateTripRequest struct {
	Name          *string `json:"name,omitempty"`
	Description   *string `json:"description,omitempty"`
	Category      *string `json:"category,omitempty"`
	Season        *string `json:"season,omitempty"`
	PrivacyLevel  *string `json:"privacy_level,omitempty"`
	StartDateUnix *int64  `json:"start_date_unix,omitempty"`
	EndDateUnix   *int64  `json:"end_date_unix,omitempty"`
	CoverS3Key    *string `json:"cover_s3_key,omitempty"`
}

// GenerateInviteLinkRequest is the REST body for POST /api/v1/trips/:id/invite
type GenerateInviteLinkRequest struct {
	ExpiresInSeconds *int64 `json:"expires_in_seconds,omitempty"`
}

// JoinTripByTokenRequest is the REST body for POST /api/v1/trips/join
type JoinTripByTokenRequest struct {
	Token string `json:"token"`
}

// ProcessMediaGroupingRequest is the REST body for POST /api/v1/trips/creation/:id/media/process-grouping
type ProcessMediaGroupingRequest struct {
	Media []MediaMetaEntry `json:"media"`
}

// MediaMetaEntry is metadata for one uploaded media file.
type MediaMetaEntry struct {
	S3Key       string   `json:"s3_key"`
	MediaType   string   `json:"media_type"`
	CapturedAt  string   `json:"captured_at,omitempty"` // ISO8601
	Latitude    *float64 `json:"latitude,omitempty"`
	Longitude   *float64 `json:"longitude,omitempty"`
	ContentHash *string  `json:"content_hash,omitempty"` // e.g. SHA-256 for duplicate detection
}

// ApplyGroupsAndProcessRequest is the REST body for POST /api/v1/trips/creation/:id/apply-groups-and-process
type ApplyGroupsAndProcessRequest struct {
	DraftPins       []DraftPinInput `json:"draft_pins"`
	DeletedMediaIDs []string        `json:"deleted_media_ids,omitempty"`
}

// DraftPinInput is one draft pin with media IDs.
type DraftPinInput struct {
	DraftPinID string   `json:"draft_pin_id"`
	MediaIDs   []string `json:"media_ids"`
}

// FinalizeTripRequest is the REST body for POST /api/v1/trips/creation/:id/finalize
type FinalizeTripRequest struct {
	PinUpdates    []PinUpdateInput `json:"pin_updates,omitempty"`
	MediaToDelete []string         `json:"media_to_delete,omitempty"`
}

// PublishTripRequest is the REST body for POST /api/v1/trips/:id/publish.
type PublishTripRequest struct {
	PublishWhole bool     `json:"publish_whole,omitempty"`
	PinIDs       []string `json:"pin_ids,omitempty"`
}

// PinUpdateInput is name and/or manual coordinates for a pin.
type PinUpdateInput struct {
	PinID        string   `json:"pin_id"`
	Name         *string  `json:"name,omitempty"`
	Latitude     *float64 `json:"latitude,omitempty"`
	Longitude    *float64 `json:"longitude,omitempty"`
	Description  *string  `json:"description,omitempty"`
	Category     *string  `json:"category,omitempty"`
	PrivacyLevel *string  `json:"privacy_level,omitempty"`
	Tags         []string `json:"tags,omitempty"`
}

type UpdatePinRequest struct {
	Name          *string  `json:"name,omitempty"`
	Description   *string  `json:"description,omitempty"`
	Category      *string  `json:"category,omitempty"`
	PrivacyLevel  *string  `json:"privacy_level,omitempty"`
	Latitude      *float64 `json:"latitude,omitempty"`
	Longitude     *float64 `json:"longitude,omitempty"`
	StartTimeUnix *int64   `json:"start_time_unix,omitempty"`
	EndTimeUnix   *int64   `json:"end_time_unix,omitempty"`
}

// AddMediaStartRequest is the REST body for POST /api/v1/trips/:id/media/add/start
type AddMediaStartRequest struct {
	FilesToUpload []FileToUploadEntry `json:"files_to_upload,omitempty"`
}

// AddMediaProcessGroupingRequest is the REST body for POST /api/v1/trips/:id/media/add/process-grouping
type AddMediaProcessGroupingRequest struct {
	SessionID string           `json:"session_id"`
	Media     []MediaMetaEntry `json:"media"`
}

// AddMediaApplyGroupsAndProcessRequest is the REST body for POST /api/v1/trips/:id/media/add/apply-groups-and-process
type AddMediaApplyGroupsAndProcessRequest struct {
	SessionID       string          `json:"session_id"`
	DraftPins       []DraftPinInput `json:"draft_pins"`
	DeletedMediaIDs []string        `json:"deleted_media_ids,omitempty"`
}

// TripSettingsRequest is the REST body for PATCH /api/v1/trips/:id/settings.
type TripSettingsRequest struct {
	NotificationsEnabled bool `json:"notifications_enabled"`
}
