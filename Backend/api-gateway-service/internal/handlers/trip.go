package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"pinz/backend/api-gateway-service/internal/middleware"
	"pinz/backend/api-gateway-service/internal/requests"
	"pinz/backend/api-gateway-service/internal/responses"
	"pinz/backend/api-gateway-service/pkg/proto"
)

// Ensure responses is parsed by swag for @Failure annotations.
var _ = responses.ErrorResponse{}

type TripHandler struct {
	tripClient TripClient
}

type TripClient interface {
	CreateTrip(ctx context.Context, req *proto.CreateTripRequest) (*proto.CreateTripResponse, error)
	GetTrip(ctx context.Context, req *proto.GetTripRequest) (*proto.GetTripResponse, error)
	ListUserTrips(ctx context.Context, req *proto.ListUserTripsRequest) (*proto.ListUserTripsResponse, error)
	UpdateTrip(ctx context.Context, req *proto.UpdateTripRequest) (*proto.UpdateTripResponse, error)
	DeleteTrip(ctx context.Context, req *proto.DeleteTripRequest) (*proto.DeleteTripResponse, error)
	GenerateInviteLink(ctx context.Context, req *proto.GenerateInviteLinkRequest) (*proto.GenerateInviteLinkResponse, error)
	JoinTripByToken(ctx context.Context, req *proto.JoinTripByTokenRequest) (*proto.JoinTripByTokenResponse, error)
	RemoveParticipant(ctx context.Context, req *proto.RemoveParticipantRequest) (*proto.RemoveParticipantResponse, error)
	LeaveTrip(ctx context.Context, req *proto.LeaveTripRequest) (*proto.LeaveTripResponse, error)
	TransferAdmin(ctx context.Context, req *proto.TransferAdminRequest) (*proto.TransferAdminResponse, error)
	ProcessMediaGrouping(ctx context.Context, req *proto.ProcessMediaGroupingRequest) (*proto.ProcessMediaGroupingResponse, error)
	ApplyGroupsAndProcess(ctx context.Context, req *proto.ApplyGroupsAndProcessRequest) (*proto.ApplyGroupsAndProcessResponse, error)
	GetTripReview(ctx context.Context, req *proto.GetTripReviewRequest) (*proto.GetTripReviewResponse, error)
	FinalizeTrip(ctx context.Context, req *proto.FinalizeTripRequest) (*proto.FinalizeTripResponse, error)
}

func NewTripHandler(tripClient TripClient) *TripHandler {
	return &TripHandler{tripClient: tripClient}
}

// ListTrips returns trips for the authenticated user.
// @Summary List current user's trips
// @Description Returns list of trips for the authenticated user. Requires JWT.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} responses.Trip
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips [get]
func (h *TripHandler) ListTrips(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	limit := int32(20)
	offset := int32(0)
	// Optional: parse limit/offset from query
	resp, err := h.tripClient.ListUserTrips(ctx, &proto.ListUserTripsRequest{
		UserId: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		handleServiceError(w, r, err, "ListUserTrips")
		return
	}
	out := make([]responses.Trip, len(resp.GetTrips()))
	for i, t := range resp.GetTrips() {
		out[i] = tripProtoToResponse(t)
	}
	respondJSON(w, http.StatusOK, out)
}

// CreateTrip creates a new trip.
// @Summary Create a new trip
// @Description Creates a new trip. Requires JWT. user_id is taken from JWT.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body requests.CreateTripRequest true "Trip creation payload"
// @Success 201 {object} responses.CreateTripResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips [post]
func (h *TripHandler) CreateTrip(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req requests.CreateTripRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	protoFiles := make([]*proto.FileToUpload, 0, len(req.FilesToUpload))
	for _, f := range req.FilesToUpload {
		protoFiles = append(protoFiles, &proto.FileToUpload{ClientId: f.ClientID, ContentType: f.ContentType})
	}
	resp, err := h.tripClient.CreateTrip(ctx, &proto.CreateTripRequest{
		OwnerUserId:   userID,
		Name:          req.Name,
		Description:   req.Description,
		Category:      req.Category,
		Season:        req.Season,
		PrivacyLevel:  req.PrivacyLevel,
		FilesToUpload: protoFiles,
	})
	if err != nil {
		handleServiceError(w, r, err, "CreateTrip")
		return
	}
	urls := make([]responses.UploadURL, len(resp.GetUploadUrls()))
	for i, u := range resp.GetUploadUrls() {
		urls[i] = responses.UploadURL{ClientID: u.GetClientId(), S3Key: u.GetS3Key(), URL: u.GetUrl()}
	}
	respondJSON(w, http.StatusCreated, responses.CreateTripResponse{
		TripID:     resp.GetTripId(),
		Status:     resp.GetStatus(),
		UploadURLs: urls,
	})
}

// GetTrip returns a single trip by ID.
// @Summary Get trip by ID
// @Description Returns a single trip by ID. Requires JWT. User must be a participant.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Success 200 {object} responses.Trip
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id} [get]
func (h *TripHandler) GetTrip(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	if tripID == "" {
		respondError(w, http.StatusBadRequest, "trip id required")
		return
	}
	resp, err := h.tripClient.GetTrip(ctx, &proto.GetTripRequest{TripId: tripID, UserId: userID})
	if err != nil {
		handleServiceError(w, r, err, "GetTrip")
		return
	}
	respondJSON(w, http.StatusOK, tripProtoToResponse(resp.GetTrip()))
}

