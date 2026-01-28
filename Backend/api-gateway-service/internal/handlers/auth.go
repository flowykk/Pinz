package handlers

import (
	"fmt"
	"net/http"

	"pinz/backend/api-gateway-service/internal/requests"
	"pinz/backend/api-gateway-service/internal/responses"
	"pinz/backend/api-gateway-service/internal/services"
)

type AuthHandler struct {
	authSvc services.AuthServiceInterface
}

func NewAuthHandler(authSvc services.AuthServiceInterface) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// @Summary Submit email (login or registration flow)
// @Description User sends email. Response indicates if user is registered (next step: password) or not (next step: code from email). registration_key is set only when is_registered is false.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body requests.SubmitEmailRequest true "Email"
// @Success 200 {object} responses.SubmitEmailResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/auth/email [post]
func (h *AuthHandler) SubmitEmail(w http.ResponseWriter, r *http.Request) {
	var req requests.SubmitEmailRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	resp, err := h.authSvc.SubmitEmail(r.Context(), req.Email)
	if err != nil {
		handleServiceError(w, err, "SubmitEmail")
		return
	}
	out := responses.SubmitEmailResponse{
		IsRegistered:    resp.GetIsRegistered(),
		RegistrationKey: resp.GetRegistrationKey(),
	}
	respondJSON(w, http.StatusOK, out)
}

// @Summary Verify email code
// @Tags auth
// @Accept json
// @Produce json
// @Param body body requests.VerifyEmailCodeRequest true "Registration ID and code"
// @Success 200 {object} responses.VerifyEmailCodeResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/auth/verify-email [post]
func (h *AuthHandler) VerifyEmailCode(w http.ResponseWriter, r *http.Request) {
	var req requests.VerifyEmailCodeRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	resp, err := h.authSvc.VerifyEmailCode(r.Context(), req.RegistrationID, req.VerificationCode)
	if err != nil {
		handleServiceError(w, err, "VerifyEmailCode")
		return
	}
	respondJSON(w, http.StatusOK, responses.VerifyEmailCodeResponse{Success: resp.GetSuccess()})
}

// @Summary Set password and username (finish registration)
// @Tags auth
// @Accept json
// @Produce json
// @Param body body requests.SetPasswordAndUsernameRequest true "Registration ID, password, username"
// @Success 200 {object} responses.SetPasswordAndUsernameResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 409 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/auth/finish-register [post]
func (h *AuthHandler) SetPasswordAndUsername(w http.ResponseWriter, r *http.Request) {
	var req requests.SetPasswordAndUsernameRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	resp, err := h.authSvc.SetPasswordAndUsername(r.Context(), req.RegistrationID, req.Password, req.Username)
	if err != nil {
		handleServiceError(w, err, "SetPasswordAndUsername")
		return
	}
	respondJSON(w, http.StatusOK, responses.SetPasswordAndUsernameResponse{
		AccessToken:  resp.GetAccessToken(),
		RefreshToken: resp.GetRefreshToken(),
	})
}

// @Summary Login
// @Tags auth
// @Accept json
// @Produce json
// @Param body body requests.LoginRequest true "Email and password"
// @Success 200 {object} responses.LoginResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req requests.LoginRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	resp, err := h.authSvc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		handleServiceError(w, err, "Login")
		return
	}
	respondJSON(w, http.StatusOK, responses.LoginResponse{
		AccessToken:  resp.GetAccessToken(),
		RefreshToken: resp.GetRefreshToken(),
	})
}

// @Summary Refresh access token
// @Tags auth
// @Accept json
// @Produce json
// @Param body body requests.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} responses.RefreshTokenResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req requests.RefreshTokenRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	resp, err := h.authSvc.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		handleServiceError(w, err, "RefreshToken")
		return
	}
	respondJSON(w, http.StatusOK, responses.RefreshTokenResponse{AccessToken: resp.GetAccessToken()})
}

// @Summary Logout
// @Tags auth
// @Accept json
// @Produce json
// @Param body body requests.LogoutRequest true "Refresh token"
// @Success 200 {object} responses.LogoutResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req requests.LogoutRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	resp, err := h.authSvc.Logout(r.Context(), req.RefreshToken)
	if err != nil {
		handleServiceError(w, err, "Logout")
		return
	}
	respondJSON(w, http.StatusOK, responses.LogoutResponse{Success: resp.GetSuccess()})
}
