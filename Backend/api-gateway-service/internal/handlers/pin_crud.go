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
// @Description Получить пин со всеми полями. Доступ — participant трипа или favourite-юзер. Если пин скрыт для caller'а через pin_hidden_by_user (soft-delete-for-self) — 404.
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

// UpdatePin updates pin metadata.
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

// DeletePin deletes a pin.
// @Summary Delete pin
// @Description Удаление пина. Если трип в избранном у других пользователей — soft-delete-for-self через pin_hidden_by_user; иначе full delete с каскадом media и S3 cleanup. Любой participant. Защита: запрет при активной pin_media_addition_session.
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

// RemoveMediaFromPin removes a single media from a pin (sessionless).
// @Summary Remove media from pin
// @Description Sessionless удаление одного медиа из пина с пересчётом агрегатов и S3 cleanup. Защита: пин не может остаться пустым.
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

func similarGroupsProtoToResponse(in []*proto.MediaSimilarGroup) [][]string {
	out := make([][]string, 0, len(in))
	for _, g := range in {
		out = append(out, g.GetMediaIds())
	}
	return out
}
