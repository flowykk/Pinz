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
	DislikesCount     int32
	MediaCount        int32
	ParticipantsCount int32
	CoverURL          string
	IsPublished   bool
	IsGenerated   bool
	IsSoftDeleted bool
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

// InvitationLink is a shareable link to join a trip.
type InvitationLink struct {
	ID        string
	TripID    string
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}
