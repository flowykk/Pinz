package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"pinz/backend/api-gateway-service/internal/middleware"
	"pinz/backend/api-gateway-service/internal/requests"
	"pinz/backend/api-gateway-service/internal/responses"
	"pinz/backend/api-gateway-service/pkg/proto"
)

// GetPin returns full pin info (media + tags + privacy_level).
// @Summary Get pin
// @Description Получить пин со всеми полями (ТЗ 4.3). Доступ — participant трипа или favourite-юзер. Если пин скрыт для caller'а через pin_hidden_by_user (soft-delete-for-self, ТЗ 4.5.2) — 404.
// @Tags pins
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param pin_id path string true "Pin ID"
// @Success 200 {object} responses.PinResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pins/{pin_id} [get]
func (h *TripHandler) GetPin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if userID := middleware.UserIDFromContext(ctx); userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	pinID := chi.URLParam(r, "pin_id")
	if tripID == "" || pinID == "" {
		respondError(w, http.StatusBadRequest, "trip id and pin id required")
		return
	}
	resp, err := h.tripClient.GetPin(ctx, &proto.GetPinRequest{TripId: tripID, PinId: pinID})
	if err != nil {
		handleServiceError(w, r, err, "GetPin")
		return
	}
	respondJSON(w, http.StatusOK, responses.PinResponse{Pin: tripPinProtoToResponse(resp.GetPin())})
}

// UpdatePin updates pin metadata (ТЗ 4.2.1, 4.2.4-4.2.9).
// @Summary Update pin
// @Description Изменение полей пина на READY-трипе: name/description/category/координаты/start-end/tags. Privacy — отдельной ручкой PUT /pins/{id}/privacy. Любой participant.
// @Tags pins
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param pin_id path string true "Pin ID"
// @Param body body requests.UpdatePinRequest true "Update payload"
// @Success 200 {object} responses.PinResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 412 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pins/{pin_id} [patch]
func (h *TripHandler) UpdatePin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if userID := middleware.UserIDFromContext(ctx); userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	pinID := chi.URLParam(r, "pin_id")
	if tripID == "" || pinID == "" {
		respondError(w, http.StatusBadRequest, "trip id and pin id required")
		return
	}
	var req requests.UpdatePinRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	pbReq := &proto.UpdatePinRequest{
		TripId: tripID,
		PinId: pinID,
		Name: req.Name,
		Description: req.Description,
		Category: req.Category,
		Latitude: req.Latitude,
		Longitude: req.Longitude,
		StartTimeUnix: req.StartTimeUnix,
		EndTimeUnix: req.EndTimeUnix,
		Tags: req.Tags,
		TagsSet: req.TagsSet,
	}
	resp, err := h.tripClient.UpdatePin(ctx, pbReq)
	if err != nil {
		handleServiceError(w, r, err, "UpdatePin")
		return
	}
	respondJSON(w, http.StatusOK, responses.PinResponse{Pin: tripPinProtoToResponse(resp.GetPin())})
}

// DeletePin deletes a pin (ТЗ 4.5).
// @Summary Delete pin
// @Description Удаление пина. Если трип в избранном у других пользователей — soft-delete-for-self через pin_hidden_by_user (ТЗ 4.5.2); иначе full delete с каскадом media и S3 cleanup (ТЗ 4.5.1). Любой participant. Защита: запрет при активной pin_media_addition_session.
// @Tags pins
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param pin_id path string true "Pin ID"
// @Success 200 {object} responses.DeletePinResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 412 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pins/{pin_id} [delete]
func (h *TripHandler) DeletePin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if userID := middleware.UserIDFromContext(ctx); userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	pinID := chi.URLParam(r, "pin_id")
	if tripID == "" || pinID == "" {
		respondError(w, http.StatusBadRequest, "trip id and pin id required")
		return
	}
	resp, err := h.tripClient.DeletePin(ctx, &proto.DeletePinRequest{TripId: tripID, PinId: pinID})
	if err != nil {
		handleServiceError(w, r, err, "DeletePin")
		return
	}
	respondJSON(w, http.StatusOK, responses.DeletePinResponse{DeletionMode: resp.GetDeletionMode()})
}

