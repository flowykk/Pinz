package handlers

import (
	"context"
	"net/http"
	"strings"

	"pinz/backend/api-gateway-service/internal/middleware"
	"pinz/backend/api-gateway-service/internal/responses"
	pb "pinz/backend/api-gateway-service/pkg/proto"
)

// NotificationClient — RPC клиент notification-service.
type NotificationClient interface {
	RegisterDeviceToken(ctx context.Context, req *pb.RegisterDeviceTokenRequest) (*pb.RegisterDeviceTokenResponse, error)
	UnregisterDeviceToken(ctx context.Context, req *pb.UnregisterDeviceTokenRequest) (*pb.UnregisterDeviceTokenResponse, error)
}

type NotificationHandler struct {
	client NotificationClient
}

func NewNotificationHandler(client NotificationClient) *NotificationHandler {
	return &NotificationHandler{client: client}
}

// RegisterDeviceToken — регистрация APNS-токена для текущего пользователя.
//
// @Summary Register APNS device token
// @Description Registers an APNS device token for the authenticated user. Idempotent: повторная регистрация того же токена допустима (обновится owner).
// @Tags notifications
// @Accept json
// @Produce json
// @Param body body responses.DeviceTokenRegisterRequest true "APNS token"
// @Success 200 {object} responses.DeviceTokenResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/profile/device-tokens [post]
func (h *NotificationHandler) RegisterDeviceToken(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "user_id is required")
		return
	}
	var req responses.DeviceTokenRegisterRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	token := strings.TrimSpace(req.ApnsToken)
	if token == "" {
		respondError(w, http.StatusBadRequest, "apns_token is required")
		return
	}
	resp, err := h.client.RegisterDeviceToken(r.Context(), &pb.RegisterDeviceTokenRequest{UserId: userID, ApnsToken: token})
	if err != nil {
		handleServiceError(w, r, err, "RegisterDeviceToken")
		return
	}
	respondJSON(w, http.StatusOK, responses.DeviceTokenResponse{TokenID: resp.GetTokenId()})
}

// UnregisterDeviceToken — удаление APNS-токена (например, при logout на устройстве).
//
// @Summary Unregister APNS device token
// @Description Removes an APNS device token for the authenticated user.
// @Tags notifications
// @Accept json
// @Produce json
// @Param body body responses.DeviceTokenUnregisterRequest true "APNS token"
// @Success 200 {object} responses.DeviceTokenUnregisterResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/profile/device-tokens [delete]
func (h *NotificationHandler) UnregisterDeviceToken(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "user_id is required")
		return
	}
	var req responses.DeviceTokenUnregisterRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	token := strings.TrimSpace(req.ApnsToken)
	if token == "" {
		respondError(w, http.StatusBadRequest, "apns_token is required")
		return
	}
	resp, err := h.client.UnregisterDeviceToken(r.Context(), &pb.UnregisterDeviceTokenRequest{UserId: userID, ApnsToken: token})
	if err != nil {
		handleServiceError(w, r, err, "UnregisterDeviceToken")
		return
	}
	respondJSON(w, http.StatusOK, responses.DeviceTokenUnregisterResponse{Success: resp.GetSuccess()})
}