// UpdateTrip updates trip metadata.
// @Summary Update trip
// @Description Updates trip metadata. Requires JWT. Only admin can update.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.UpdateTripRequest true "Update payload"
// @Success 200 {object} responses.Trip
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id} [patch]
func (h *TripHandler) UpdateTrip(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	if tripID == "" {
		respondError(w, http.StatusBadRequest, "trip id required")
		return
	}
	var req requests.UpdateTripRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	protoReq := &proto.UpdateTripRequest{TripId: tripID, UserId: userID}
	if req.Name != nil {
		protoReq.Name = req.Name
	}
	if req.Description != nil {
		protoReq.Description = req.Description
	}
	if req.Category != nil {
		protoReq.Category = req.Category
	}
	if req.Season != nil {
		protoReq.Season = req.Season
	}
	if req.PrivacyLevel != nil {
		protoReq.PrivacyLevel = req.PrivacyLevel
	}
	if req.StartDateUnix != nil {
		protoReq.StartDateUnix = req.StartDateUnix
	}
	if req.EndDateUnix != nil {
		protoReq.EndDateUnix = req.EndDateUnix
	}
	if req.CoverURL != nil {
		protoReq.CoverUrl = req.CoverURL
	}
	resp, err := h.tripClient.UpdateTrip(ctx, protoReq)
	if err != nil {
		handleServiceError(w, r, err, "UpdateTrip")
		return
	}
	respondJSON(w, http.StatusOK, tripProtoToResponse(resp.GetTrip()))
}

