package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"pinz/backend/api-gateway-service/internal/middleware"
	"pinz/backend/api-gateway-service/internal/requests"
	"pinz/backend/api-gateway-service/internal/responses"
	"pinz/backend/api-gateway-service/pkg/proto"
)

// CreatePinStart starts sessioned creation flow for a single pin (ТЗ 4.1, 4.6).
// @Summary Start pin creation session
// @Description Старт sessioned-флоу создания одиночного пина в READY-трипе. Создаёт сессию (UNIQUE active per trip — `412 FailedPrecondition`, если уже идёт) и выдаёт presigned PUT URLs. Дальше: PUT в S3 → /commit-upload → /process → /review → /finalize или /cancel.
// @Tags pins
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.CreatePinStartRequest true "Files to upload"
// @Success 200 {object} responses.CreatePinStartResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 412 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pins/creation/start [post]
func (h *TripHandler) CreatePinStart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if userID := middleware.UserIDFromContext(ctx); userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	if tripID == "" {
		respondError(w, http.StatusBadRequest, "trip id required")
		return
	}
	var req requests.CreatePinStartRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	resp, err := h.tripClient.CreatePinStart(ctx, &proto.CreatePinStartRequest{
		TripId: tripID,
		FilesToUpload: filesToProto(req.FilesToUpload),
	})
	if err != nil {
		handleServiceError(w, r, err, "CreatePinStart")
		return
	}
	respondJSON(w, http.StatusOK, responses.CreatePinStartResponse{
		SessionID: resp.GetSessionId(),
		UploadURLs: uploadUrlsProtoToResponse(resp.GetUploadUrls()),
	})
}

// RequestPinCreationUploadUrls returns extra presigned URLs for an active pin-creation session.
// @Summary Request more presigned URLs (pin creation)
// @Description Догрузка presigned URLs к активной сессии создания пина.
// @Tags pins
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param sid path string true "Session ID"
// @Param body body requests.RequestPinCreationUploadUrlsRequest true "Files to upload"
// @Success 200 {object} responses.RequestPinCreationUploadUrlsResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pins/creation/sessions/{sid}/upload-urls [post]
func (h *TripHandler) RequestPinCreationUploadUrls(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if userID := middleware.UserIDFromContext(ctx); userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	sessionID := chi.URLParam(r, "sid")
	if tripID == "" || sessionID == "" {
		respondError(w, http.StatusBadRequest, "trip id and session id required")
		return
	}
	var req requests.RequestPinCreationUploadUrlsRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	resp, err := h.tripClient.RequestPinCreationUploadUrls(ctx, &proto.RequestPinCreationUploadUrlsRequest{
		TripId: tripID,
		SessionId: sessionID,
		FilesToUpload: filesToProto(req.FilesToUpload),
	})
	if err != nil {
		handleServiceError(w, r, err, "RequestPinCreationUploadUrls")
		return
	}
	respondJSON(w, http.StatusOK, responses.RequestPinCreationUploadUrlsResponse{
		UploadURLs: uploadUrlsProtoToResponse(resp.GetUploadUrls()),
	})
}

// CommitPinCreationUpload registers a successful S3 upload in a pin-creation session.
// @Summary Commit pin creation upload
// @Description Фиксация s3-загрузки в активную сессию создания пина: создаётся media с pin_id=NULL и pin_creation_session_id=session.
// @Tags pins
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param sid path string true "Session ID"
// @Param body body requests.CommitPinCreationUploadRequest true "Commit payload"
// @Success 200 {object} responses.CommitPinCreationUploadResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 412 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pins/creation/sessions/{sid}/commit-upload [post]
func (h *TripHandler) CommitPinCreationUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if userID := middleware.UserIDFromContext(ctx); userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	sessionID := chi.URLParam(r, "sid")
	if tripID == "" || sessionID == "" {
		respondError(w, http.StatusBadRequest, "trip id and session id required")
		return
	}
	var req requests.CommitPinCreationUploadRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	resp, err := h.tripClient.CommitPinCreationUpload(ctx, &proto.CommitPinCreationUploadRequest{
		TripId: tripID,
		SessionId: sessionID,
		S3Key: req.S3Key,
		MediaType: req.MediaType,
		CapturedAtUnix: req.CapturedAtUnix,
		Latitude: req.Latitude,
		Longitude: req.Longitude,
	})
	if err != nil {
		handleServiceError(w, r, err, "CommitPinCreationUpload")
		return
	}
	respondJSON(w, http.StatusOK, responses.CommitPinCreationUploadResponse{
		MediaID: resp.GetMediaId(),
		MediaCountInSession: resp.GetMediaCountInSession(),
	})
}

// ProcessPinCreation runs synchronous ML-stub on session media.
// @Summary Process pin creation (ML stub)
// @Description Синхронный запуск ML-stub: хеш-дедуп (4.7.6.a), NSFW (4.7.5, заглушка), similar (4.7.7, заглушка), suggested поля пина (4.7.2.a-f), pin issues (4.7.3-4.7.4). Snapshot сохраняется в сессию.
// @Tags pins
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param sid path string true "Session ID"
// @Success 200 {object} responses.ProcessPinCreationResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pins/creation/sessions/{sid}/process [post]
func (h *TripHandler) ProcessPinCreation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if userID := middleware.UserIDFromContext(ctx); userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	sessionID := chi.URLParam(r, "sid")
	if tripID == "" || sessionID == "" {
		respondError(w, http.StatusBadRequest, "trip id and session id required")
		return
	}
	resp, err := h.tripClient.ProcessPinCreation(ctx, &proto.ProcessPinCreationRequest{
		TripId: tripID, SessionId: sessionID,
	})
	if err != nil {
		handleServiceError(w, r, err, "ProcessPinCreation")
		return
	}
	respondJSON(w, http.StatusOK, responses.ProcessPinCreationResponse{
		SessionID: resp.GetSessionId(),
		Draft: pinCreationDraftProtoToResponse(resp.GetDraft()),
		Similar: similarGroupsProtoToResponse(resp.GetSimilar()),
	})
}

