package models

import "time"

// Pin is a point on the trip map (pin from tripCreationFlow).
type Pin struct {
	ID                string
	TripID            string
	Name              string
	Description       string
	Latitude          *float64
	Longitude         *float64
	Category          string
	PrivacyLevel      string
	MediaCount        int32
	StartTime         *time.Time
	EndTime           *time.Time
	IsPublishedInFeed bool
	CreatedAt         time.Time
}

// Media is a photo or video in a trip.
// SimilarGroupID: медиа с одинаковым значением внутри одного пина образуют одну группу похожих (только для ответа review, в БД храним только идентификатор группы).
type Media struct {
	ID             string
	TripID         string
	PinID          *string
	S3Key          string
	MediaType      string
	Latitude       *float64
	Longitude      *float64
	CapturedAt     *time.Time
	BattleRating   int32
	PrivacyLevel   string
	SimilarGroupID *string // один и тот же UUID = одна группа похожих внутри пина
	CreatedAt      time.Time
}

// Tag is a tag on a pin (trip_id, pin_id, tag).
type Tag struct {
	ID     string
	TripID string
	PinID  string
	Tag    string
}
