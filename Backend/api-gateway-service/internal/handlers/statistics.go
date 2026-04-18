package handlers

import (
	"context"
	"net/http"

	"pinz/backend/api-gateway-service/internal/middleware"
	"pinz/backend/api-gateway-service/internal/responses"
	pb "pinz/backend/api-gateway-service/pkg/proto"
)

// StatisticsClient — RPC клиент statistics-service (персональные счётчики + локации).
type StatisticsClient interface {
	GetUserStats(ctx context.Context, req *pb.GetUserStatsRequest) (*pb.GetUserStatsResponse, error)
	GetVisitedLocations(ctx context.Context, req *pb.GetVisitedLocationsRequest) (*pb.GetVisitedLocationsResponse, error)
}

// StatsTripClient — часть trip-service клиента, нужная для агрегации в gateway.
type StatsTripClient interface {
	ListUserTripSummaries(ctx context.Context, req *pb.ListUserTripSummariesRequest) (*pb.ListUserTripSummariesResponse, error)
}

type StatisticsHandler struct {
	stats StatisticsClient
	trip  StatsTripClient
}

func NewStatisticsHandler(stats StatisticsClient, trip StatsTripClient) *StatisticsHandler {
	return &StatisticsHandler{stats: stats, trip: trip}
}

// GetProfileStats — собирает профильную статистику (ТЗ 10.2):
//   - total_trips = len(summaries), total_pins/total_media = суммы по трипам
//   - total_likes/total_dislikes/battles_finished — из statistics-service
//
// @Summary Get current user's stats
// @Description Returns counters (trips, pins, media, likes, dislikes, battles) for authenticated user.
// @Tags statistics
// @Produce json
// @Success 200 {object} responses.UserStatsResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/profile/stats [get]
func (h *StatisticsHandler) GetProfileStats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "user_id is required")
		return
	}

	summariesResp, err := h.trip.ListUserTripSummaries(r.Context(), &pb.ListUserTripSummariesRequest{UserId: userID})
	if err != nil {
		handleServiceError(w, r, err, "ListUserTripSummaries")
		return
	}
	out := responses.UserStatsResponse{UserID: userID}
	for _, t := range summariesResp.GetTrips() {
		out.TotalTrips++
		out.TotalPins += t.GetPinsCount()
		out.TotalMedia += t.GetMediaCount()
	}

	statsResp, err := h.stats.GetUserStats(r.Context(), &pb.GetUserStatsRequest{UserId: userID})
	if err != nil {
		handleServiceError(w, r, err, "GetUserStats")
		return
	}
	if s := statsResp.GetStats(); s != nil {
		out.TotalLikes = s.GetTotalLikes()
		out.TotalDislikes = s.GetTotalDislikes()
		out.BattlesFinished = s.GetBattlesFinished()
	}
	respondJSON(w, http.StatusOK, out)
}

// GetProfileVisitedLocations — ТЗ 10.1: посещённые страны/города пользователя.
// Реализация: получает список trip_ids из trip-service (включая те, что появились после
// поздних join'ов) и агрегирует mirror в statistics-service.
//
// @Summary Get current user's visited locations
// @Description Returns countries/cities where the user had trips. Optional ?type=Country|City.
// @Tags statistics
// @Produce json
// @Param type query string false "Filter by type: Country | City"
// @Success 200 {object} responses.VisitedLocationsResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/profile/visited-locations [get]
func (h *StatisticsHandler) GetProfileVisitedLocations(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "user_id is required")
		return
	}
	typeFilter := r.URL.Query().Get("type")

	summariesResp, err := h.trip.ListUserTripSummaries(r.Context(), &pb.ListUserTripSummariesRequest{UserId: userID})
	if err != nil {
		handleServiceError(w, r, err, "ListUserTripSummaries")
		return
	}
	tripIDs := make([]string, 0, len(summariesResp.GetTrips()))
	for _, t := range summariesResp.GetTrips() {
		tripIDs = append(tripIDs, t.GetTripId())
	}

	resp, err := h.stats.GetVisitedLocations(r.Context(), &pb.GetVisitedLocationsRequest{
		UserId:  userID,
		TripIds: tripIDs,
		Type:    typeFilter,
	})
	if err != nil {
		handleServiceError(w, r, err, "GetVisitedLocations")
		return
	}
	out := responses.VisitedLocationsResponse{Locations: make([]responses.VisitedLocationResponse, 0, len(resp.GetLocations()))}
	for _, l := range resp.GetLocations() {
		out.Locations = append(out.Locations, responses.VisitedLocationResponse{
			LocationID:      l.GetLocationId(),
			Name:            l.GetName(),
			Type:            l.GetType(),
			ParentID:        l.GetParentId(),
			VisitCount:      l.GetVisitCount(),
			LastVisitAtUnix: l.GetLastVisitAtUnix(),
		})
	}
	respondJSON(w, http.StatusOK, out)
}
