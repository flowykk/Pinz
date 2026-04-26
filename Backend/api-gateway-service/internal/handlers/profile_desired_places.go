package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"pinz/backend/api-gateway-service/internal/middleware"
	"pinz/backend/api-gateway-service/internal/requests"
	"pinz/backend/api-gateway-service/internal/responses"
	pbproto "pinz/backend/api-gateway-service/pkg/proto"
)

// =============================================================================
// Желаемые места: CRUD под JWT (ТЗ 1.13).
// =============================================================================

// @Summary List desired places of current user
// @Description Возвращает список желаемых мест текущего пользователя в порядке от новых к старым. image_url — presigned GET URL, пустой если картинка не загружена.
// @Tags desired-places
// @Produce json
// @Success 200 {object} responses.DesiredPlacesListResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/profile/desired-places [get]
func (h *AuthHandler) ListDesiredPlaces(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	resp, err := h.authSvc.ListDesiredPlaces(r.Context(), userID)
	if err != nil {
		handleServiceError(w, r, err, "ListDesiredPlaces")
		return
	}
	respondJSON(w, http.StatusOK, responses.DesiredPlacesListResponse{
		Places: desiredPlacesProtoToResponse(resp.GetPlaces()),
	})
}

// @Summary Create desired place
// @Description Создаёт новое желаемое место. s3_key опционален — передаётся, если клиент уже загрузил картинку через /profile/desired-places/upload-url.
// @Tags desired-places
// @Accept json
// @Produce json
// @Param body body requests.CreateDesiredPlaceRequest true "Place payload"
// @Success 200 {object} responses.DesiredPlaceResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/profile/desired-places [post]
func (h *AuthHandler) CreateDesiredPlace(w http.ResponseWriter, r *http.Request) {
	var req requests.CreateDesiredPlaceRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	resp, err := h.authSvc.CreateDesiredPlace(r.Context(), userID, req.Name, req.Description, req.S3Key)
	if err != nil {
		handleServiceError(w, r, err, "CreateDesiredPlace")
		return
	}
	respondJSON(w, http.StatusOK, desiredPlaceProtoToResponse(resp.GetPlace()))
}

// @Summary Update desired place
// @Description Изменение name/description, опционально картинки. image_s3_key (опц): не передан — картинка не меняется; пустая строка — картинка сбрасывается; иначе — заменяется.
// @Tags desired-places
// @Accept json
// @Produce json
// @Param place_id path string true "Place ID"
// @Param body body requests.UpdateDesiredPlaceRequest true "Update payload"
// @Success 200 {object} responses.DesiredPlaceResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/profile/desired-places/{place_id} [patch]
func (h *AuthHandler) UpdateDesiredPlace(w http.ResponseWriter, r *http.Request) {
	placeID := chi.URLParam(r, "place_id")
	if placeID == "" {
		respondError(w, http.StatusBadRequest, "place_id is required")
		return
	}
	var req requests.UpdateDesiredPlaceRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	setImageKey := req.ImageUpdate != nil
	s3Key := ""
	if setImageKey {
		s3Key = *req.ImageUpdate
	}
	resp, err := h.authSvc.UpdateDesiredPlace(r.Context(), userID, placeID, req.Name, req.Description, setImageKey, s3Key)
	if err != nil {
		handleServiceError(w, r, err, "UpdateDesiredPlace")
		return
	}
	respondJSON(w, http.StatusOK, desiredPlaceProtoToResponse(resp.GetPlace()))
}

// @Summary Delete desired place
// @Description Удаляет желаемое место и связанный S3-объект (если есть).
// @Tags desired-places
// @Produce json
// @Param place_id path string true "Place ID"
// @Success 200 {object} responses.DeleteDesiredPlaceResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/profile/desired-places/{place_id} [delete]
func (h *AuthHandler) DeleteDesiredPlace(w http.ResponseWriter, r *http.Request) {
	placeID := chi.URLParam(r, "place_id")
	if placeID == "" {
		respondError(w, http.StatusBadRequest, "place_id is required")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	resp, err := h.authSvc.DeleteDesiredPlace(r.Context(), userID, placeID)
	if err != nil {
		handleServiceError(w, r, err, "DeleteDesiredPlace")
		return
	}
	respondJSON(w, http.StatusOK, responses.DeleteDesiredPlaceResponse{Success: resp.GetSuccess()})
}

// @Summary Request presigned URL for desired place image upload
// @Description Возвращает presigned PUT URL и s3_key. После загрузки клиент передаёт s3_key в Create или Update.
// @Tags desired-places
// @Accept json
// @Produce json
// @Param body body requests.RequestDesiredPlaceImageUploadRequest true "Filename and content type"
// @Success 200 {object} responses.DesiredPlaceImageUploadResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 503 {object} responses.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/profile/desired-places/upload-url [post]
func (h *AuthHandler) RequestDesiredPlaceImageUpload(w http.ResponseWriter, r *http.Request) {
	var req requests.RequestDesiredPlaceImageUploadRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	resp, err := h.authSvc.RequestDesiredPlaceImageUpload(r.Context(), userID, req.Filename, req.ContentType)
	if err != nil {
		handleServiceError(w, r, err, "RequestDesiredPlaceImageUpload")
		return
	}
	respondJSON(w, http.StatusOK, responses.DesiredPlaceImageUploadResponse{
		UploadURL: resp.GetUploadUrl(),
		S3Key:     resp.GetS3Key(),
	})
}

// @Summary Delete desired place image
// @Description Сбрасывает картинку у желаемого места и удаляет S3-объект.
// @Tags desired-places
// @Produce json
// @Param place_id path string true "Place ID"
// @Success 200 {object} responses.DesiredPlaceResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/profile/desired-places/{place_id}/image [delete]
func (h *AuthHandler) DeleteDesiredPlaceImage(w http.ResponseWriter, r *http.Request) {
	placeID := chi.URLParam(r, "place_id")
	if placeID == "" {
		respondError(w, http.StatusBadRequest, "place_id is required")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	resp, err := h.authSvc.DeleteDesiredPlaceImage(r.Context(), userID, placeID)
	if err != nil {
		handleServiceError(w, r, err, "DeleteDesiredPlaceImage")
		return
	}
	respondJSON(w, http.StatusOK, desiredPlaceProtoToResponse(resp.GetPlace()))
}

// =============================================================================
// helpers
// =============================================================================

func desiredPlaceProtoToResponse(p *pbproto.DesiredPlace) responses.DesiredPlaceResponse {
	if p == nil {
		return responses.DesiredPlaceResponse{}
	}
	return responses.DesiredPlaceResponse{
		ID:          p.GetId(),
		Name:        p.GetName(),
		Description: p.GetDescription(),
		ImageURL:    p.GetImageUrl(),
		CreatedAt:   p.GetCreatedAtUnix(),
	}
}

func desiredPlacesProtoToResponse(in []*pbproto.DesiredPlace) []responses.DesiredPlaceResponse {
	out := make([]responses.DesiredPlaceResponse, 0, len(in))
	for _, p := range in {
		out = append(out, desiredPlaceProtoToResponse(p))
	}
	return out
}
