package responses

// Trip is the REST representation of a trip.
type Trip struct {
	ID            string `json:"id"`
	OwnerUserID   string `json:"owner_user_id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	Season        string `json:"season"`
	Status        string `json:"status"`
	PrivacyLevel  string `json:"privacy_level"`
	StartDateUnix int64  `json:"start_date_unix,omitempty"`
	EndDateUnix   int64  `json:"end_date_unix,omitempty"`
	LikesCount    int32  `json:"likes_count"`
	DislikesCount     int32  `json:"dislikes_count"`
	MediaCount        int32  `json:"media_count"`
	ParticipantsCount int32  `json:"participants_count"`
	CoverURL          string `json:"cover_url,omitempty"`
	IsPublished   bool   `json:"is_published"`
	IsGenerated   bool   `json:"is_generated"`
	CreatedAtUnix int64  `json:"created_at_unix"`
	UpdatedAtUnix int64  `json:"updated_at_unix"`
}

// GetTripResponse is the response for GET /api/v1/trips/{id} (trip with pins and media).
type GetTripResponse struct {
	Trip Trip      `json:"trip"`
	Pins []TripPin `json:"pins"`
}

// TripPin is a pin with media for trip view.
type TripPin struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Category      string         `json:"category"`
	Latitude      *float64       `json:"latitude,omitempty"`
	Longitude     *float64       `json:"longitude,omitempty"`
	StartTimeUnix int64          `json:"start_time_unix,omitempty"`
	EndTimeUnix   int64          `json:"end_time_unix,omitempty"`
	PrivacyLevel  string         `json:"privacy_level"`
	Tags          []string       `json:"tags,omitempty"`
	Media         []TripPinMedia `json:"media,omitempty"`
}

// TripPinMedia is media inside a pin.
type TripPinMedia struct {
	MediaID        string `json:"media_id"`
	URL            string `json:"url"`
	MediaType      string `json:"media_type"`
	CapturedAtUnix int64  `json:"captured_at_unix,omitempty"`
	PrivacyLevel   string `json:"privacy_level"`
}

// CreateTripResponse is the response for POST /api/v1/trips
type CreateTripResponse struct {
	TripID     string      `json:"trip_id"`
	Status     string      `json:"status"`
	UploadURLs []UploadURL `json:"upload_urls"`
}

// UploadURL is a single presigned URL (phase 3).
type UploadURL struct {
	ClientID string `json:"client_id"`
	S3Key    string `json:"s3_key"`
	URL      string `json:"url"`
}

// TripCoverUploadResponse — ответ POST /api/v1/trips/:id/cover/upload: presigned PUT URL + ключ для последующего confirm.
type TripCoverUploadResponse struct {
	UploadURL string `json:"upload_url"`
	S3Key     string `json:"s3_key"`
}

// GenerateInviteLinkResponse is the response for POST /api/v1/trips/:id/invite
type GenerateInviteLinkResponse struct {
	InviteLinkID  string `json:"invite_link_id"`
	Token         string `json:"token"`
	InviteURL     string `json:"invite_url,omitempty"`
	ExpiresAtUnix int64  `json:"expires_at_unix"`
}

// JoinTripByTokenResponse is the response for POST /api/v1/trips/join
type JoinTripByTokenResponse struct {
	TripID        string `json:"trip_id"`
	AlreadyJoined bool   `json:"already_joined"`
}

// LeaveTripResponse is the response for POST /api/v1/trips/:id/leave
type LeaveTripResponse struct {
	Success     bool `json:"success"`
	TripDeleted bool `json:"trip_deleted"`
}

// ProcessMediaGroupingResponse is the response for POST /api/v1/trips/creation/:id/media/process-grouping
type ProcessMediaGroupingResponse struct {
	TripID    string     `json:"trip_id"`
	Status    string     `json:"status"`
	DraftPins []DraftPin `json:"draft_pins"`
}

// DraftPin is one draft pin with media list.
type DraftPin struct {
	DraftPinID string          `json:"draft_pin_id"`
	Media      []DraftPinMedia `json:"media"`
}

// DraftPinMedia is media in a draft pin.
type DraftPinMedia struct {
	MediaID string `json:"media_id"`
	URL     string `json:"url"`
	Type    string `json:"type"`
}

// ApplyGroupsAndProcessResponse is the response for POST /api/v1/trips/creation/:id/apply-groups-and-process (202)
type ApplyGroupsAndProcessResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

// GetTripReviewResponse is the response for GET /api/v1/trips/creation/:id/review
type GetTripReviewResponse struct {
	TripID  string      `json:"trip_id"`
	Status  string      `json:"status"`
	Similar [][]string  `json:"similar"` // groups of similar media IDs
	Pins    []ReviewPin `json:"pins"`
}

// ReviewPin is a pin in the review step.
type ReviewPin struct {
	PinID         string           `json:"pin_id"`
	Name          string           `json:"name"`
	Category      string           `json:"category"`
	Latitude      *float64         `json:"latitude,omitempty"`
	Longitude     *float64         `json:"longitude,omitempty"`
	LocationName  string           `json:"location_name,omitempty"`
	StartTimeUnix int64            `json:"start_time_unix,omitempty"`
	EndTimeUnix   int64            `json:"end_time_unix,omitempty"`
	Issues        []string         `json:"issues,omitempty"`
	Tags          []string         `json:"tags,omitempty"`
	Media         []ReviewPinMedia `json:"media,omitempty"`
}

// ReviewPinMedia is media in a review pin.
type ReviewPinMedia struct {
	MediaID      string `json:"media_id"`
	URL          string `json:"url"`
	PrivacyLevel string `json:"privacy_level"`
}

// FinalizeTripResponse is the response for POST /api/v1/trips/creation/:id/finalize
type FinalizeTripResponse struct {
	TripID  string `json:"trip_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// TripSettingsResponse is the response for PATCH /api/v1/trips/:id/settings (PINZ-98).
type TripSettingsResponse struct {
	Success bool `json:"success"`
}

// FeedPin is a lightweight pin for the feed card map view (PINZ-130).
type FeedPin struct {
	ID        string  `json:"id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// FeedMedia is a lightweight media item for the feed card carousel (PINZ-130).
type FeedMedia struct {
	MediaID   string `json:"media_id"`
	URL       string `json:"url"`
	MediaType string `json:"media_type"`
}

// FeedItem is a single card in the feed: trip data + pins + media (PINZ-130).
type FeedItem struct {
	Trip  Trip        `json:"trip"`
	Pins  []FeedPin   `json:"pins"`
	Media []FeedMedia `json:"media"`
}

// SuccessResponse is a generic success response for like/dislike/favourite.
type SuccessResponse struct {
	Success bool `json:"success"`
}

// AddMediaStartResponse is the response for POST /api/v1/trips/:id/media/add/start (PINZ-131).
type AddMediaStartResponse struct {
	SessionID  string      `json:"session_id"`
	Status     string      `json:"status"`
	UploadURLs []UploadURL `json:"upload_urls"`
}

// AddMediaProcessGroupingResponse is the response for POST /api/v1/trips/:id/media/add/process-grouping (PINZ-131).
// ExistingMediaIDs marks the original media as read-only per spec 5.3.3.
type AddMediaProcessGroupingResponse struct {
	TripID           string     `json:"trip_id"`
	SessionID        string     `json:"session_id"`
	Status           string     `json:"status"`
	DraftPins        []DraftPin `json:"draft_pins"`
	ExistingMediaIDs []string   `json:"existing_media_ids"`
}

// AddMediaApplyGroupsAndProcessResponse is the response for POST /api/v1/trips/:id/media/add/apply-groups-and-process (PINZ-131).
type AddMediaApplyGroupsAndProcessResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}
