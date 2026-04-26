package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"pinz/backend/api-gateway-service/internal/middleware"
	"pinz/backend/api-gateway-service/internal/responses"
)

// @Summary Get public profile of another user
// @Description Возвращает публичные поля профиля (username, avatar, created_at) и список желаемых мест пользователя. Email и другие приватные поля не отдаются. Используется при переходе на чужой профиль из участников трипа или карточки фида.
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} responses.PublicUserProfileResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/users/{id} [get]
func (h *AuthHandler) GetPublicUserProfile(w http.ResponseWriter, r *http.Request) {
	if middleware.UserIDFromContext(r.Context()) == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID := chi.URLParam(r, "id")
	if userID == "" {
		respondError(w, http.StatusBadRequest, "user id is required")
		return
	}
	resp, err := h.authSvc.GetPublicUserProfile(r.Context(), userID)
	if err != nil {
		handleServiceError(w, r, err, "GetPublicUserProfile")
		return
	}
	profile := resp.GetProfile()
	respondJSON(w, http.StatusOK, responses.PublicUserProfileResponse{
		ID:            profile.GetUserId(),
		Username:      profile.GetUsername(),
		AvatarURL:     profile.GetAvatarUrl(),
		CreatedAt:     profile.GetCreatedAtUnix(),
		DesiredPlaces: desiredPlacesProtoToResponse(resp.GetDesiredPlaces()),
	})
}
