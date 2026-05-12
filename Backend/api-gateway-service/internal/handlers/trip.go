package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
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
	tripClient        TripClient
	authEnricher      AuthProfileEnricher
	// tripShareLinkBase — базовый URL для формирования share-ссылки:
	// в ответ Trip пишется "{base}/{id}". Берётся из env TRIP_SHARE_LINK_BASE,
	// дефолт задаётся в DI. Пустое значение → share_url остаётся пустым.
	tripShareLinkBase string
}

// AuthProfileEnricher — минимальный контракт auth-client'а для обогащения ответов
// публичными профилями (N2). В тестах заменяется stub'ом.
type AuthProfileEnricher interface {
	GetUsersProfiles(ctx context.Context, req *proto.GetUsersProfilesRequest) (*proto.GetUsersProfilesResponse, error)
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
	AddMediaStart(ctx context.Context, req *proto.AddMediaStartRequest) (*proto.AddMediaStartResponse, error)
	AddMediaRequestUploadUrls(ctx context.Context, req *proto.AddMediaRequestUploadUrlsRequest) (*proto.AddMediaRequestUploadUrlsResponse, error)
	AddMediaCommitUpload(ctx context.Context, req *proto.AddMediaCommitUploadRequest) (*proto.AddMediaCommitUploadResponse, error)
	AddMediaGetSessionMedia(ctx context.Context, req *proto.AddMediaGetSessionMediaRequest) (*proto.AddMediaGetSessionMediaResponse, error)
	AddMediaProcessGrouping(ctx context.Context, req *proto.AddMediaProcessGroupingRequest) (*proto.AddMediaProcessGroupingResponse, error)
	AddMediaGetGrouping(ctx context.Context, req *proto.AddMediaGetGroupingRequest) (*proto.AddMediaGetGroupingResponse, error)
	AddMediaApplyGroupsAndProcess(ctx context.Context, req *proto.AddMediaApplyGroupsAndProcessRequest) (*proto.AddMediaApplyGroupsAndProcessResponse, error)
	AddMediaGetReview(ctx context.Context, req *proto.AddMediaGetReviewRequest) (*proto.AddMediaGetReviewResponse, error)
	AddMediaConfirm(ctx context.Context, req *proto.AddMediaConfirmRequest) (*proto.AddMediaConfirmResponse, error)
	AddMediaCancel(ctx context.Context, req *proto.AddMediaCancelRequest) (*proto.AddMediaCancelResponse, error)
	AddMediaTakeover(ctx context.Context, req *proto.AddMediaTakeoverRequest) (*proto.AddMediaTakeoverResponse, error)
	StartBattle(ctx context.Context, req *proto.StartBattleRequest) (*proto.StartBattleResponse, error)
	SubmitBattleResult(ctx context.Context, req *proto.SubmitBattleResultRequest) (*proto.SubmitBattleResultResponse, error)
	GetBestMemories(ctx context.Context, req *proto.GetBestMemoriesRequest) (*proto.GetBestMemoriesResponse, error)
	SearchPins(ctx context.Context, req *proto.SearchPinsRequest) (*proto.SearchPinsResponse, error)
	UpsertTripPrivacy(ctx context.Context, req *proto.UpsertTripPrivacyRequest) (*proto.UpsertPrivacyResponse, error)
	UpsertPinPrivacy(ctx context.Context, req *proto.UpsertPinPrivacyRequest) (*proto.UpsertPrivacyResponse, error)
	UpsertMediaPrivacy(ctx context.Context, req *proto.UpsertMediaPrivacyRequest) (*proto.UpsertPrivacyResponse, error)
	// Pin RUD
	GetPin(ctx context.Context, req *proto.GetPinRequest) (*proto.GetPinResponse, error)
	UpdatePin(ctx context.Context, req *proto.UpdatePinRequest) (*proto.UpdatePinResponse, error)
	DeletePin(ctx context.Context, req *proto.DeletePinRequest) (*proto.DeletePinResponse, error)
	RemoveMediaFromPin(ctx context.Context, req *proto.RemoveMediaFromPinRequest) (*proto.RemoveMediaFromPinResponse, error)
	// Pin upload (унифицированный creation/addition)
	PinUploadStart(ctx context.Context, req *proto.PinUploadStartRequest) (*proto.PinUploadStartResponse, error)
	RequestPinUploadUrls(ctx context.Context, req *proto.RequestPinUploadUrlsRequest) (*proto.RequestPinUploadUrlsResponse, error)
	CommitPinUpload(ctx context.Context, req *proto.CommitPinUploadRequest) (*proto.CommitPinUploadResponse, error)
	ProcessPinUpload(ctx context.Context, req *proto.ProcessPinUploadRequest) (*proto.ProcessPinUploadResponse, error)
	GetPinUploadReview(ctx context.Context, req *proto.GetPinUploadReviewRequest) (*proto.GetPinUploadReviewResponse, error)
	FinalizePinUpload(ctx context.Context, req *proto.FinalizePinUploadRequest) (*proto.FinalizePinUploadResponse, error)
	CancelPinUpload(ctx context.Context, req *proto.CancelPinUploadRequest) (*proto.CancelPinUploadResponse, error)
	GetRecommendations(ctx context.Context, req *proto.GetRecommendationsRequest) (*proto.GetRecommendationsResponse, error)
	SaveRecommendation(ctx context.Context, req *proto.SaveRecommendationRequest) (*proto.SaveRecommendationResponse, error)
}

func NewTripHandler(tripClient TripClient, authEnricher AuthProfileEnricher, tripShareLinkBase string) *TripHandler {
	return &TripHandler{tripClient: tripClient, authEnricher: authEnricher, tripShareLinkBase: tripShareLinkBase}
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
		Limit: limit,
		Offset: offset,
	})
	if err != nil {
		handleServiceError(w, r, err, "ListUserTrips")
		return
	}
	out := make([]responses.Trip, len(resp.GetTrips()))
	for i, t := range resp.GetTrips() {
		out[i] = h.tripProtoToResponse(t)
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
		OwnerUserId: userID,
		Name: req.Name,
		Description: req.Description,
		Category: req.Category,
		Season: req.Season,
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
		TripID: resp.GetTripId(),
		Status: resp.GetStatus(),
		UploadURLs: urls,
	})
}