// DeleteTrip deletes a trip.
// @Summary Delete trip
// @Description Deletes a trip. Requires JWT. Only owner can delete.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Success 204
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id} [delete]
func (h *TripHandler) DeleteTrip(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	if tripID == "" {
		respondError(w, http.StatusBadRequest, "trip id required")
		return
	}
	_, err := h.tripClient.DeleteTrip(ctx, &proto.DeleteTripRequest{TripId: tripID, UserId: userID})
	if err != nil {
		handleServiceError(w, r, err, "DeleteTrip")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GenerateInviteLink creates an invite link for a trip (participants only).
// @Summary Generate invite link
// @Description Creates an invite link for the trip. Only participants can generate. Requires JWT.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.GenerateInviteLinkRequest false "Optional expires_in_seconds"
// @Success 201 {object} responses.GenerateInviteLinkResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/invite [post]
func (h *TripHandler) GenerateInviteLink(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	if tripID == "" {
		respondError(w, http.StatusBadRequest, "trip id required")
		return
	}
	var req requests.GenerateInviteLinkRequest
	_ = decodeJSONBody(r, &req)
	protoReq := &proto.GenerateInviteLinkRequest{TripId: tripID}
	if req.ExpiresInSeconds != nil {
		protoReq.ExpiresInSeconds = *req.ExpiresInSeconds
	}
	resp, err := h.tripClient.GenerateInviteLink(ctx, protoReq)
	if err != nil {
		handleServiceError(w, r, err, "GenerateInviteLink")
		return
	}
	respondJSON(w, http.StatusCreated, responses.GenerateInviteLinkResponse{
		InviteLinkID:  resp.GetInviteLinkId(),
		Token:         resp.GetToken(),
		InviteURL:     resp.GetInviteUrl(),
		ExpiresAtUnix: resp.GetExpiresAtUnix(),
	})
}

// JoinTripByToken joins a trip using an invite token.
// @Summary Join trip by token
// @Description Joins the current user to a trip using an invite token. Requires JWT.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body requests.JoinTripByTokenRequest true "Token"
// @Success 200 {object} responses.JoinTripByTokenResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/join [post]
func (h *TripHandler) JoinTripByToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req requests.JoinTripByTokenRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	resp, err := h.tripClient.JoinTripByToken(ctx, &proto.JoinTripByTokenRequest{Token: req.Token})
	if err != nil {
		handleServiceError(w, r, err, "JoinTripByToken")
		return
	}
	respondJSON(w, http.StatusOK, responses.JoinTripByTokenResponse{
		TripID:        resp.GetTripId(),
		AlreadyJoined: resp.GetAlreadyJoined(),
	})
}

// RemoveParticipant removes a participant from the trip (admin only).
// @Summary Remove participant
// @Description Removes a participant from the trip. Only admin can remove. Requires JWT.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param user_id path string true "User ID to remove"
// @Success 204
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/participants/{user_id} [delete]
func (h *TripHandler) RemoveParticipant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	participantUserID := chi.URLParam(r, "user_id")
	if tripID == "" || participantUserID == "" {
		respondError(w, http.StatusBadRequest, "trip id and user_id required")
		return
	}
	_, err := h.tripClient.RemoveParticipant(ctx, &proto.RemoveParticipantRequest{TripId: tripID, UserId: participantUserID})
	if err != nil {
		handleServiceError(w, r, err, "RemoveParticipant")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// LeaveTrip makes the current user leave the trip.
// @Summary Leave trip
// @Description Current user leaves the trip. If sole admin, trip may be deleted or new admin assigned. Requires JWT.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Success 200 {object} responses.LeaveTripResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/leave [post]
func (h *TripHandler) LeaveTrip(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	if tripID == "" {
		respondError(w, http.StatusBadRequest, "trip id required")
		return
	}
	resp, err := h.tripClient.LeaveTrip(ctx, &proto.LeaveTripRequest{TripId: tripID})
	if err != nil {
		handleServiceError(w, r, err, "LeaveTrip")
		return
	}
	respondJSON(w, http.StatusOK, responses.LeaveTripResponse{
		Success:     resp.GetSuccess(),
		TripDeleted: resp.GetTripDeleted(),
	})
}

// TransferAdmin transfers admin rights to another participant.
// @Summary Transfer admin
// @Description Transfers trip admin to another participant. Only current admin can transfer. Requires JWT.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.TransferAdminRequest true "New admin user ID"
// @Success 200 {object} responses.TransferAdminResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/transfer-admin [post]
func (h *TripHandler) TransferAdmin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	if tripID == "" {
		respondError(w, http.StatusBadRequest, "trip id required")
		return
	}
	var req requests.TransferAdminRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.NewAdminUserID == "" {
		respondError(w, http.StatusBadRequest, "new_admin_user_id required")
		return
	}
	_, err := h.tripClient.TransferAdmin(ctx, &proto.TransferAdminRequest{TripId: tripID, NewAdminUserId: req.NewAdminUserID})
	if err != nil {
		handleServiceError(w, r, err, "TransferAdmin")
		return
	}
	respondJSON(w, http.StatusOK, responses.TransferAdminResponse{Success: true})
}

// ProcessMediaGrouping saves media metadata and returns draft pins (tripCreationFlow stage 2).
// @Summary Process media grouping
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.ProcessMediaGroupingRequest true "Media metadata"
// @Success 200 {object} responses.ProcessMediaGroupingResponse
// @Router /api/v1/trips/{id}/media/process-grouping [post]
func (h *TripHandler) ProcessMediaGrouping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	if tripID == "" {
		respondError(w, http.StatusBadRequest, "trip id required")
		return
	}
	var req requests.ProcessMediaGroupingRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	media := make([]*proto.MediaMeta, 0, len(req.Media))
	for _, m := range req.Media {
		mm := &proto.MediaMeta{S3Key: m.S3Key, MediaType: m.MediaType}
		if m.CapturedAt != "" {
			if t, err := time.Parse(time.RFC3339, m.CapturedAt); err == nil {
				mm.CapturedAtUnix = t.Unix()
			}
		}
		if m.Latitude != nil {
			mm.Latitude = m.Latitude
		}
		if m.Longitude != nil {
			mm.Longitude = m.Longitude
		}
		media = append(media, mm)
	}
	resp, err := h.tripClient.ProcessMediaGrouping(ctx, &proto.ProcessMediaGroupingRequest{TripId: tripID, Media: media})
	if err != nil {
		handleServiceError(w, r, err, "ProcessMediaGrouping")
		return
	}
	draftPins := make([]responses.DraftPin, 0, len(resp.GetDraftPins()))
	for _, dp := range resp.GetDraftPins() {
		mediaList := make([]responses.DraftPinMedia, 0, len(dp.GetMedia()))
		for _, m := range dp.GetMedia() {
			mediaList = append(mediaList, responses.DraftPinMedia{MediaID: m.GetMediaId(), URL: m.GetUrl(), Type: m.GetType()})
		}
		draftPins = append(draftPins, responses.DraftPin{DraftPinID: dp.GetDraftPinId(), Media: mediaList})
	}
	respondJSON(w, http.StatusOK, responses.ProcessMediaGroupingResponse{
		TripID:    resp.GetTripId(),
		Status:    resp.GetStatus(),
		DraftPins: draftPins,
	})
}

