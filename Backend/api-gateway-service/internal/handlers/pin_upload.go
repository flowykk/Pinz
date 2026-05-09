package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"pinz/backend/api-gateway-service/internal/middleware"
	"pinz/backend/api-gateway-service/internal/requests"
	"pinz/backend/api-gateway-service/internal/responses"
	"pinz/backend/api-gateway-service/pkg/proto"
)

// PinUploadStart starts a unified pin-upload session.
// @Summary Start pin upload session
// @Description Унифицированная сессия загрузки медиа. target_pin_id null → создание нового пина (UNIQUE per trip), заполнен → добавление медиа в существующий пин (UNIQUE per pin). Дальше: PUT в S3 → /commit-upload → /process → ws + /review → /finalize или /cancel.
// @Tags pins
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param body body requests.PinUploadStartRequest true "Files + optional target_pin_id"
// @Success 200 {object} responses.PinUploadStartResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 409 {object} responses.ErrorResponse
// @Failure 412 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pin-uploads/start [post]
func (h *TripHandler) PinUploadStart(w http.ResponseWriter, r *http.Request) {
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
	var req requests.PinUploadStartRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	pbReq := &proto.PinUploadStartRequest{
		TripId:        tripID,
		FilesToUpload: filesToProto(req.FilesToUpload),
	}
	if req.TargetPinID != nil && *req.TargetPinID != "" {
		v := *req.TargetPinID
		pbReq.TargetPinId = &v
	}
	resp, err := h.tripClient.PinUploadStart(ctx, pbReq)
	if err != nil {
		handleServiceError(w, r, err, "PinUploadStart")
		return
	}
	respondJSON(w, http.StatusOK, responses.PinUploadStartResponse{
		SessionID:  resp.GetSessionId(),
		UploadURLs: uploadUrlsProtoToResponse(resp.GetUploadUrls()),
	})
}

// RequestPinUploadUrls returns extra presigned URLs for an active pin-upload session.
// @Summary Request more presigned URLs (pin upload)
// @Tags pins
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param sid path string true "Session ID"
// @Param body body requests.RequestPinUploadUrlsRequest true "Files to upload"
// @Success 200 {object} responses.RequestPinUploadUrlsResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pin-uploads/{sid}/upload-urls [post]
func (h *TripHandler) RequestPinUploadUrls(w http.ResponseWriter, r *http.Request) {
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
	var req requests.RequestPinUploadUrlsRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	resp, err := h.tripClient.RequestPinUploadUrls(ctx, &proto.RequestPinUploadUrlsRequest{
		TripId:        tripID,
		SessionId:     sessionID,
		FilesToUpload: filesToProto(req.FilesToUpload),
	})
	if err != nil {
		handleServiceError(w, r, err, "RequestPinUploadUrls")
		return
	}
	respondJSON(w, http.StatusOK, responses.RequestPinUploadUrlsResponse{
		UploadURLs: uploadUrlsProtoToResponse(resp.GetUploadUrls()),
	})
}

// CommitPinUpload registers a successful S3 upload in an active pin-upload session.
// @Summary Commit pin upload
// @Tags pins
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param sid path string true "Session ID"
// @Param body body requests.CommitPinUploadRequest true "Commit payload"
// @Success 200 {object} responses.CommitPinUploadResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 412 {object} responses.ErrorResponse
// @Failure 422 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pin-uploads/{sid}/commit-upload [post]
func (h *TripHandler) CommitPinUpload(w http.ResponseWriter, r *http.Request) {
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
	var req requests.CommitPinUploadRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	resp, err := h.tripClient.CommitPinUpload(ctx, &proto.CommitPinUploadRequest{
		TripId:         tripID,
		SessionId:      sessionID,
		S3Key:          req.S3Key,
		MediaType:      req.MediaType,
		CapturedAtUnix: req.CapturedAtUnix,
		Latitude:       req.Latitude,
		Longitude:      req.Longitude,
	})
	if err != nil {
		handleServiceError(w, r, err, "CommitPinUpload")
		return
	}
	respondJSON(w, http.StatusOK, responses.CommitPinUploadResponse{
		MediaID:             resp.GetMediaId(),
		MediaCountInSession: resp.GetMediaCountInSession(),
	})
}