// GetTrip returns a single trip by ID with pins and media.
// @Summary Get trip by ID
// @Description Returns a single trip by ID with pins and media in each pin. Requires JWT.
// @Description Доступ: участник трипа или владелец трипа в избранных получает полный ответ с participants/current_user_settings.
// @Description Любой залогиненный пользователь может открыть опубликованный трип по share-ссылке; в этом случае возвращаются только публичные пины (выбранные при публикации) с публичными медиа, без participants/settings.
// @Description Если трип не опубликован, а пользователь не участник и не имеет трип в избранных — 403.
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
	respondJSON(w, http.StatusOK, h.getTripResponseToREST(ctx, resp))
}

// UpdateTrip updates trip metadata.
// @Summary Update trip
// @Description Updates trip metadata (name, description, category, season, dates). Requires JWT. Any trip participant can update. Обложка — через /cover/upload + /cover/confirm. Приватность — через PUT /trips/{id}/privacy (per-user).
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
	respondJSON(w, http.StatusOK, h.tripProtoToResponse(resp.GetTrip()))
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
		TripId: tripID,
		UserId: userID,
		Filename: req.Filename,
		ContentType: req.ContentType,
	})
	if err != nil {
		handleServiceError(w, r, err, "RequestTripCoverUpload")
		return
	}
	respondJSON(w, http.StatusOK, responses.TripCoverUploadResponse{
		UploadURL: resp.GetUploadUrl(),
		S3Key: resp.GetS3Key(),
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
		S3Key: req.S3Key,
	})
	if err != nil {
		handleServiceError(w, r, err, "ConfirmTripCoverUpload")
		return
	}
	respondJSON(w, http.StatusOK, h.tripProtoToResponse(resp.GetTrip()))
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
	respondJSON(w, http.StatusOK, h.tripProtoToResponse(resp.GetTrip()))
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
		InviteLinkID: resp.GetInviteLinkId(),
		Token: resp.GetToken(),
		InviteURL: resp.GetInviteUrl(),
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
		TripID: resp.GetTripId(),
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
		Success: resp.GetSuccess(),
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
		TripID: resp.GetTripId(),
		Status: resp.GetStatus(),
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
		TripId: tripID,
		DraftPins: draftPins,
		DeletedMediaIds: req.DeletedMediaIDs,
	})
	if err != nil {
		handleServiceError(w, r, err, "ApplyGroupsAndProcess")
		return
	}
	respondJSON(w, http.StatusAccepted, responses.ApplyGroupsAndProcessResponse{
		Message: resp.GetMessage(),
		Status: resp.GetStatus(),
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

// PublishTrip публикует трип в общую ленту.
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
		TripId: tripID,
		PublishWhole: req.PublishWhole,
		PinIds: req.PinIDs,
	}
	resp, err := h.tripClient.PublishTrip(ctx, protoReq)
	if err != nil {
		handleServiceError(w, r, err, "PublishTrip")
		return
	}
	respondJSON(w, http.StatusOK, h.tripProtoToResponse(resp.GetTrip()))
}

// UpdateTripPrivacy sets the caller's per-user privacy level on a trip.
// @Summary Set per-user trip privacy
// @Description Текущий пользователь выставляет свой уровень приватности на путешествии. Эффективный privacy_level пересчитывается по AggregatePrivacyLevel и возвращается в ответе. Только участник трипа.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.UpsertPrivacyRequest true "privacy_level (Public|Private)"
// @Success 200 {object} responses.PrivacyResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 412 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/privacy [put]
func (h *TripHandler) UpdateTripPrivacy(w http.ResponseWriter, r *http.Request) {
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
	var req requests.UpsertPrivacyRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	resp, err := h.tripClient.UpsertTripPrivacy(ctx, &proto.UpsertTripPrivacyRequest{
		TripId: tripID,
		PrivacyLevel: req.PrivacyLevel,
	})
	if err != nil {
		handleServiceError(w, r, err, "UpsertTripPrivacy")
		return
	}
	respondJSON(w, http.StatusOK, responses.PrivacyResponse{PrivacyLevel: resp.GetEffectivePrivacyLevel()})
}

// UpdatePinPrivacy sets the caller's per-user privacy level on a pin.
// @Summary Set per-user pin privacy
// @Description Текущий пользователь выставляет свой уровень приватности на пине. Эффективный privacy_level пересчитывается по AggregatePrivacyLevel и возвращается в ответе. Только участник трипа.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param pin_id path string true "Pin ID"
// @Param body body requests.UpsertPrivacyRequest true "privacy_level (Public|Private)"
// @Success 200 {object} responses.PrivacyResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 412 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pins/{pin_id}/privacy [put]
func (h *TripHandler) UpdatePinPrivacy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	pinID := chi.URLParam(r, "pin_id")
	if tripID == "" || pinID == "" {
		respondError(w, http.StatusBadRequest, "trip id and pin id required")
		return
	}
	var req requests.UpsertPrivacyRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	resp, err := h.tripClient.UpsertPinPrivacy(ctx, &proto.UpsertPinPrivacyRequest{
		TripId: tripID,
		PinId: pinID,
		PrivacyLevel: req.PrivacyLevel,
	})
	if err != nil {
		handleServiceError(w, r, err, "UpsertPinPrivacy")
		return
	}
	respondJSON(w, http.StatusOK, responses.PrivacyResponse{PrivacyLevel: resp.GetEffectivePrivacyLevel()})
}

