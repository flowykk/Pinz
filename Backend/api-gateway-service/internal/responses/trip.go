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
	DislikesCount int32  `json:"dislikes_count"`
	CoverURL      string `json:"cover_url,omitempty"`
	IsPublished   bool   `json:"is_published"`
	IsGenerated   bool   `json:"is_generated"`
	CreatedAtUnix int64  `json:"created_at_unix"`
	UpdatedAtUnix int64  `json:"updated_at_unix"`
}

type TripParticipantRef struct {
	UserID  string `json:"user_id"`
	IsAdmin bool   `json:"is_admin"`
}

// GetTripResponse is the response for GET /api/v1/trips/{id} (trip with pins and media).
type GetTripResponse struct {
	Trip         TripDetail           `json:"trip"`
	Pins         []TripDetailPinItem  `json:"pins"`
	Participants []TripParticipantRef `json:"participants"`
}

// TripDetail is the same as Trip, used as nested in GetTripResponse.
type TripDetail = Trip

// TripDetailPinItem is a pin with media for GET trip.
type TripDetailPinItem struct {
	ID            string                `json:"id"`
	Name          string                `json:"name"`
	Description   string                `json:"description"`
	Category      string                `json:"category"`
	PrivacyLevel  string                `json:"privacy_level"`
	Latitude      *float64              `json:"latitude,omitempty"`
	Longitude     *float64              `json:"longitude,omitempty"`
	StartTimeUnix int64                 `json:"start_time_unix,omitempty"`
	EndTimeUnix   int64                 `json:"end_time_unix,omitempty"`
	Media         []TripDetailMediaItem `json:"media"`
}

// TripDetailMediaItem is media in a trip detail pin.
type TripDetailMediaItem struct {
	ID           string `json:"id"`
	S3Key        string `json:"s3_key"`
	MediaType    string `json:"media_type"`
	PrivacyLevel string `json:"privacy_level"`
}

// CreateTripResponse is the response for POST /api/v1/trips
type CreateTripResponse struct {
	TripID     string      `json:"trip_id"`
	Status     string      `json:"status"`
	UploadURLs []UploadURL `json:"upload_urls"`
}

// AddMediaStartResponse is the response for POST /api/v1/trips/:id/media/add/start
type AddMediaStartResponse struct {
	TripID     string      `json:"trip_id"`
	SessionID  string      `json:"session_id"`
	UploadURLs []UploadURL `json:"upload_urls"`
}

// AddMediaGroupedPin represents a pin group (existing or new) in add-media grouping response.
type AddMediaGroupedPin struct {
	PinID    string                 `json:"pin_id"`
	ReadOnly bool                   `json:"read_only"`
	Media    []AddMediaGroupedMedia `json:"media"`
}

type AddMediaGroupedMedia struct {
	MediaID  string `json:"media_id"`
	ReadOnly bool   `json:"read_only"`
	URL      string `json:"url"`
	Type     string `json:"type"`
}

// AddMediaProcessGroupingResponse is the response for POST /api/v1/trips/:id/media/add/process-grouping
type AddMediaProcessGroupingResponse struct {
	TripID    string               `json:"trip_id"`
	SessionID string               `json:"session_id"`
	Pins      []AddMediaGroupedPin `json:"pins"`
}

// UploadURL is a single presigned URL (phase 3).
type UploadURL struct {
	ClientID string `json:"client_id"`
	S3Key    string `json:"s3_key"`
	URL      string `json:"url"`
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

// TransferAdminResponse is the response for POST /api/v1/trips/:id/transfer-admin
type TransferAdminResponse struct {
	Success bool `json:"success"`
}

// ProcessMediaGroupingResponse is the response for POST /api/v1/trips/:id/media/process-grouping
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

// ApplyGroupsAndProcessResponse is the response for POST /api/v1/trips/:id/apply-groups-and-process (202)
type ApplyGroupsAndProcessResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

// GetTripReviewResponse is the response for GET /api/v1/trips/:id/review
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

// FinalizeTripResponse is the response for POST /api/v1/trips/:id/finalize
type FinalizeTripResponse struct {
	TripID  string `json:"trip_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// TripSettingsResponse is the response for PATCH /api/v1/trips/:id/settings.
type TripSettingsResponse struct {
	Success bool `json:"success"`
}

// SuccessResponse is a generic success response for like/dislike/favourite.
type SuccessResponse struct {
	Success bool `json:"success"`
}

// SearchPinItem is one pin in search results.
type SearchPinItem struct {
	PinID       string   `json:"pin_id"`
	TripID      string   `json:"trip_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
}

// SearchPinsResponse is the response for GET /api/v1/pins/search.
type SearchPinsResponse struct {
	Pins []SearchPinItem `json:"pins"`
}

// CreatePinResponse is the response for POST /api/v1/pins.
type CreatePinResponse struct {
	PinID string `json:"pin_id"`
}

type FeedCardPin struct {
	ID        string  `json:"id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type FeedCardMedia struct {
	ID        string `json:"id"`
	S3Key     string `json:"s3_key"`
	MediaType string `json:"media_type"`
}

type FeedCard struct {
	Trip  Trip            `json:"trip"`
	Pins  []FeedCardPin   `json:"pins"`
	Media []FeedCardMedia `json:"media"`
}

type ListFeedResponse struct {
	Trips []Trip     `json:"trips,omitempty"`
	Cards []FeedCard `json:"cards"`
}