// AddMediaToPinStart starts a sessioned add-media-to-pin flow (ТЗ 4.2.2 + 4.12-4.14).
// @Summary Start add-media-to-pin session
// @Description Старт sessioned-флоу добавления медиа в существующий пин: создаёт сессию (UNIQUE per-pin) и выдаёт presigned PUT URLs. Дальше клиент пушит файлы в S3 → /commit-upload → /process → /review → /finalize или /cancel.
// @Tags pins
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param pin_id path string true "Pin ID"
// @Param body body requests.AddMediaToPinStartRequest true "Files to upload"
// @Success 200 {object} responses.AddMediaToPinStartResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 412 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pins/{pin_id}/media/sessions/start [post]
func (h *TripHandler) AddMediaToPinStart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if userID := middleware.UserIDFromContext(ctx); userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	pinID := chi.URLParam(r, "pin_id")
	if tripID == "" || pinID == "" {
		respondError(w, http.StatusBadRequest, "trip id and pin id required")
		return
	}
	var req requests.AddMediaToPinStartRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	pbReq := &proto.AddMediaToPinStartRequest{
		TripId: tripID,
		PinId: pinID,
		FilesToUpload: filesToProto(req.FilesToUpload),
	}
	resp, err := h.tripClient.AddMediaToPinStart(ctx, pbReq)
	if err != nil {
		handleServiceError(w, r, err, "AddMediaToPinStart")
		return
	}
	respondJSON(w, http.StatusOK, responses.AddMediaToPinStartResponse{
		SessionID: resp.GetSessionId(),
		UploadURLs: uploadUrlsProtoToResponse(resp.GetUploadUrls()),
	})
}

// RequestPinMediaUploadUrls returns extra presigned URLs for an active pin-media session.
// @Summary Request more presigned URLs (pin add-media)
// @Description Догрузка presigned URLs к активной сессии добавления медиа в пин.
// @Tags pins
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param pin_id path string true "Pin ID"
// @Param sid path string true "Session ID"
// @Param body body requests.RequestPinMediaUploadUrlsRequest true "Files to upload"
// @Success 200 {object} responses.RequestPinMediaUploadUrlsResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pins/{pin_id}/media/sessions/{sid}/upload-urls [post]
func (h *TripHandler) RequestPinMediaUploadUrls(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if userID := middleware.UserIDFromContext(ctx); userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	pinID := chi.URLParam(r, "pin_id")
	sessionID := chi.URLParam(r, "sid")
	if tripID == "" || pinID == "" || sessionID == "" {
		respondError(w, http.StatusBadRequest, "trip id, pin id, session id required")
		return
	}
	var req requests.RequestPinMediaUploadUrlsRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	pbReq := &proto.RequestPinMediaUploadUrlsRequest{
		TripId: tripID,
		PinId: pinID,
		SessionId: sessionID,
		FilesToUpload: filesToProto(req.FilesToUpload),
	}
	resp, err := h.tripClient.RequestPinMediaUploadUrls(ctx, pbReq)
	if err != nil {
		handleServiceError(w, r, err, "RequestPinMediaUploadUrls")
		return
	}
	respondJSON(w, http.StatusOK, responses.RequestPinMediaUploadUrlsResponse{
		UploadURLs: uploadUrlsProtoToResponse(resp.GetUploadUrls()),
	})
}

// CommitPinMediaUpload registers a successful S3 upload in an active pin-media session.
// @Summary Commit pin media upload
// @Description Фиксация s3-загрузки в активную сессию добавления медиа в пин: создаётся media с pin_id=NULL и pin_addition_session_id=session.
// @Tags pins
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param pin_id path string true "Pin ID"
// @Param sid path string true "Session ID"
// @Param body body requests.CommitPinMediaUploadRequest true "Commit payload"
// @Success 200 {object} responses.CommitPinMediaUploadResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 412 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pins/{pin_id}/media/sessions/{sid}/commit-upload [post]
func (h *TripHandler) CommitPinMediaUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if userID := middleware.UserIDFromContext(ctx); userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	pinID := chi.URLParam(r, "pin_id")
	sessionID := chi.URLParam(r, "sid")
	if tripID == "" || pinID == "" || sessionID == "" {
		respondError(w, http.StatusBadRequest, "trip id, pin id, session id required")
		return
	}
	var req requests.CommitPinMediaUploadRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	pbReq := &proto.CommitPinMediaUploadRequest{
		TripId: tripID,
		PinId: pinID,
		SessionId: sessionID,
		S3Key: req.S3Key,
		MediaType: req.MediaType,
		CapturedAtUnix: req.CapturedAtUnix,
		Latitude: req.Latitude,
		Longitude: req.Longitude,
	}
	resp, err := h.tripClient.CommitPinMediaUpload(ctx, pbReq)
	if err != nil {
		handleServiceError(w, r, err, "CommitPinMediaUpload")
		return
	}
	respondJSON(w, http.StatusOK, responses.CommitPinMediaUploadResponse{
		MediaID: resp.GetMediaId(),
		MediaCountInSession: resp.GetMediaCountInSession(),
	})
}

