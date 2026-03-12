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
// @Description User sends email. Response indicates if user is registered (next step: passkey login) or not (next step: email verification). registration_id is set only when is_registered is false.
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
	respondJSON(w, http.StatusOK, responses.SubmitEmailResponse{
		IsRegistered:   resp.GetIsRegistered(),
		RegistrationID: resp.GetRegistrationKey(),
	})
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

// @Summary Begin passkey registration (step 1 of 2)
// @Description Returns PublicKeyCredentialCreationOptions (options_json) to pass to navigator.credentials.create(). Call after email verification.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body requests.PasskeyRegisterBeginRequest true "Registration ID and username"
// @Success 200 {object} responses.PasskeyRegisterBeginResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/auth/passkey/register/begin [post]
func (h *AuthHandler) PasskeyRegisterBegin(w http.ResponseWriter, r *http.Request) {
	var req requests.PasskeyRegisterBeginRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	resp, err := h.authSvc.PasskeyRegisterBegin(r.Context(), req.RegistrationID, req.Username)
	if err != nil {
		handleServiceError(w, err, "PasskeyRegisterBegin")
		return
	}
	respondJSON(w, http.StatusOK, responses.PasskeyRegisterBeginResponse{
		OptionsJSON: resp.GetOptionsJson(),
	})
}

// @Summary Finish passkey registration (step 2 of 2)
// @Description Verifies the attestation from the authenticator and creates the user account. Returns access and refresh tokens.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body requests.PasskeyRegisterFinishRequest true "Registration ID and attestation credential JSON"
// @Success 200 {object} responses.PasskeyRegisterFinishResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 409 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/auth/passkey/register/finish [post]
func (h *AuthHandler) PasskeyRegisterFinish(w http.ResponseWriter, r *http.Request) {
	var req requests.PasskeyRegisterFinishRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	resp, err := h.authSvc.PasskeyRegisterFinish(r.Context(), req.RegistrationID, req.CredentialJSON)
	if err != nil {
		handleServiceError(w, err, "PasskeyRegisterFinish")
		return
	}
	respondJSON(w, http.StatusOK, responses.PasskeyRegisterFinishResponse{
		AccessToken:  resp.GetAccessToken(),
		RefreshToken: resp.GetRefreshToken(),
	})
}

// @Summary Begin passkey login (step 1 of 2)
// @Description Returns PublicKeyCredentialRequestOptions (options_json) to pass to navigator.credentials.get(). Call after /auth/email confirms is_registered=true.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body requests.PasskeyLoginBeginRequest true "User email"
// @Success 200 {object} responses.PasskeyLoginBeginResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/auth/passkey/login/begin [post]
func (h *AuthHandler) PasskeyLoginBegin(w http.ResponseWriter, r *http.Request) {
	var req requests.PasskeyLoginBeginRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	resp, err := h.authSvc.PasskeyLoginBegin(r.Context(), req.Email)
	if err != nil {
		handleServiceError(w, err, "PasskeyLoginBegin")
		return
	}
	respondJSON(w, http.StatusOK, responses.PasskeyLoginBeginResponse{
		OptionsJSON: resp.GetOptionsJson(),
	})
}

// @Summary Finish passkey login (step 2 of 2)
// @Description Verifies the assertion from the authenticator. Returns access and refresh tokens.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body requests.PasskeyLoginFinishRequest true "User email and assertion credential JSON"
// @Success 200 {object} responses.PasskeyLoginFinishResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/auth/passkey/login/finish [post]
func (h *AuthHandler) PasskeyLoginFinish(w http.ResponseWriter, r *http.Request) {
	var req requests.PasskeyLoginFinishRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	resp, err := h.authSvc.PasskeyLoginFinish(r.Context(), req.Email, req.CredentialJSON)
	if err != nil {
		handleServiceError(w, err, "PasskeyLoginFinish")
		return
	}
	respondJSON(w, http.StatusOK, responses.PasskeyLoginFinishResponse{
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

// @Summary Dev login — bypass passkey, get tokens by email only
// @Description For development use only. Returns access and refresh tokens for any existing user without passkey verification.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body requests.DevLoginRequest true "User email"
// @Success 200 {object} responses.DevLoginResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /api/v1/auth/dev-login [post]
func (h *AuthHandler) DevLogin(w http.ResponseWriter, r *http.Request) {
	var req requests.DevLoginRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	resp, err := h.authSvc.DevLogin(r.Context(), req.Email)
	if err != nil {
		handleServiceError(w, err, "DevLogin")
		return
	}
	respondJSON(w, http.StatusOK, responses.DevLoginResponse{
		AccessToken:  resp.GetAccessToken(),
		RefreshToken: resp.GetRefreshToken(),
	})
}

// @Summary Logout
// @Tags auth
// @Accept json
// @Produce json
// @Param body body requests.LogoutRequest true "Refresh token"
// @Success 200 {object} responses.LogoutResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Security BearerAuth
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
