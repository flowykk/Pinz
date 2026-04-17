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
	RequestTripCoverUpload(ctx context.Context, req *proto.RequestTripCoverUploadRequest) (*proto.RequestTripCoverUploadResponse, error)
	ConfirmTripCoverUpload(ctx context.Context, req *proto.ConfirmTripCoverUploadRequest) (*proto.ConfirmTripCoverUploadResponse, error)
	DeleteTripCover(ctx context.Context, req *proto.DeleteTripCoverRequest) (*proto.DeleteTripCoverResponse, error)
	GenerateInviteLink(ctx context.Context, req *proto.GenerateInviteLinkRequest) (*proto.GenerateInviteLinkResponse, error)
	JoinTripByToken(ctx context.Context, req *proto.JoinTripByTokenRequest) (*proto.JoinTripByTokenResponse, error)
	RemoveParticipant(ctx context.Context, req *proto.RemoveParticipantRequest) (*proto.RemoveParticipantResponse, error)
	LeaveTrip(ctx context.Context, req *proto.LeaveTripRequest) (*proto.LeaveTripResponse, error)
	ProcessMediaGrouping(ctx context.Context, req *proto.ProcessMediaGroupingRequest) (*proto.ProcessMediaGroupingResponse, error)
	ApplyGroupsAndProcess(ctx context.Context, req *proto.ApplyGroupsAndProcessRequest) (*proto.ApplyGroupsAndProcessResponse, error)
	GetTripReview(ctx context.Context, req *proto.GetTripReviewRequest) (*proto.GetTripReviewResponse, error)
	FinalizeTrip(ctx context.Context, req *proto.FinalizeTripRequest) (*proto.FinalizeTripResponse, error)
	PublishTrip(ctx context.Context, req *proto.PublishTripRequest) (*proto.PublishTripResponse, error)
	UpdateTripSettings(ctx context.Context, req *proto.UpdateTripSettingsRequest) (*proto.UpdateTripSettingsResponse, error)
	ListFeed(ctx context.Context, req *proto.ListFeedRequest) (*proto.ListFeedResponse, error)
	LikeTrip(ctx context.Context, req *proto.LikeTripRequest) (*proto.LikeTripResponse, error)
	DislikeTrip(ctx context.Context, req *proto.DislikeTripRequest) (*proto.DislikeTripResponse, error)
	AddToFavourites(ctx context.Context, req *proto.AddToFavouritesRequest) (*proto.AddToFavouritesResponse, error)
	RemoveFromFavourites(ctx context.Context, req *proto.RemoveFromFavouritesRequest) (*proto.RemoveFromFavouritesResponse, error)
	ListFavourites(ctx context.Context, req *proto.ListFavouritesRequest) (*proto.ListFavouritesResponse, error)
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

// CreateTrip creates a new trip (creation flow, stage 1).
// @Summary [1] Create a new trip
// @Description Creates a new trip (creation flow stage 1: init + S3 upload URLs). Requires JWT. user_id is taken from JWT.
// @Tags trip-creation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body requests.CreateTripRequest true "Trip creation payload"
// @Success 201 {object} responses.CreateTripResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/creation/start [post]
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

// GetTrip returns a single trip by ID with pins and media.
// @Summary Get trip by ID
// @Description Returns a single trip by ID with pins and media in each pin. Requires JWT. User must be a participant.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Success 200 {object} responses.GetTripResponse
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
	respondJSON(w, http.StatusOK, getTripResponseToREST(resp))
}

// UpdateTrip updates trip metadata.
// @Summary Update trip
// @Description Updates trip metadata (name, description, category, season, dates, privacy_level). Requires JWT. Any trip participant can update (ТЗ 3.2). For cover use the /cover/upload + /cover/confirm endpoints.
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
	resp, err := h.tripClient.UpdateTrip(ctx, protoReq)
	if err != nil {
		handleServiceError(w, r, err, "UpdateTrip")
		return
	}
	respondJSON(w, http.StatusOK, tripProtoToResponse(resp.GetTrip()))
}

