package models

import "time"

type UserStats struct {
	UserID string
	TotalLikes int32
	TotalDislikes int32
	BattlesFinished int32
	UpdatedAt time.Time
}

type GeoLocation struct {
	ID int32
	ParentID *int32
	Name string
	Type string
}

type VisitedLocation struct {
	LocationID int32
	Name string
	Type string
	ParentID int32
	VisitCount int32
	LastVisitAt time.Time
}
