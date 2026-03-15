package requests

// CreateTripRequest is the REST body for POST /api/v1/trips
type CreateTripRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Category     string `json:"category"`
	Season       string `json:"season"`
	PrivacyLevel string `json:"privacy_level"`
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
	CoverURL      *string `json:"cover_url,omitempty"`
}