// GetPinCreationReview returns the review snapshot for an active pin-creation session.
// @Summary Get pin creation review
// @Description Чтение snapshot Process: suggested-поля пина (имя/категория/теги/координаты/start-end), список новых медиа, pin issues, similar groups.
// @Tags pins
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param sid path string true "Session ID"
// @Success 200 {object} responses.GetPinCreationReviewResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pins/creation/sessions/{sid}/review [get]
func (h *TripHandler) GetPinCreationReview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if userID := middleware.UserIDFromContext(ctx); userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	sessionID := chi.URLParam(r, "sid")
	if tripID == "" || sessionID == "" {
		respondError(w, http.StatusBadRequest, "trip id and session id required")
		return
	}
	resp, err := h.tripClient.GetPinCreationReview(ctx, &proto.GetPinCreationReviewRequest{
		TripId: tripID, SessionId: sessionID,
	})
	if err != nil {
		handleServiceError(w, r, err, "GetPinCreationReview")
		return
	}
	respondJSON(w, http.StatusOK, responses.GetPinCreationReviewResponse{
		SessionID: resp.GetSessionId(),
		Draft: pinCreationDraftProtoToResponse(resp.GetDraft()),
		Similar: similarGroupsProtoToResponse(resp.GetSimilar()),
	})
}

// FinalizePinCreation creates a new pin from session media (ТЗ 4.9-4.11).
// @Summary Finalize pin creation
// @Description Создание новой записи pins с финальными полями (правки клиента поверх ML-suggested), привязка media, теги, reverse-geocoding, публикация PIN_ADDED, закрытие сессии.
// @Tags pins
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param sid path string true "Session ID"
// @Param body body requests.FinalizePinCreationRequest true "Finalize payload"
// @Success 200 {object} responses.PinResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 412 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pins/creation/sessions/{sid}/finalize [post]
func (h *TripHandler) FinalizePinCreation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if userID := middleware.UserIDFromContext(ctx); userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	sessionID := chi.URLParam(r, "sid")
	if tripID == "" || sessionID == "" {
		respondError(w, http.StatusBadRequest, "trip id and session id required")
		return
	}
	var req requests.FinalizePinCreationRequest
	_ = decodeJSONBody(r, &req)
	pbReq := &proto.FinalizePinCreationRequest{
		TripId: tripID,
		SessionId: sessionID,
		Name: req.Name,
		Description: req.Description,
		Category: req.Category,
		Latitude: req.Latitude,
		Longitude: req.Longitude,
		StartTimeUnix: req.StartTimeUnix,
		EndTimeUnix: req.EndTimeUnix,
		Tags: req.Tags,
		TagsSet: req.TagsSet,
		MediaToDelete: req.MediaToDelete,
	}
	resp, err := h.tripClient.FinalizePinCreation(ctx, pbReq)
	if err != nil {
		handleServiceError(w, r, err, "FinalizePinCreation")
		return
	}
	respondJSON(w, http.StatusOK, responses.PinResponse{Pin: tripPinProtoToResponse(resp.GetPin())})
}

// CancelPinCreation cancels an active pin-creation session and removes orphan media.
// @Summary Cancel pin creation
// @Description Удаляет orphan media сессии (pin_id=NULL, pin_creation_session_id=session) с S3 cleanup и закрывает сессию.
// @Tags pins
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param sid path string true "Session ID"
// @Success 200 {object} responses.CancelPinCreationResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pins/creation/sessions/{sid}/cancel [post]
func (h *TripHandler) CancelPinCreation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if userID := middleware.UserIDFromContext(ctx); userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	sessionID := chi.URLParam(r, "sid")
	if tripID == "" || sessionID == "" {
		respondError(w, http.StatusBadRequest, "trip id and session id required")
		return
	}
	resp, err := h.tripClient.CancelPinCreation(ctx, &proto.CancelPinCreationRequest{
		TripId: tripID, SessionId: sessionID,
	})
	if err != nil {
		handleServiceError(w, r, err, "CancelPinCreation")
		return
	}
	respondJSON(w, http.StatusOK, responses.CancelPinCreationResponse{Status: resp.GetStatus()})
}

// =============================================================================
// helpers
// =============================================================================

func pinCreationDraftProtoToResponse(d *proto.PinCreationDraft) responses.PinCreationDraft {
	if d == nil {
		return responses.PinCreationDraft{}
	}
	out := responses.PinCreationDraft{
		SuggestedName: d.GetSuggestedName(),
		SuggestedCategory: d.GetSuggestedCategory(),
		SuggestedTags: d.GetSuggestedTags(),
		SuggestedLatitude: d.SuggestedLatitude,
		SuggestedLongitude: d.SuggestedLongitude,
		SuggestedStartTimeUnix: d.SuggestedStartTimeUnix,
		SuggestedEndTimeUnix: d.SuggestedEndTimeUnix,
		PinIssues: d.GetPinIssues(),
		NSFWMediaIDs: d.GetNsfwMediaIds(),
		DedupedMediaIDs: d.GetDedupedMediaIds(),
	}
	for _, m := range d.GetMedia() {
		out.Media = append(out.Media, responses.ReviewPinMedia{
			MediaID: m.GetMediaId(),
			URL: m.GetUrl(),
			PrivacyLevel: m.GetPrivacyLevel(),
		})
	}
	return out
}