// UpdateMediaPrivacy sets the caller's per-user privacy level on a media.
// @Summary Set per-user media privacy
// @Description Текущий пользователь выставляет свой уровень приватности на медиа. Эффективный privacy_level пересчитывается по AggregatePrivacyLevel и возвращается в ответе. Только участник трипа.
// @Tags trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param media_id path string true "Media ID"
// @Param body body requests.UpsertPrivacyRequest true "privacy_level (Public|Private)"
// @Success 200 {object} responses.PrivacyResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 412 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/media/{media_id}/privacy [put]
func (h *TripHandler) UpdateMediaPrivacy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	mediaID := chi.URLParam(r, "media_id")
	if tripID == "" || mediaID == "" {
		respondError(w, http.StatusBadRequest, "trip id and media id required")
		return
	}
	var req requests.UpsertPrivacyRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	resp, err := h.tripClient.UpsertMediaPrivacy(ctx, &proto.UpsertMediaPrivacyRequest{
		TripId: tripID,
		MediaId: mediaID,
		PrivacyLevel: req.PrivacyLevel,
	})
	if err != nil {
		handleServiceError(w, r, err, "UpsertMediaPrivacy")
		return
	}
	respondJSON(w, http.StatusOK, responses.PrivacyResponse{PrivacyLevel: resp.GetEffectivePrivacyLevel()})
}

// UpdateTripSettings updates notifications for the trip.
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
		TripId: tripID,
		NotificationsEnabled: req.NotificationsEnabled,
	})
	if err != nil {
		handleServiceError(w, r, err, "UpdateTripSettings")
		return
	}
	respondJSON(w, http.StatusOK, responses.TripSettingsResponse{Success: true})
}

// ListFeed returns published trips for the feed.
// @Summary List feed
// @Tags feed
// @Produce json
// @Security BearerAuth
// @Param limit query int false "limit"
// @Param offset query int false "offset"
// @Param category query string false "category"
// @Param season query string false "season"
// @Param city query string false "city name in lower-case (mutually exclusive with country, city wins)"
// @Param country query string false "country name in lower-case (mutually exclusive with city)"
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
	city := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("city")))
	country := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("country")))
	resp, err := h.tripClient.ListFeed(ctx, &proto.ListFeedRequest{
		Limit: limit,
		Offset: offset,
		Category: category,
		Season: season,
		City: city,
		Country: country,
		SortBy: sortBy,
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
			pinMedia := make([]responses.FeedMedia, len(p.GetMedia()))
			for k, m := range p.GetMedia() {
				pinMedia[k] = responses.FeedMedia{
					MediaID: m.GetMediaId(),
					URL: m.GetUrl(),
					MediaType: m.GetMediaType(),
				}
			}
			pins[j] = responses.FeedPin{
				ID: p.GetId(),
				Latitude: p.GetLatitude(),
				Longitude: p.GetLongitude(),
				Media: pinMedia,
			}
		}
		media := make([]responses.FeedMedia, len(item.GetMedia()))
		for j, m := range item.GetMedia() {
			media[j] = responses.FeedMedia{
				MediaID: m.GetMediaId(),
				URL: m.GetUrl(),
				MediaType: m.GetMediaType(),
			}
		}
		out[i] = responses.FeedItem{
			Trip: h.tripProtoToResponse(item.GetTrip()),
			Pins: pins,
			Media: media,
			IsLiked: item.GetIsLiked(),
			IsDisliked: item.GetIsDisliked(),
			IsSaved: item.GetIsSaved(),
		}
	}
	respondJSON(w, http.StatusOK, out)
}

// LikeTrip adds a like to the trip.
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

// DislikeTrip adds a dislike to the trip.
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

// AddToFavourites adds the trip to user's favourites.
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

// RemoveFromFavourites removes the trip from user's favourites.
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

// ListFavourites returns the current user's favourite trips.
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
		out[i] = h.tripProtoToResponse(t)
	}
	respondJSON(w, http.StatusOK, out)
}

