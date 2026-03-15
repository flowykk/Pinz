package models

import "time"

// Trip represents a journey (путешествие).
type Trip struct {
	ID            string
	OwnerUserID   string
	Name          string
	Description   string
	Category      string
	Season        string
	Status        string
	PrivacyLevel  string
	StartDate     *time.Time
	EndDate       *time.Time
	LikesCount    int32
	DislikesCount int32
	CoverURL      string
	IsPublished   bool
	IsGenerated   bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// TripParticipant links a user to a trip with optional admin flag.
type TripParticipant struct {
	TripID   string
	UserID   string
	IsAdmin  bool
	JoinedAt time.Time
}