// ProcessPinMediaAddition runs synchronous ML-stub on session media (NSFW/dedup/similar).
// @Summary Process pin media addition (ML stub)
// @Description Синхронный запуск ML-stub: хеш-дедуп (4.7.6.a), NSFW (4.7.5), similar (4.7.7), pin issues (4.7.3-4.7.4). Snapshot сохраняется в сессию для последующего GetReview/Finalize.
// @Tags pins
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param pin_id path string true "Pin ID"
// @Param sid path string true "Session ID"
// @Success 200 {object} responses.ProcessPinMediaAdditionResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pins/{pin_id}/media/sessions/{sid}/process [post]
func (h *TripHandler) ProcessPinMediaAddition(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if userID := middleware.UserIDFromContext(ctx); userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	pinID := chi.URLParam(r, "pin_id")
	sessionID := chi.URLParam(r, "sid")
	if tripID == "" || pinID == "" || sessionID == "" {
		respondError(w, http.StatusBadRequest, "trip id, pin id, session id required")
		return
	}
	resp, err := h.tripClient.ProcessPinMediaAddition(ctx, &proto.ProcessPinMediaAdditionRequest{
		TripId: tripID, PinId: pinID, SessionId: sessionID,
	})
	if err != nil {
		handleServiceError(w, r, err, "ProcessPinMediaAddition")
		return
	}
	respondJSON(w, http.StatusOK, responses.ProcessPinMediaAdditionResponse{
		SessionID: resp.GetSessionId(),
		Draft: pinAdditionDraftProtoToResponse(resp.GetDraft()),
		Similar: similarGroupsProtoToResponse(resp.GetSimilar()),
	})
}

// GetPinMediaAdditionReview returns the review snapshot for an active pin-media session.
// @Summary Get pin media addition review
// @Description Чтение snapshot Process: список новых медиа (presigned URLs), pin issues, similar groups. Если Process ещё не вызывался — поля пустые.
// @Tags pins
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param pin_id path string true "Pin ID"
// @Param sid path string true "Session ID"
// @Success 200 {object} responses.GetPinMediaAdditionReviewResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pins/{pin_id}/media/sessions/{sid}/review [get]
func (h *TripHandler) GetPinMediaAdditionReview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if userID := middleware.UserIDFromContext(ctx); userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	pinID := chi.URLParam(r, "pin_id")
	sessionID := chi.URLParam(r, "sid")
	if tripID == "" || pinID == "" || sessionID == "" {
		respondError(w, http.StatusBadRequest, "trip id, pin id, session id required")
		return
	}
	resp, err := h.tripClient.GetPinMediaAdditionReview(ctx, &proto.GetPinMediaAdditionReviewRequest{
		TripId: tripID, PinId: pinID, SessionId: sessionID,
	})
	if err != nil {
		handleServiceError(w, r, err, "GetPinMediaAdditionReview")
		return
	}
	respondJSON(w, http.StatusOK, responses.GetPinMediaAdditionReviewResponse{
		SessionID: resp.GetSessionId(),
		Draft: pinAdditionDraftProtoToResponse(resp.GetDraft()),
		Similar: similarGroupsProtoToResponse(resp.GetSimilar()),
	})
}

// FinalizePinMediaAddition applies media_to_delete, attaches remaining media to pin, closes session.
// @Summary Finalize pin media addition
// @Description Применение media_to_delete (orphan-cleanup), привязка оставшихся медиа к пину, пересчёт start/end/lat/lon, reverse-geocoding (если у пина появились координаты), закрытие сессии.
// @Tags pins
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param pin_id path string true "Pin ID"
// @Param sid path string true "Session ID"
// @Param body body requests.FinalizePinMediaAdditionRequest true "Finalize payload"
// @Success 200 {object} responses.PinResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pins/{pin_id}/media/sessions/{sid}/finalize [post]
func (h *TripHandler) FinalizePinMediaAddition(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if userID := middleware.UserIDFromContext(ctx); userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	pinID := chi.URLParam(r, "pin_id")
	sessionID := chi.URLParam(r, "sid")
	if tripID == "" || pinID == "" || sessionID == "" {
		respondError(w, http.StatusBadRequest, "trip id, pin id, session id required")
		return
	}
	var req requests.FinalizePinMediaAdditionRequest
	_ = decodeJSONBody(r, &req)
	resp, err := h.tripClient.FinalizePinMediaAddition(ctx, &proto.FinalizePinMediaAdditionRequest{
		TripId: tripID, PinId: pinID, SessionId: sessionID, MediaToDelete: req.MediaToDelete,
	})
	if err != nil {
		handleServiceError(w, r, err, "FinalizePinMediaAddition")
		return
	}
	respondJSON(w, http.StatusOK, responses.PinResponse{Pin: tripPinProtoToResponse(resp.GetPin())})
}