// RequestTripCoverUpload returns a presigned PUT URL for uploading a new trip cover.
// @Summary Request presigned URL for trip cover upload
// @Description Step 1 of the cover upload flow (mirrors user avatar). Requires JWT. Any trip participant can perform this action.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.RequestTripCoverUploadRequest true "Filename and content type"
// @Success 200 {object} responses.TripCoverUploadResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/cover/upload [post]
func (h *TripHandler) RequestTripCoverUpload(w http.ResponseWriter, r *http.Request) {
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
	var req requests.RequestTripCoverUploadRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	resp, err := h.tripClient.RequestTripCoverUpload(ctx, &proto.RequestTripCoverUploadRequest{
		TripId:      tripID,
		UserId:      userID,
		Filename:    req.Filename,
		ContentType: req.ContentType,
	})
	if err != nil {
		handleServiceError(w, r, err, "RequestTripCoverUpload")
		return
	}
	respondJSON(w, http.StatusOK, responses.TripCoverUploadResponse{
		UploadURL: resp.GetUploadUrl(),
		S3Key:     resp.GetS3Key(),
	})
}

// ConfirmTripCoverUpload persists the uploaded cover s3_key.
// @Summary Confirm trip cover upload after uploading to S3
// @Description Step 2 of the cover upload flow. Requires JWT. Any trip participant can perform this action.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.ConfirmTripCoverUploadRequest true "S3 key"
// @Success 200 {object} responses.Trip
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/cover/confirm [post]
func (h *TripHandler) ConfirmTripCoverUpload(w http.ResponseWriter, r *http.Request) {
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
	var req requests.ConfirmTripCoverUploadRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	resp, err := h.tripClient.ConfirmTripCoverUpload(ctx, &proto.ConfirmTripCoverUploadRequest{
		TripId: tripID,
		UserId: userID,
		S3Key:  req.S3Key,
	})
	if err != nil {
		handleServiceError(w, r, err, "ConfirmTripCoverUpload")
		return
	}
	respondJSON(w, http.StatusOK, tripProtoToResponse(resp.GetTrip()))
}