// SearchPins searches pins by text query across trips where the authenticated user is a participant.
// @Summary Search pins by query
// @Description Text search over pin name, description and tags within trips where the user participates. Requires JWT.
// @Tags pins
// @Produce json
// @Security BearerAuth
// @Param q query string true "search query (1..128 chars)"
// @Param limit query int false "limit (1..100, default 20)"
// @Param offset query int false "offset (>=0, default 0)"
// @Success 200 {array} responses.TripPin
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/pins/search [get]
func (h *TripHandler) SearchPins(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	query := r.URL.Query().Get("q")
	if query == "" {
		respondError(w, http.StatusBadRequest, "query parameter q is required")
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
	resp, err := h.tripClient.SearchPins(ctx, &proto.SearchPinsRequest{
		Query: query,
		Limit: limit,
		Offset: offset,
	})
	if err != nil {
		handleServiceError(w, r, err, "SearchPins")
		return
	}
	out := make([]responses.TripPin, 0, len(resp.GetPins()))
	for _, p := range resp.GetPins() {
		if p == nil {
			continue
		}
		out = append(out, tripPinProtoToResponse(p))
	}
	respondJSON(w, http.StatusOK, out)
}

func (h *TripHandler) tripProtoToResponse(t *proto.Trip) responses.Trip {
	if t == nil {
		return responses.Trip{}
	}
	out := responses.Trip{
		ID: t.GetId(),
		OwnerUserID: t.GetOwnerUserId(),
		Name: t.GetName(),
		Description: t.GetDescription(),
		Category: t.GetCategory(),
		Season: t.GetSeason(),
		Status: t.GetStatus(),
		PrivacyLevel: t.GetPrivacyLevel(),
		StartDateUnix: t.GetStartDateUnix(),
		EndDateUnix: t.GetEndDateUnix(),
		LikesCount: t.GetLikesCount(),
		DislikesCount: t.GetDislikesCount(),
		MediaCount: t.GetMediaCount(),
		ParticipantsCount: t.GetParticipantsCount(),
		CoverURL: t.GetCoverUrl(),
		IsPublished: t.GetIsPublished(),
		IsGenerated: t.GetIsGenerated(),
		CreatedAtUnix: t.GetCreatedAtUnix(),
		UpdatedAtUnix: t.GetUpdatedAtUnix(),
	}
	if h.tripShareLinkBase != "" && t.GetId() != "" {
		out.ShareURL = h.tripShareLinkBase + "/" + t.GetId()
	}
	return out
}

func (h *TripHandler) getTripResponseToREST(ctx context.Context, resp *proto.GetTripResponse) responses.GetTripResponse {
	out := responses.GetTripResponse{
		Trip: h.tripProtoToResponse(resp.GetTrip()),
		Pins: make([]responses.TripPin, 0, len(resp.GetPins())),
		Participants: make([]responses.TripParticipant, 0, len(resp.GetParticipants())),
	}
	for _, p := range resp.GetPins() {
		if p == nil {
			continue
		}
		out.Pins = append(out.Pins, tripPinProtoToResponse(p))
	}
	// Один batched enrichProfiles на участников + initiator add-media сессии (N2):
	// собираем все user_id заранее, идём в auth один раз, переиспользуем map ниже.
	userIDs := make([]string, 0, len(resp.GetParticipants())+1)
	for _, p := range resp.GetParticipants() {
		if p == nil || p.GetUserId() == "" {
			continue
		}
		userIDs = append(userIDs, p.GetUserId())
	}
	if active := resp.GetActiveAddMediaSession(); active != nil {
		if uid := active.GetCurrentInitiatorUserId(); uid != "" {
			userIDs = append(userIDs, uid)
		}
	}
	profiles := h.enrichProfiles(ctx, userIDs)
	for _, p := range resp.GetParticipants() {
		if p == nil {
			continue
		}
		part := responses.TripParticipant{
			UserID: p.GetUserId(),
			PrivacyLevel: p.GetPrivacyLevel(),
			Role: p.GetRole(),
		}
		if profile, ok := profiles[p.GetUserId()]; ok {
			part.Username = profile.Username
			part.AvatarURL = profile.AvatarURL
		}
		out.Participants = append(out.Participants, part)
	}
	if cs := resp.GetCurrentUserSettings(); cs != nil {
		out.CurrentUserSettings = responses.TripSettings{
			NotificationsEnabled: cs.GetNotificationsEnabled(),
			PrivacyLevel: cs.GetPrivacyLevel(),
		}
	}
	if active := resp.GetActiveAddMediaSession(); active != nil {
		rest := &responses.ActiveAddMediaSession{
			SessionID:           active.GetSessionId(),
			InitiatorAssignedAt: unixToRFC3339(active.GetInitiatorAssignedAtUnix()),
			TakeoverAvailableAt: unixToRFC3339(active.GetTakeoverAvailableAtUnix()),
			MediaCountInSession: active.GetMediaCountInSession(),
		}
		if uid := active.GetCurrentInitiatorUserId(); uid != "" {
			if p, ok := profiles[uid]; ok {
				rest.CurrentInitiator = &p
			} else {
				rest.CurrentInitiator = &responses.PublicUserProfile{UserID: uid}
			}
		}
		out.ActiveAddMediaSession = rest
	}
	return out
}

// enrichProfiles — N2: один batched вызов auth.GetUsersProfiles на набор user_id'ов.
// Пустой набор или nil enricher — возвращает пустую map; ошибка auth не падает,
// просто логируется и клиент получает profile без username/avatar (UserID сохранён).
func (h *TripHandler) enrichProfiles(ctx context.Context, userIDs []string) map[string]responses.PublicUserProfile {
	if len(userIDs) == 0 || h.authEnricher == nil {
		return nil
	}
	// Дедупликация — один и тот же user_id может встретиться несколько раз в ответе.
	seen := make(map[string]struct{}, len(userIDs))
	ids := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}
	resp, err := h.authEnricher.GetUsersProfiles(ctx, &proto.GetUsersProfilesRequest{UserIds: ids})
	if err != nil {
		slog.WarnContext(ctx, "auth.GetUsersProfiles failed; returning raw user ids", "count", len(ids), "error", err)
		return nil
	}
	out := make(map[string]responses.PublicUserProfile, len(resp.GetProfiles()))
	for _, p := range resp.GetProfiles() {
		if p == nil {
			continue
		}
		out[p.GetUserId()] = responses.PublicUserProfile{
			UserID:    p.GetUserId(),
			Username:  p.GetUsername(),
			AvatarURL: p.GetAvatarUrl(),
		}
	}
	return out
}

func tripPinProtoToResponse(p *proto.TripPin) responses.TripPin {
	pin := responses.TripPin{
		ID: p.GetId(),
		TripID: p.GetTripId(),
		Name: p.GetName(),
		Description: p.GetDescription(),
		Category: p.GetCategory(),
		Latitude: p.Latitude,
		Longitude: p.Longitude,
		StartTimeUnix: p.GetStartTimeUnix(),
		EndTimeUnix: p.GetEndTimeUnix(),
		PrivacyLevel: p.GetPrivacyLevel(),
		Tags: p.GetTags(),
		Media: make([]responses.TripPinMedia, 0, len(p.GetMedia())),
	}
	for _, m := range p.GetMedia() {
		if m == nil {
			continue
		}
		pin.Media = append(pin.Media, responses.TripPinMedia{
			MediaID: m.GetMediaId(),
			URL: m.GetUrl(),
			MediaType: m.GetMediaType(),
			CapturedAtUnix: m.GetCapturedAtUnix(),
			PrivacyLevel: m.GetPrivacyLevel(),
		})
	}
	return pin
}

