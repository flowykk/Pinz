package models

import "time"

// Pin is a point on the trip map.
type Pin struct {
	ID string
	TripID string
	Name string
	Description string
	Latitude *float64
	Longitude *float64
	Category string
	PrivacyLevel string
	MediaCount int32
	StartTime *time.Time
	EndTime *time.Time
	LocationName string
	IsPublishedInFeed bool
	CreatedAt time.Time
}

// Media is a photo or video in a trip.
// SimilarGroupID: медиа с одинаковым значением внутри одного пина образуют одну группу похожих (только для ответа review, в БД храним только идентификатор группы).
// UploadedBy: автор загрузки . nil для медиа, загруженных до этой миграции.
// PinAdditionSessionID: связь с активной pin-add-media-сессией (ТЗ 4.2.2 + 4.12-4.14).
// До FinalizePinMediaAddition media лежит с pin_id=NULL и pin_addition_session_id=session;
// при finalize pin_id заполняется и pin_addition_session_id очищается через ON DELETE SET NULL FK.
// PinCreationSessionID: связь с pin-creation-сессией (ТЗ 4.1, 4.6-4.11). До
// FinalizePinCreation media лежит с pin_id=NULL и pin_creation_session_id=session;
// на finalize создаётся новый pin и pin_id заполняется через UpdatePinIDByIDs.
type Media struct {
	ID string
	TripID string
	PinID *string
	S3Key string
	MediaType string
	Latitude *float64
	Longitude *float64
	CapturedAt *time.Time
	BattleRating int32
	PrivacyLevel string
	SimilarGroupID *string
	ContentHash *string
	UploadedBy *string
	PinAdditionSessionID *string
	PinCreationSessionID *string
	CreatedAt time.Time
}

// Tag is a tag on a pin (trip_id, pin_id, tag).
type Tag struct {
	ID string
	TripID string
	PinID string
	Tag string
}

// MediaBattle — сессия фотобатла (ТЗ 8.1). MediaIDs — 8 случайных медиа, WinnerMediaID заполняется на SubmitBattleResult.
type MediaBattle struct {
	ID string
	TripID string
	UserID string
	MediaIDs []string
	WinnerMediaID *string
	CreatedAt time.Time
	FinishedAt *time.Time
}