// CancelPinMediaAddition cancels the session and cleans up orphan media.
// @Summary Cancel pin media addition
// @Description Удаляет orphan media сессии (pin_id=NULL, pin_addition_session_id=session) с S3 cleanup и закрывает сессию. Идемпотентен.
// @Tags pins
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param pin_id path string true "Pin ID"
// @Param sid path string true "Session ID"
// @Success 200 {object} responses.CancelPinMediaAdditionResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pins/{pin_id}/media/sessions/{sid}/cancel [post]
func (h *TripHandler) CancelPinMediaAddition(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if userID := middleware.UserIDFromContext(ctx); userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	pinID := chi.URLParam(r, "pin_id")
	sessionID := chi.URLParam(r, "sid")
	if tripID == "" || pinID == "" || sessionID == "" {
		respondError(w, http.StatusBadRequest, "trip id, pin id, session id required")
		return
	}
	resp, err := h.tripClient.CancelPinMediaAddition(ctx, &proto.CancelPinMediaAdditionRequest{
		TripId: tripID, PinId: pinID, SessionId: sessionID,
	})
	if err != nil {
		handleServiceError(w, r, err, "CancelPinMediaAddition")
		return
	}
	respondJSON(w, http.StatusOK, responses.CancelPinMediaAdditionResponse{Status: resp.GetStatus()})
}

// RemoveMediaFromPin removes a single media from a pin (sessionless, ТЗ 4.2.3).
// @Summary Remove media from pin
// @Description Sessionless удаление одного медиа из пина с пересчётом агрегатов и S3 cleanup. Защита: пин не может остаться пустым (ТЗ 2.2.9).
// @Tags pins
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trip ID"
// @Param pin_id path string true "Pin ID"
// @Param media_id path string true "Media ID"
// @Success 200 {object} responses.PinResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 403 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 412 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/trips/{id}/pins/{pin_id}/media/{media_id} [delete]
func (h *TripHandler) RemoveMediaFromPin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if userID := middleware.UserIDFromContext(ctx); userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID := chi.URLParam(r, "id")
	pinID := chi.URLParam(r, "pin_id")
	mediaID := chi.URLParam(r, "media_id")
	if tripID == "" || pinID == "" || mediaID == "" {
		respondError(w, http.StatusBadRequest, "trip id, pin id, media id required")
		return
	}
	resp, err := h.tripClient.RemoveMediaFromPin(ctx, &proto.RemoveMediaFromPinRequest{
		TripId: tripID, PinId: pinID, MediaId: mediaID,
	})
	if err != nil {
		handleServiceError(w, r, err, "RemoveMediaFromPin")
		return
	}
	respondJSON(w, http.StatusOK, responses.PinResponse{Pin: tripPinProtoToResponse(resp.GetPin())})
}

// =============================================================================
// helpers
// =============================================================================

func filesToProto(in []requests.FileToUploadEntry) []*proto.FileToUpload {
	out := make([]*proto.FileToUpload, 0, len(in))
	for _, f := range in {
		out = append(out, &proto.FileToUpload{ClientId: f.ClientID, ContentType: f.ContentType})
	}
	return out
}

func uploadUrlsProtoToResponse(in []*proto.UploadUrl) []responses.UploadURL {
	out := make([]responses.UploadURL, 0, len(in))
	for _, u := range in {
		out = append(out, responses.UploadURL{ClientID: u.GetClientId(), S3Key: u.GetS3Key(), URL: u.GetUrl()})
	}
	return out
}

func pinAdditionDraftProtoToResponse(d *proto.PinAdditionDraft) responses.PinAdditionDraft {
	if d == nil {
		return responses.PinAdditionDraft{}
	}
	out := responses.PinAdditionDraft{
		PinIssues: d.GetPinIssues(),
		NSFWMediaIDs: d.GetNsfwMediaIds(),
		DedupedMediaIDs: d.GetDedupedMediaIds(),
	}
	for _, m := range d.GetNewMedia() {
		out.NewMedia = append(out.NewMedia, responses.ReviewPinMedia{
			MediaID: m.GetMediaId(),
			URL: m.GetUrl(),
			PrivacyLevel: m.GetPrivacyLevel(),
		})
	}
	return out
}

func similarGroupsProtoToResponse(in []*proto.MediaSimilarGroup) [][]string {
	out := make([][]string, 0, len(in))
	for _, g := range in {
		out = append(out, g.GetMediaIds())
	}
	return out
}