// ApplyGroupsAndProcess applies user grouping and starts processing (202).
// @Summary Apply groups and process
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.ApplyGroupsAndProcessRequest true "Draft pins and deleted media"
// @Success 202 {object} responses.ApplyGroupsAndProcessResponse
// @Router /api/v1/trips/{id}/apply-groups-and-process [post]
func (h *TripHandler) ApplyGroupsAndProcess(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	if tripID == "" {
		respondError(w, http.StatusBadRequest, "trip id required")
		return
	}
	var req requests.ApplyGroupsAndProcessRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	draftPins := make([]*proto.DraftPinInput, 0, len(req.DraftPins))
	for _, dp := range req.DraftPins {
		draftPins = append(draftPins, &proto.DraftPinInput{DraftPinId: dp.DraftPinID, MediaIds: dp.MediaIDs})
	}
	resp, err := h.tripClient.ApplyGroupsAndProcess(ctx, &proto.ApplyGroupsAndProcessRequest{
		TripId:          tripID,
		DraftPins:       draftPins,
		DeletedMediaIds: req.DeletedMediaIDs,
	})
	if err != nil {
		handleServiceError(w, r, err, "ApplyGroupsAndProcess")
		return
	}
	respondJSON(w, http.StatusAccepted, responses.ApplyGroupsAndProcessResponse{
		Message: resp.GetMessage(),
		Status:  resp.GetStatus(),
	})
}