// AddMediaStart starts a session for adding media to an existing READY trip.
// @Summary [1] Start add-media session
// @Tags trip-add-media
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.AddMediaStartRequest true "Files to upload"
// @Success 200 {object} responses.AddMediaStartResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/media/add/start [post]
func (h *TripHandler) AddMediaStart(w http.ResponseWriter, r *http.Request) {
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
	var req requests.AddMediaStartRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	protoFiles := make([]*proto.FileToUpload, 0, len(req.FilesToUpload))
	for _, f := range req.FilesToUpload {
		protoFiles = append(protoFiles, &proto.FileToUpload{ClientId: f.ClientID, ContentType: f.ContentType})
	}
	resp, err := h.tripClient.AddMediaStart(ctx, &proto.AddMediaStartRequest{
		TripId: tripID,
		FilesToUpload: protoFiles,
	})
	if err != nil {
		handleServiceError(w, r, err, "AddMediaStart")
		return
	}
	urls := make([]responses.UploadURL, len(resp.GetUploadUrls()))
	for i, u := range resp.GetUploadUrls() {
		urls[i] = responses.UploadURL{ClientID: u.GetClientId(), S3Key: u.GetS3Key(), URL: u.GetUrl()}
	}
	respondJSON(w, http.StatusOK, responses.AddMediaStartResponse{
		SessionID: resp.GetSessionId(),
		Status: resp.GetStatus(),
		UploadURLs: urls,
		Joined: resp.GetJoined(),
	})
}

// AddMediaProcessGrouping clusters new media using existing pins as seeds.
// @Summary [2] Process grouping for add-media
// @Tags trip-add-media
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.AddMediaProcessGroupingRequest true "Session id and media metadata"
// @Success 200 {object} responses.AddMediaProcessGroupingResponse
// @Router /api/v1/trips/{id}/media/add/process-grouping [post]
func (h *TripHandler) AddMediaProcessGrouping(w http.ResponseWriter, r *http.Request) {
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
	var req requests.AddMediaProcessGroupingRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	// media[] больше не передаётся — сервер берёт из БД (commit-upload).
	// Флаг add_more=true откатывает GROUPING_REVIEW → UPLOADING.
	resp, err := h.tripClient.AddMediaProcessGrouping(ctx, &proto.AddMediaProcessGroupingRequest{
		TripId: tripID,
		SessionId: req.SessionID,
		AddMore: req.AddMore,
	})
	if err != nil {
		handleServiceError(w, r, err, "AddMediaProcessGrouping")
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
	respondJSON(w, http.StatusOK, responses.AddMediaProcessGroupingResponse{
		TripID: resp.GetTripId(),
		SessionID: resp.GetSessionId(),
		Status: resp.GetStatus(),
		DraftPins: draftPins,
		ExistingMediaIDs: resp.GetExistingMediaIds(),
	})
}

// AddMediaApplyGroupsAndProcess applies user grouping for add-media and starts ML processing.
// @Summary [3] Apply groups for add-media
// @Tags trip-add-media
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.AddMediaApplyGroupsAndProcessRequest true "Draft pins, deleted media, session id"
// @Success 202 {object} responses.AddMediaApplyGroupsAndProcessResponse
// @Router /api/v1/trips/{id}/media/add/apply-groups-and-process [post]
func (h *TripHandler) AddMediaApplyGroupsAndProcess(w http.ResponseWriter, r *http.Request) {
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
	var req requests.AddMediaApplyGroupsAndProcessRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	draftPins := make([]*proto.DraftPinInput, 0, len(req.DraftPins))
	for _, dp := range req.DraftPins {
		draftPins = append(draftPins, &proto.DraftPinInput{DraftPinId: dp.DraftPinID, MediaIds: dp.MediaIDs})
	}
	resp, err := h.tripClient.AddMediaApplyGroupsAndProcess(ctx, &proto.AddMediaApplyGroupsAndProcessRequest{
		TripId: tripID,
		SessionId: req.SessionID,
		DraftPins: draftPins,
		DeletedMediaIds: req.DeletedMediaIDs,
	})
	if err != nil {
		handleServiceError(w, r, err, "AddMediaApplyGroupsAndProcess")
		return
	}
	respondJSON(w, http.StatusAccepted, responses.AddMediaApplyGroupsAndProcessResponse{
		Message: resp.GetMessage(),
		Status: resp.GetStatus(),
	})
}

// AddMediaRequestUploadUrls — presigned URLs для догрузки файлов в активную сессию.
// @Summary Request upload URLs for active add-media session
// @Tags trip-add-media
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.AddMediaRequestUploadUrlsRequest true "Session id and files to upload"
// @Success 200 {object} responses.AddMediaRequestUploadUrlsResponse
// @Router /api/v1/trips/{id}/media/add/request-upload-urls [post]
func (h *TripHandler) AddMediaRequestUploadUrls(w http.ResponseWriter, r *http.Request) {
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
	var req requests.AddMediaRequestUploadUrlsRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	files := make([]*proto.FileToUpload, 0, len(req.FilesToUpload))
	for _, f := range req.FilesToUpload {
		files = append(files, &proto.FileToUpload{ClientId: f.ClientID, ContentType: f.ContentType})
	}
	resp, err := h.tripClient.AddMediaRequestUploadUrls(ctx, &proto.AddMediaRequestUploadUrlsRequest{
		TripId: tripID,
		SessionId: req.SessionID,
		FilesToUpload: files,
	})
	if err != nil {
		handleServiceError(w, r, err, "AddMediaRequestUploadUrls")
		return
	}
	urls := make([]responses.UploadURL, 0, len(resp.GetUploadUrls()))
	for _, u := range resp.GetUploadUrls() {
		urls = append(urls, responses.UploadURL{ClientID: u.GetClientId(), S3Key: u.GetS3Key(), URL: u.GetUrl()})
	}
	respondJSON(w, http.StatusOK, responses.AddMediaRequestUploadUrlsResponse{UploadURLs: urls})
}

// AddMediaCommitUpload — регистрация факта успешного PUT в S3.
// Клиент вызывает после каждого файла; сервер создаёт media entry и публикует
// WS ADD_MEDIA_PROGRESS остальным participant'ам.
// @Summary Commit uploaded file to add-media session
// @Tags trip-add-media
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.AddMediaCommitUploadRequest true "Uploaded file metadata"
// @Success 200 {object} responses.AddMediaCommitUploadResponse
// @Router /api/v1/trips/{id}/media/add/commit-upload [post]
func (h *TripHandler) AddMediaCommitUpload(w http.ResponseWriter, r *http.Request) {
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
	var req requests.AddMediaCommitUploadRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	protoReq := &proto.AddMediaCommitUploadRequest{
		TripId: tripID,
		SessionId: req.SessionID,
		S3Key: req.S3Key,
		MediaType: req.MediaType,
	}
	if req.CapturedAt != "" {
		if t, err := time.Parse(time.RFC3339, req.CapturedAt); err == nil {
			u := t.Unix()
			protoReq.CapturedAtUnix = &u
		}
	}
	if req.Latitude != nil {
		protoReq.Latitude = req.Latitude
	}
	if req.Longitude != nil {
		protoReq.Longitude = req.Longitude
	}
	resp, err := h.tripClient.AddMediaCommitUpload(ctx, protoReq)
	if err != nil {
		handleServiceError(w, r, err, "AddMediaCommitUpload")
		return
	}
	respondJSON(w, http.StatusOK, responses.AddMediaCommitUploadResponse{
		MediaID: resp.GetMediaId(),
		MediaCountInSession: resp.GetMediaCountInSession(),
		RemainingSlots: resp.GetRemainingSlots(),
	})
}

// AddMediaGetSessionMedia — снимок медиа активной сессии.
// @Summary Session media snapshot
// @Tags trip-add-media
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param session_id query string true "Add-media session id"
// @Success 200 {object} responses.AddMediaGetSessionMediaResponse
// @Router /api/v1/trips/{id}/media/add/session-media [get]
func (h *TripHandler) AddMediaGetSessionMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	sessionID := r.URL.Query().Get("session_id")
	if tripID == "" || sessionID == "" {
		respondError(w, http.StatusBadRequest, "trip id and session_id required")
		return
	}
	resp, err := h.tripClient.AddMediaGetSessionMedia(ctx, &proto.AddMediaGetSessionMediaRequest{
		TripId: tripID,
		SessionId: sessionID,
	})
	if err != nil {
		handleServiceError(w, r, err, "AddMediaGetSessionMedia")
		return
	}
	media := make([]responses.SessionMediaEntry, 0, len(resp.GetMedia()))
	for _, m := range resp.GetMedia() {
		media = append(media, responses.SessionMediaEntry{
			MediaID: m.GetMediaId(),
			URL: m.GetUrl(),
			Type: m.GetType(),
			ActorUserID: m.GetActorUserId(),
			UploadedAt: unixToRFC3339(m.GetUploadedAtUnix()),
		})
	}
	respondJSON(w, http.StatusOK, responses.AddMediaGetSessionMediaResponse{
		SessionID: resp.GetSessionId(),
		Media: media,
		MediaCountInSession: resp.GetMediaCountInSession(),
	})
}

