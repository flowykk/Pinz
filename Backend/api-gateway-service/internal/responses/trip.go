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
