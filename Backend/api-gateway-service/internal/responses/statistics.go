package responses

// UserStatsResponse — счётчики пользователя для экрана профиля (ТЗ 10.2).
// total_trips/pins/media агрегируются API Gateway из trip-service.ListUserTripSummaries,
// остальные — из statistics-service (event-sourced).
type UserStatsResponse struct {
	UserID string `json:"user_id"`
	TotalTrips int32 `json:"total_trips"`
	TotalPins int32 `json:"total_pins"`
	TotalMedia int32 `json:"total_media"`
	TotalLikes int32 `json:"total_likes"`
	TotalDislikes int32 `json:"total_dislikes"`
	BattlesFinished int32 `json:"battles_finished"`
}

type VisitedLocationResponse struct {
	LocationID int32 `json:"location_id"`
	Name string `json:"name"`
	Type string `json:"type"`
	ParentID int32 `json:"parent_id"`
	VisitCount int32 `json:"visit_count"`
	LastVisitAtUnix int64 `json:"last_visit_at_unix"`
}

type VisitedLocationsResponse struct {
	Locations []VisitedLocationResponse `json:"locations"`
}