// AddMediaGetGrouping — снимок draft_pins для экрана GROUPING_REVIEW.
// @Summary Add-media grouping snapshot
// @Tags trip-add-media
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param session_id query string true "Add-media session id"
// @Success 200 {object} responses.AddMediaGetGroupingResponse
// @Router /api/v1/trips/{id}/media/add/grouping [get]
func (h *TripHandler) AddMediaGetGrouping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	sessionID := r.URL.Query().Get("session_id")
	if tripID == "" || sessionID == "" {
		respondError(w, http.StatusBadRequest, "trip id and session_id required")
		return
	}
	resp, err := h.tripClient.AddMediaGetGrouping(ctx, &proto.AddMediaGetGroupingRequest{
		TripId: tripID,
		SessionId: sessionID,
	})
	if err != nil {
		handleServiceError(w, r, err, "AddMediaGetGrouping")
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
	respondJSON(w, http.StatusOK, responses.AddMediaGetGroupingResponse{
		TripID: resp.GetTripId(),
		SessionID: resp.GetSessionId(),
		DraftPins: draftPins,
		ExistingMediaIDs: resp.GetExistingMediaIds(),
	})
}

// AddMediaGetReview — финальное ревью add-media.
// @Summary Add-media final review
// @Tags trip-add-media
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param session_id query string true "Add-media session id"
// @Success 200 {object} responses.AddMediaGetReviewResponse
// @Router /api/v1/trips/{id}/media/add/review [get]
func (h *TripHandler) AddMediaGetReview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	sessionID := r.URL.Query().Get("session_id")
	if tripID == "" || sessionID == "" {
		respondError(w, http.StatusBadRequest, "trip id and session_id required")
		return
	}
	resp, err := h.tripClient.AddMediaGetReview(ctx, &proto.AddMediaGetReviewRequest{
		TripId: tripID,
		SessionId: sessionID,
	})
	if err != nil {
		handleServiceError(w, r, err, "AddMediaGetReview")
		return
	}
	pins := make([]responses.TripPin, 0, len(resp.GetPins()))
	for _, p := range resp.GetPins() {
		if p == nil {
			continue
		}
		pins = append(pins, tripPinProtoToResponse(p))
	}
	out := responses.AddMediaGetReviewResponse{
		TripID:              resp.GetTripId(),
		SessionID:           resp.GetSessionId(),
		Pins:                pins,
		NewPinIDs:           resp.GetNewPinIds(),
		ProtectedMediaIDs:   resp.GetProtectedMediaIds(),
		TakeoverAvailableAt: unixToRFC3339(resp.GetTakeoverAvailableAtUnix()),
		CanEdit:             resp.GetCanEdit(),
	}
	// N2: обогащение ведущего публичным профилем, чтобы клиент мог сразу
	// показать «Алиса завершает ревью», а не резолвить user_id отдельным запросом.
	if uid := resp.GetCurrentInitiatorUserId(); uid != "" {
		profiles := h.enrichProfiles(ctx, []string{uid})
		if p, ok := profiles[uid]; ok {
			out.CurrentInitiator = &p
		} else {
			out.CurrentInitiator = &responses.PublicUserProfile{UserID: uid}
		}
	}
	respondJSON(w, http.StatusOK, out)
}