// GetTripReview returns pins with tags, issues, similar for review (tripCreationFlow stage 4).
// @Summary Get trip review
// @Tags trips
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Success 200 {object} responses.GetTripReviewResponse
// @Router /api/v1/trips/{id}/review [get]
func (h *TripHandler) GetTripReview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	if tripID == "" {
		respondError(w, http.StatusBadRequest, "trip id required")
		return
	}
	resp, err := h.tripClient.GetTripReview(ctx, &proto.GetTripReviewRequest{TripId: tripID})
	if err != nil {
		handleServiceError(w, r, err, "GetTripReview")
		return
	}
	similar := make([][]string, 0, len(resp.GetSimilar()))
	for _, g := range resp.GetSimilar() {
		similar = append(similar, g.GetMediaIds())
	}
	pins := make([]responses.ReviewPin, 0, len(resp.GetPins()))
	for _, p := range resp.GetPins() {
		mediaList := make([]responses.ReviewPinMedia, 0, len(p.GetMedia()))
		for _, m := range p.GetMedia() {
			mediaList = append(mediaList, responses.ReviewPinMedia{
				MediaID: m.GetMediaId(), URL: m.GetUrl(), PrivacyLevel: m.GetPrivacyLevel(),
			})
		}
		rp := responses.ReviewPin{
			PinID: p.GetPinId(), Name: p.GetName(), Category: p.GetCategory(),
			LocationName: p.GetLocationName(), StartTimeUnix: p.GetStartTimeUnix(), EndTimeUnix: p.GetEndTimeUnix(),
			Issues: p.GetIssues(), Tags: p.GetTags(), Media: mediaList,
		}
		if p.Latitude != nil {
			rp.Latitude = p.Latitude
		}
		if p.Longitude != nil {
			rp.Longitude = p.Longitude
		}
		pins = append(pins, rp)
	}
	respondJSON(w, http.StatusOK, responses.GetTripReviewResponse{
		TripID: resp.GetTripId(), Status: resp.GetStatus(), Similar: similar, Pins: pins,
	})
}

// FinalizeTrip applies pin updates, deletes media, sets trip READY (tripCreationFlow stage 5).
// @Summary Finalize trip
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.FinalizeTripRequest true "Pin updates and media to delete"
// @Success 200 {object} responses.FinalizeTripResponse
// @Router /api/v1/trips/{id}/finalize [post]
func (h *TripHandler) FinalizeTrip(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	if tripID == "" {
		respondError(w, http.StatusBadRequest, "trip id required")
		return
	}
	var req requests.FinalizeTripRequest
	_ = decodeJSONBody(r, &req)
	pinUpdates := make([]*proto.PinUpdate, 0, len(req.PinUpdates))
	for _, pu := range req.PinUpdates {
		pinUpdates = append(pinUpdates, &proto.PinUpdate{
			PinId: pu.PinID, Name: pu.Name, Latitude: pu.Latitude, Longitude: pu.Longitude,
		})
	}
	resp, err := h.tripClient.FinalizeTrip(ctx, &proto.FinalizeTripRequest{
		TripId: tripID, PinUpdates: pinUpdates, MediaToDelete: req.MediaToDelete,
	})
	if err != nil {
		handleServiceError(w, r, err, "FinalizeTrip")
		return
	}
	respondJSON(w, http.StatusOK, responses.FinalizeTripResponse{
		TripID: resp.GetTripId(), Status: resp.GetStatus(), Message: resp.GetMessage(),
	})
}

func tripProtoToResponse(t *proto.Trip) responses.Trip {
	if t == nil {
		return responses.Trip{}
	}
	return responses.Trip{
		ID:            t.GetId(),
		OwnerUserID:   t.GetOwnerUserId(),
		Name:          t.GetName(),
		Description:   t.GetDescription(),
		Category:      t.GetCategory(),
		Season:        t.GetSeason(),
		Status:        t.GetStatus(),
		PrivacyLevel:  t.GetPrivacyLevel(),
		StartDateUnix: t.GetStartDateUnix(),
		EndDateUnix:   t.GetEndDateUnix(),
		LikesCount:    t.GetLikesCount(),
		DislikesCount: t.GetDislikesCount(),
		CoverURL:      t.GetCoverUrl(),
		IsPublished:   t.GetIsPublished(),
		IsGenerated:   t.GetIsGenerated(),
		CreatedAtUnix: t.GetCreatedAtUnix(),
		UpdatedAtUnix: t.GetUpdatedAtUnix(),
	}
}