// ProcessPinUpload kicks off async ML processing for a pin-upload session.
// @Summary Process pin upload (async)
// @Description 202 + processing_status="PROCESSING". По завершении воркер шлёт WS PIN_UPLOAD_PROCESSING_COMPLETED.
// @Tags pins
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param sid path string true "Session ID"
// @Success 202 {object} responses.ProcessPinUploadResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 409 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pin-uploads/{sid}/process [post]
func (h *TripHandler) ProcessPinUpload(w http.ResponseWriter, r *http.Request) {
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
	resp, err := h.tripClient.ProcessPinUpload(ctx, &proto.ProcessPinUploadRequest{
		TripId: tripID, SessionId: sessionID,
	})
	if err != nil {
		handleServiceError(w, r, err, "ProcessPinUpload")
		return
	}
	respondJSON(w, http.StatusAccepted, responses.ProcessPinUploadResponse{
		SessionID:        resp.GetSessionId(),
		ProcessingStatus: resp.GetProcessingStatus(),
	})
}

// GetPinUploadReview returns the review snapshot.
// @Summary Get pin upload review
// @Tags pins
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param sid path string true "Session ID"
// @Success 200 {object} responses.GetPinUploadReviewResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pin-uploads/{sid}/review [get]
func (h *TripHandler) GetPinUploadReview(w http.ResponseWriter, r *http.Request) {
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
	resp, err := h.tripClient.GetPinUploadReview(ctx, &proto.GetPinUploadReviewRequest{
		TripId: tripID, SessionId: sessionID,
	})
	if err != nil {
		handleServiceError(w, r, err, "GetPinUploadReview")
		return
	}
	respondJSON(w, http.StatusOK, responses.GetPinUploadReviewResponse{
		SessionID:        resp.GetSessionId(),
		ProcessingStatus: resp.GetProcessingStatus(),
		Draft:            pinUploadDraftProtoToResponse(resp.GetDraft()),
		Similar:          similarGroupsProtoToResponse(resp.GetSimilar()),
	})
}

// FinalizePinUpload — финал сессии.
// @Summary Finalize pin upload
// @Tags pins
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param sid path string true "Session ID"
// @Param body body requests.FinalizePinUploadRequest true "Finalize payload"
// @Success 200 {object} responses.PinResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 409 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pin-uploads/{sid}/finalize [post]
func (h *TripHandler) FinalizePinUpload(w http.ResponseWriter, r *http.Request) {
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
	var req requests.FinalizePinUploadRequest
	_ = decodeJSONBody(r, &req)
	pbReq := &proto.FinalizePinUploadRequest{
		TripId:        tripID,
		SessionId:     sessionID,
		Name:          req.Name,
		Description:   req.Description,
		Category:      req.Category,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		StartTimeUnix: req.StartTimeUnix,
		EndTimeUnix:   req.EndTimeUnix,
		Tags:          req.Tags,
		TagsSet:       req.TagsSet,
		MediaToDelete: req.MediaToDelete,
	}
	resp, err := h.tripClient.FinalizePinUpload(ctx, pbReq)
	if err != nil {
		handleServiceError(w, r, err, "FinalizePinUpload")
		return
	}
	respondJSON(w, http.StatusOK, responses.PinResponse{Pin: tripPinProtoToResponse(resp.GetPin())})
}

// CancelPinUpload — отмена сессии.
// @Summary Cancel pin upload
// @Tags pins
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param sid path string true "Session ID"
// @Success 200 {object} responses.CancelPinUploadResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pin-uploads/{sid}/cancel [post]
func (h *TripHandler) CancelPinUpload(w http.ResponseWriter, r *http.Request) {
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
	resp, err := h.tripClient.CancelPinUpload(ctx, &proto.CancelPinUploadRequest{
		TripId: tripID, SessionId: sessionID,
	})
	if err != nil {
		handleServiceError(w, r, err, "CancelPinUpload")
		return
	}
	respondJSON(w, http.StatusOK, responses.CancelPinUploadResponse{Status: resp.GetStatus()})
}

// =============================================================================
// helpers
// =============================================================================

func pinUploadDraftProtoToResponse(d *proto.PinUploadDraft) responses.PinUploadDraft {
	if d == nil {
		return responses.PinUploadDraft{}
	}
	out := responses.PinUploadDraft{
		PinIssues:       d.GetPinIssues(),
		NSFWMediaIDs:    d.GetNsfwMediaIds(),
		DedupedMediaIDs: d.GetDedupedMediaIds(),
	}
	if s := d.GetSuggested(); s != nil {
		out.Suggested = &responses.PinSuggestedFields{
			Name:          s.GetName(),
			Category:      s.GetCategory(),
			Tags:          s.GetTags(),
			Latitude:      s.Latitude,
			Longitude:     s.Longitude,
			StartTimeUnix: s.StartTimeUnix,
			EndTimeUnix:   s.EndTimeUnix,
		}
	}
	for _, m := range d.GetMedia() {
		out.Media = append(out.Media, responses.ReviewPinMedia{
			MediaID:      m.GetMediaId(),
			URL:          m.GetUrl(),
			PrivacyLevel: m.GetPrivacyLevel(),
		})
	}
	return out
}