// AddMediaConfirm — финализация add-media сессии, трип → READY.
// @Summary Confirm add-media session
// @Tags trip-add-media
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.AddMediaConfirmRequest true "Session id"
// @Success 200 {object} responses.AddMediaConfirmResponse
// @Router /api/v1/trips/{id}/media/add/confirm [post]
func (h *TripHandler) AddMediaConfirm(w http.ResponseWriter, r *http.Request) {
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
	var req requests.AddMediaConfirmRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	pinUpdates := make([]*proto.PinUpdate, 0, len(req.PinUpdates))
	for _, pu := range req.PinUpdates {
		pinUpdates = append(pinUpdates, &proto.PinUpdate{
			PinId: pu.PinID, Name: pu.Name, Latitude: pu.Latitude, Longitude: pu.Longitude,
		})
	}
	resp, err := h.tripClient.AddMediaConfirm(ctx, &proto.AddMediaConfirmRequest{
		TripId:        tripID,
		SessionId:     req.SessionID,
		PinUpdates:    pinUpdates,
		MediaToDelete: req.MediaToDelete,
	})
	if err != nil {
		handleServiceError(w, r, err, "AddMediaConfirm")
		return
	}
	respondJSON(w, http.StatusOK, responses.AddMediaConfirmResponse{
		Status: resp.GetStatus(),
		AlreadyConfirmed: resp.GetAlreadyConfirmed(),
	})
}

// AddMediaCancel — отмена add-media сессии.
// @Summary Cancel add-media session
// @Tags trip-add-media
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.AddMediaCancelRequest true "Session id"
// @Success 200 {object} responses.AddMediaCancelResponse
// @Router /api/v1/trips/{id}/media/add/cancel [post]
func (h *TripHandler) AddMediaCancel(w http.ResponseWriter, r *http.Request) {
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
	var req requests.AddMediaCancelRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	resp, err := h.tripClient.AddMediaCancel(ctx, &proto.AddMediaCancelRequest{
		TripId: tripID,
		SessionId: req.SessionID,
	})
	if err != nil {
		handleServiceError(w, r, err, "AddMediaCancel")
		return
	}
	respondJSON(w, http.StatusOK, responses.AddMediaCancelResponse{Status: resp.GetStatus()})
}

// AddMediaTakeover — explicit передача роли ведущего после истечения часа.
// @Summary Take over add-media session leadership
// @Description Идемпотентен: если caller уже ведущий, вернёт 200 без изменений. Если час не прошёл — 403 NOT_INITIATOR.
// @Tags trip-add-media
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.AddMediaTakeoverRequest true "Session id"
// @Success 200 {object} responses.AddMediaTakeoverResponse
// @Router /api/v1/trips/{id}/media/add/takeover [post]
func (h *TripHandler) AddMediaTakeover(w http.ResponseWriter, r *http.Request) {
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
	var req requests.AddMediaTakeoverRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	resp, err := h.tripClient.AddMediaTakeover(ctx, &proto.AddMediaTakeoverRequest{
		TripId:    tripID,
		SessionId: req.SessionID,
	})
	if err != nil {
		handleServiceError(w, r, err, "AddMediaTakeover")
		return
	}
	out := responses.AddMediaTakeoverResponse{
		IsInitiator:         resp.GetIsInitiator(),
		TakeoverAvailableAt: unixToRFC3339(resp.GetTakeoverAvailableAtUnix()),
	}
	if uid := resp.GetCurrentInitiatorUserId(); uid != "" {
		profiles := h.enrichProfiles(ctx, []string{uid})
		if p, ok := profiles[uid]; ok {
			out.CurrentInitiator = &p
		} else {
			out.CurrentInitiator = &responses.PublicUserProfile{UserID: uid}
		}
	}
	respondJSON(w, http.StatusOK, out)
}

// StartBattle starts a new photo battle for the trip.
// @Summary Start photo battle
// @Description Picks 8 random media from the trip and starts a battle session. Returns 412 if the trip has fewer than 8 available media.
// @Tags trip-battles
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Success 200 {object} responses.StartBattleResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 412 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/battles [post]
func (h *TripHandler) StartBattle(w http.ResponseWriter, r *http.Request) {
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
	resp, err := h.tripClient.StartBattle(ctx, &proto.StartBattleRequest{TripId: tripID})
	if err != nil {
		handleServiceError(w, r, err, "StartBattle")
		return
	}
	media := make([]responses.BattleMedia, 0, len(resp.GetMedia()))
	for _, m := range resp.GetMedia() {
		media = append(media, responses.BattleMedia{
			MediaID: m.GetMediaId(),
			URL: m.GetUrl(),
			MediaType: m.GetMediaType(),
		})
	}
	respondJSON(w, http.StatusOK, responses.StartBattleResponse{
		BattleID: resp.GetBattleId(),
		Media: media,
	})
}

// SubmitBattleResult finalizes a battle with the chosen winner.
// @Summary Submit battle winner
// @Description Records the final winner of a photo battle; increments media battle_rating by 1. Can be called once per battle.
// @Tags trip-battles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param battle_id path string true "Battle ID"
// @Param body body requests.SubmitBattleResultRequest true "Winner media id"
// @Success 200 {object} responses.SubmitBattleResultResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 412 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/battles/{battle_id}/result [post]
func (h *TripHandler) SubmitBattleResult(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	battleID := chi.URLParam(r, "battle_id")
	if battleID == "" {
		respondError(w, http.StatusBadRequest, "battle_id required")
		return
	}
	var req requests.SubmitBattleResultRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.WinnerMediaID == "" {
		respondError(w, http.StatusBadRequest, "winner_media_id required")
		return
	}
	resp, err := h.tripClient.SubmitBattleResult(ctx, &proto.SubmitBattleResultRequest{
		BattleId: battleID,
		WinnerMediaId: req.WinnerMediaID,
	})
	if err != nil {
		handleServiceError(w, r, err, "SubmitBattleResult")
		return
	}
	respondJSON(w, http.StatusOK, responses.SubmitBattleResultResponse{
		NewBattleRating: resp.GetNewBattleRating(),
	})
}