// DeleteTripCover clears the trip cover.
// @Summary Delete trip cover
// @Description Removes the trip cover: deletes the file from S3 and clears cover_url. Requires JWT. Any trip participant can perform this action.
// @Tags trips
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Success 200 {object} responses.Trip
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/cover [delete]
func (h *TripHandler) DeleteTripCover(w http.ResponseWriter, r *http.Request) {
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
	resp, err := h.tripClient.DeleteTripCover(ctx, &proto.DeleteTripCoverRequest{
		TripId: tripID,
		UserId: userID,
	})
	if err != nil {
		handleServiceError(w, r, err, "DeleteTripCover")
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

// ProcessMediaGrouping saves media metadata and returns draft pins (tripCreationFlow stage 2).
// @Summary [2] Process media grouping
// @Tags trip-creation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.ProcessMediaGroupingRequest true "Media metadata"
// @Success 200 {object} responses.ProcessMediaGroupingResponse
// @Router /api/v1/trips/creation/{id}/media/process-grouping [post]
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
// @Summary [3] Apply groups and process
// @Tags trip-creation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.ApplyGroupsAndProcessRequest true "Draft pins and deleted media"
// @Success 202 {object} responses.ApplyGroupsAndProcessResponse
// @Router /api/v1/trips/creation/{id}/apply-groups-and-process [post]
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
// @Summary [4] Get trip review
// @Tags trip-creation
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Success 200 {object} responses.GetTripReviewResponse
// @Router /api/v1/trips/creation/{id}/review [get]
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
// @Summary [5] Finalize trip
// @Tags trip-creation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.FinalizeTripRequest true "Pin updates and media to delete"
// @Success 200 {object} responses.FinalizeTripResponse
// @Router /api/v1/trips/creation/{id}/finalize [post]
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

// PublishTrip публикует трип в общую ленту (PINZ-105, ТЗ 3.3).
// @Summary Publish trip to feed
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.PublishTripRequest true "Publish payload (publish_whole or pin_ids)"
// @Success 200 {object} responses.Trip
// @Router /api/v1/trips/{id}/publish [post]
func (h *TripHandler) PublishTrip(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	_ = userID // user_id пробрасывается в gRPC метаданных, в теле не нужен

	tripID := chi.URLParam(r, "id")
	if tripID == "" {
		respondError(w, http.StatusBadRequest, "trip id required")
		return
	}
	var req requests.PublishTripRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	protoReq := &proto.PublishTripRequest{
		TripId:       tripID,
		PublishWhole: req.PublishWhole,
		PinIds:       req.PinIDs,
	}
	resp, err := h.tripClient.PublishTrip(ctx, protoReq)
	if err != nil {
		handleServiceError(w, r, err, "PublishTrip")
		return
	}
	respondJSON(w, http.StatusOK, tripProtoToResponse(resp.GetTrip()))
}

// UpdateTripSettings updates notifications for the trip (PINZ-98, ТЗ 12.4.1).
// @Summary Update trip notification settings
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.TripSettingsRequest true "notifications_enabled"
// @Success 200 {object} responses.TripSettingsResponse
// @Router /api/v1/trips/{id}/settings [patch]
func (h *TripHandler) UpdateTripSettings(w http.ResponseWriter, r *http.Request) {
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
	var req requests.TripSettingsRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	_, err := h.tripClient.UpdateTripSettings(ctx, &proto.UpdateTripSettingsRequest{
		TripId:               tripID,
		NotificationsEnabled: req.NotificationsEnabled,
	})
	if err != nil {
		handleServiceError(w, r, err, "UpdateTripSettings")
		return
	}
	respondJSON(w, http.StatusOK, responses.TripSettingsResponse{Success: true})
}

// ListFeed returns published trips for the feed (PINZ-98).
// @Summary List feed
// @Tags feed
// @Produce json
// @Security BearerAuth
// @Param limit query int false "limit"
// @Param offset query int false "offset"
// @Param category query string false "category"
// @Param season query string false "season"
// @Param location_id query int false "location_id"
// @Param sort_by query string false "date|rating"
// @Success 200 {array} responses.FeedItem
// @Router /api/v1/feed [get]
func (h *TripHandler) ListFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	limit := int32(20)
	offset := int32(0)
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := parseInt(l); err == nil && n > 0 {
			limit = int32(n)
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := parseInt(o); err == nil && n >= 0 {
			offset = int32(n)
		}
	}
	category := r.URL.Query().Get("category")
	season := r.URL.Query().Get("season")
	sortBy := r.URL.Query().Get("sort_by")
	if sortBy == "" {
		sortBy = "date"
	}
	locationIDs := make([]int32, 0, 4)
	if lid := r.URL.Query().Get("location_id"); lid != "" {
		if n, err := parseInt(lid); err == nil {
			locationIDs = append(locationIDs, int32(n))
		}
	}
	for _, raw := range r.URL.Query()["location_ids"] {
		if n, err := parseInt(raw); err == nil {
			locationIDs = append(locationIDs, int32(n))
		}
	}
	resp, err := h.tripClient.ListFeed(ctx, &proto.ListFeedRequest{
		Limit:       limit,
		Offset:      offset,
		Category:    category,
		Season:      season,
		LocationIds: locationIDs,
		SortBy:      sortBy,
	})
	if err != nil {
		handleServiceError(w, r, err, "ListFeed")
		return
	}
	items := resp.GetItems()
	out := make([]responses.FeedItem, len(items))
	for i, item := range items {
		pins := make([]responses.FeedPin, len(item.GetPins()))
		for j, p := range item.GetPins() {
			pins[j] = responses.FeedPin{
				ID:        p.GetId(),
				Latitude:  p.GetLatitude(),
				Longitude: p.GetLongitude(),
			}
		}
		media := make([]responses.FeedMedia, len(item.GetMedia()))
		for j, m := range item.GetMedia() {
			media[j] = responses.FeedMedia{
				MediaID:   m.GetMediaId(),
				URL:       m.GetUrl(),
				MediaType: m.GetMediaType(),
			}
		}
		out[i] = responses.FeedItem{
			Trip:  tripProtoToResponse(item.GetTrip()),
			Pins:  pins,
			Media: media,
		}
	}
	respondJSON(w, http.StatusOK, out)
}

// LikeTrip adds a like to the trip (PINZ-98).
// @Summary Like trip
// @Tags trips
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Success 200 {object} responses.SuccessResponse
// @Router /api/v1/trips/{id}/like [post]
func (h *TripHandler) LikeTrip(w http.ResponseWriter, r *http.Request) {
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
	_, err := h.tripClient.LikeTrip(ctx, &proto.LikeTripRequest{TripId: tripID})
	if err != nil {
		handleServiceError(w, r, err, "LikeTrip")
		return
	}
	respondJSON(w, http.StatusOK, responses.SuccessResponse{Success: true})
}

// DislikeTrip adds a dislike to the trip (PINZ-98).
// @Summary Dislike trip
// @Tags trips
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Success 200 {object} responses.SuccessResponse
// @Router /api/v1/trips/{id}/dislike [post]
func (h *TripHandler) DislikeTrip(w http.ResponseWriter, r *http.Request) {
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
	_, err := h.tripClient.DislikeTrip(ctx, &proto.DislikeTripRequest{TripId: tripID})
	if err != nil {
		handleServiceError(w, r, err, "DislikeTrip")
		return
	}
	respondJSON(w, http.StatusOK, responses.SuccessResponse{Success: true})
}

// AddToFavourites adds the trip to user's favourites (PINZ-98).
// @Summary Add trip to favourites
// @Tags trips
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Success 200 {object} responses.SuccessResponse
// @Router /api/v1/trips/{id}/favourite [post]
func (h *TripHandler) AddToFavourites(w http.ResponseWriter, r *http.Request) {
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
	_, err := h.tripClient.AddToFavourites(ctx, &proto.AddToFavouritesRequest{TripId: tripID})
	if err != nil {
		handleServiceError(w, r, err, "AddToFavourites")
		return
	}
	respondJSON(w, http.StatusOK, responses.SuccessResponse{Success: true})
}

// RemoveFromFavourites removes the trip from user's favourites (PINZ-98).
// @Summary Remove trip from favourites
// @Tags trips
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Success 204
// @Router /api/v1/trips/{id}/favourite [delete]
func (h *TripHandler) RemoveFromFavourites(w http.ResponseWriter, r *http.Request) {
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
	_, err := h.tripClient.RemoveFromFavourites(ctx, &proto.RemoveFromFavouritesRequest{TripId: tripID})
	if err != nil {
		handleServiceError(w, r, err, "RemoveFromFavourites")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListFavourites returns the current user's favourite trips (PINZ-98).
// @Summary List favourite trips
// @Description Returns trips the user has added to favourites. Supports limit and offset query params.
// @Tags trips
// @Security BearerAuth
// @Param limit query int false "Limit (default 20, max 100)"
// @Param offset query int false "Offset (default 0)"
// @Success 200 {array} responses.Trip
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/favourites [get]
func (h *TripHandler) ListFavourites(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	limit := int32(20)
	offset := int32(0)
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := parseInt(l); err == nil && n > 0 && n <= 100 {
			limit = int32(n)
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := parseInt(o); err == nil && n >= 0 {
			offset = int32(n)
		}
	}
	resp, err := h.tripClient.ListFavourites(ctx, &proto.ListFavouritesRequest{Limit: limit, Offset: offset})
	if err != nil {
		handleServiceError(w, r, err, "ListFavourites")
		return
	}
	out := make([]responses.Trip, len(resp.GetTrips()))
	for i, t := range resp.GetTrips() {
		out[i] = tripProtoToResponse(t)
	}
	respondJSON(w, http.StatusOK, out)
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
		DislikesCount:     t.GetDislikesCount(),
		MediaCount:        t.GetMediaCount(),
		ParticipantsCount: t.GetParticipantsCount(),
		CoverURL:          t.GetCoverUrl(),
		IsPublished:   t.GetIsPublished(),
		IsGenerated:   t.GetIsGenerated(),
		CreatedAtUnix: t.GetCreatedAtUnix(),
		UpdatedAtUnix: t.GetUpdatedAtUnix(),
	}
}

func getTripResponseToREST(resp *proto.GetTripResponse) responses.GetTripResponse {
	out := responses.GetTripResponse{
		Trip: tripProtoToResponse(resp.GetTrip()),
		Pins: make([]responses.TripPin, 0, len(resp.GetPins())),
	}
	for _, p := range resp.GetPins() {
		if p == nil {
			continue
		}
		pin := responses.TripPin{
			ID:            p.GetId(),
			Name:          p.GetName(),
			Description:   p.GetDescription(),
			Category:      p.GetCategory(),
			Latitude:      p.Latitude,
			Longitude:     p.Longitude,
			StartTimeUnix: p.GetStartTimeUnix(),
			EndTimeUnix:   p.GetEndTimeUnix(),
			PrivacyLevel:  p.GetPrivacyLevel(),
			Tags:          p.GetTags(),
			Media:         make([]responses.TripPinMedia, 0, len(p.GetMedia())),
		}
		for _, m := range p.GetMedia() {
			if m == nil {
				continue
			}
			pin.Media = append(pin.Media, responses.TripPinMedia{
				MediaID:        m.GetMediaId(),
				URL:            m.GetUrl(),
				MediaType:      m.GetMediaType(),
				CapturedAtUnix: m.GetCapturedAtUnix(),
				PrivacyLevel:   m.GetPrivacyLevel(),
			})
		}
		out.Pins = append(out.Pins, pin)
	}
	return out
}
