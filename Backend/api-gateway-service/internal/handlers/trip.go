package handlers

import (
	"net/http"

	"pinz/backend/api-gateway-service/internal/clients/trip"
	"pinz/backend/api-gateway-service/pkg/proto"
)

type TripHandler struct {
	tripClient *trip.Client
}

func NewTripHandler(tripClient *trip.Client) *TripHandler {
	return &TripHandler{tripClient: tripClient}
}

// ListTrips stub — CRUD will be implemented in task 2.
// @Summary List current user's trips
// @Description Returns list of trips for the authenticated user. Requires JWT.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} object
// @Failure 401 {object} responses.ErrorResponse
// @Failure 501 {object} responses.ErrorResponse
// @Router /api/v1/trips [get]
func (h *TripHandler) ListTrips(w http.ResponseWriter, r *http.Request) {
	_, err := h.tripClient.ListUserTrips(r.Context(), &proto.ListUserTripsRequest{
		UserId: "", // will be set from JWT in task 2
		Limit:  20,
		Offset: 0,
	})
	if err != nil {
		handleServiceError(w, r, err, "ListUserTrips")
		return
	}
	respondJSON(w, http.StatusOK, []interface{}{})
}

// CreateTrip stub — CRUD will be implemented in task 2.
// @Summary Create a new trip
// @Description Creates a new trip. Requires JWT. user_id will be taken from JWT in task 2.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body object true "Trip creation payload"
// @Success 201 {object} object
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 501 {object} responses.ErrorResponse
// @Router /api/v1/trips [post]
func (h *TripHandler) CreateTrip(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "CreateTrip will be implemented in task 2")
}

// GetTrip stub — CRUD will be implemented in task 2.
// @Summary Get trip by ID
// @Description Returns a single trip by ID. Requires JWT.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Success 200 {object} object
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 501 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id} [get]
func (h *TripHandler) GetTrip(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "GetTrip will be implemented in task 2")
}

// UpdateTrip stub — CRUD will be implemented in task 2.
// @Summary Update trip
// @Description Updates trip metadata. Requires JWT.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body object true "Update payload"
// @Success 200 {object} object
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 501 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id} [patch]
func (h *TripHandler) UpdateTrip(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "UpdateTrip will be implemented in task 2")
}

// DeleteTrip stub — CRUD will be implemented in task 2.
// @Summary Delete trip
// @Description Deletes a trip. Requires JWT.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Success 204
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 501 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id} [delete]
func (h *TripHandler) DeleteTrip(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "DeleteTrip will be implemented in task 2")
}