// GetBestMemories returns trip media with battle_rating > 0 for story-mode.
// @Summary Get best memories (story-mode)
// @Description Returns media of the trip with battle_rating > 0, sorted by rating DESC. Empty array when the trip has no winners yet.
// @Tags trip-battles
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Success 200 {object} responses.GetBestMemoriesResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/best-memories [get]
func (h *TripHandler) GetBestMemories(w http.ResponseWriter, r *http.Request) {
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
	resp, err := h.tripClient.GetBestMemories(ctx, &proto.GetBestMemoriesRequest{TripId: tripID})
	if err != nil {
		handleServiceError(w, r, err, "GetBestMemories")
		return
	}
	media := make([]responses.BestMemory, 0, len(resp.GetMedia()))
	for _, m := range resp.GetMedia() {
		media = append(media, responses.BestMemory{
			MediaID: m.GetMediaId(),
			URL: m.GetUrl(),
			MediaType: m.GetMediaType(),
			BattleRating: m.GetBattleRating(),
			CapturedAtUnix: m.GetCapturedAtUnix(),
			PinName: m.GetPinName(),
		})
	}
	respondJSON(w, http.StatusOK, responses.GetBestMemoriesResponse{Media: media})
}

// GetRecommendations returns a popular-places map for the requested city or country.
// @Summary Get recommendations
// @Tags recommendations
// @Produce json
// @Security BearerAuth
// @Param city query string false "city name in lower-case (mutually exclusive with country)"
// @Param country query string false "country name in lower-case (mutually exclusive with city)"
// @Param category query string false "trip category filter "
// @Param season query string false "trip season filter "
// @Success 200 {object} responses.GetRecommendationsResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Router /api/v1/recommendations [get]
func (h *TripHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if middleware.UserIDFromContext(ctx) == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	city := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("city")))
	country := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("country")))
	if (city == "") == (country == "") {
		respondError(w, http.StatusBadRequest, "exactly one of city or country must be provided")
		return
	}
	resp, err := h.tripClient.GetRecommendations(ctx, &proto.GetRecommendationsRequest{
		City: city,
		Country: country,
		Category: r.URL.Query().Get("category"),
		Season: r.URL.Query().Get("season"),
	})
	if err != nil {
		handleServiceError(w, r, err, "GetRecommendations")
		return
	}
	respondJSON(w, http.StatusOK, responses.GetRecommendationsResponse{Map: h.recommendedMapProtoToResponse(resp.GetMap())})
}

// SaveRecommendation persists the popular-places map as a generated trip in the user's favourites.
// @Summary Save recommendation as trip
// @Tags recommendations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body responses.SaveRecommendationRequest true "snapshot_token (fast-path) или pin_ids+city/country (fallback)"
// @Success 200 {object} responses.SaveRecommendationResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Router /api/v1/recommendations/save [post]
func (h *TripHandler) SaveRecommendation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if middleware.UserIDFromContext(ctx) == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body responses.SaveRecommendationRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	body.City = strings.ToLower(strings.TrimSpace(body.City))
	body.Country = strings.ToLower(strings.TrimSpace(body.Country))
	if body.SnapshotToken == "" {
		if (body.City == "") == (body.Country == "") {
			respondError(w, http.StatusBadRequest, "exactly one of city or country must be provided")
			return
		}
		if len(body.PinIDs) == 0 {
			respondError(w, http.StatusBadRequest, "pin_ids is required when snapshot_token is missing")
			return
		}
	}
	resp, err := h.tripClient.SaveRecommendation(ctx, &proto.SaveRecommendationRequest{
		SnapshotToken: body.SnapshotToken,
		PinIds: body.PinIDs,
		City: body.City,
		Country: body.Country,
		Category: body.Category,
		Season: body.Season,
	})
	if err != nil {
		handleServiceError(w, r, err, "SaveRecommendation")
		return
	}
	respondJSON(w, http.StatusOK, responses.SaveRecommendationResponse{Trip: h.tripProtoToResponse(resp.GetTrip())})
}

func (h *TripHandler) recommendedMapProtoToResponse(m *proto.RecommendedMap) responses.RecommendedMap {
	if m == nil {
		return responses.RecommendedMap{}
	}
	pins := make([]responses.RecommendedPin, len(m.GetPins()))
	for i, p := range m.GetPins() {
		media := make([]responses.FeedMedia, len(p.GetMedia()))
		for j, fm := range p.GetMedia() {
			media[j] = responses.FeedMedia{
				MediaID: fm.GetMediaId(),
				URL: fm.GetUrl(),
				MediaType: fm.GetMediaType(),
			}
		}
		pins[i] = responses.RecommendedPin{
			ID: p.GetId(),
			TripID: p.GetTripId(),
			Latitude: p.GetLatitude(),
			Longitude: p.GetLongitude(),
			Name: p.GetName(),
			Description: p.GetDescription(),
			Category: p.GetCategory(),
			LocationName: p.GetLocationName(),
			MediaCount: p.GetMediaCount(),
			Media: media,
		}
	}
	media := make([]responses.FeedMedia, len(m.GetMedia()))
	for i, fm := range m.GetMedia() {
		media[i] = responses.FeedMedia{
			MediaID: fm.GetMediaId(),
			URL: fm.GetUrl(),
			MediaType: fm.GetMediaType(),
		}
	}
	return responses.RecommendedMap{
		RegionName: m.GetRegionName(),
		RegionType: m.GetRegionType(),
		Pins: pins,
		Trip: h.tripProtoToResponse(m.GetTrip()),
		Media: media,
		SnapshotToken: m.GetSnapshotToken(),
	}
}
