package handlers

import (
	"fmt"
	"net/http"

	"pinz/backend/api-gateway-service/internal/middleware"
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
		handleServiceError(w, r, err, "SubmitEmail")
		return
	}
	respondJSON(w, http.StatusOK, responses.SubmitEmailResponse{
		IsRegistered: resp.GetIsRegistered(),
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
		handleServiceError(w, r, err, "VerifyEmailCode")
		return
	}
	respondJSON(w, http.StatusOK, responses.VerifyEmailCodeResponse{Success: resp.GetSuccess()})
}

// @Summary Begin passkey registration (step 1 of 2)
// @Description Returns PublicKeyCredentialCreationOptions (options_json) to pass to navigator.credentials.create. Call after email verification.
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
		handleServiceError(w, r, err, "PasskeyRegisterBegin")
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
		handleServiceError(w, r, err, "PasskeyRegisterFinish")
		return
	}
	respondJSON(w, http.StatusOK, responses.PasskeyRegisterFinishResponse{
		AccessToken: resp.GetAccessToken(),
		RefreshToken: resp.GetRefreshToken(),
	})
}

// @Summary Begin passkey login (step 1 of 2)
// @Description Returns PublicKeyCredentialRequestOptions (options_json) to pass to navigator.credentials.get. Call after /auth/email confirms is_registered=true.
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
		handleServiceError(w, r, err, "PasskeyLoginBegin")
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
		handleServiceError(w, r, err, "PasskeyLoginFinish")
		return
	}
	respondJSON(w, http.StatusOK, responses.PasskeyLoginFinishResponse{
		AccessToken: resp.GetAccessToken(),
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
		handleServiceError(w, r, err, "RefreshToken")
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
		handleServiceError(w, r, err, "DevLogin")
		return
	}
	respondJSON(w, http.StatusOK, responses.DevLoginResponse{
		AccessToken: resp.GetAccessToken(),
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
		handleServiceError(w, r, err, "Logout")
		return
	}
	respondJSON(w, http.StatusOK, responses.LogoutResponse{Success: resp.GetSuccess()})
}

// @Summary Get current user profile
// @Tags profile
// @Produce json
// @Success 200 {object} responses.ProfileResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/profile [get]
func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	resp, err := h.authSvc.GetProfile(r.Context(), userID)
	if err != nil {
		handleServiceError(w, r, err, "GetProfile")
		return
	}
	u := resp.GetUser()
	respondJSON(w, http.StatusOK, responses.ProfileResponse{
		ID: u.GetId(),
		Username: u.GetUsername(),
		Email: u.GetEmail(),
		AvatarURL: u.GetAvatarUrl(),
		CreatedAt: u.GetCreatedAtUnix(),
	})
}

// @Summary Update profile (change username)
// @Tags profile
// @Accept json
// @Produce json
// @Param body body requests.UpdateProfileRequest true "New username"
// @Success 200 {object} responses.ProfileResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 409 {object} responses.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/profile [patch]
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req requests.UpdateProfileRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	resp, err := h.authSvc.UpdateProfile(r.Context(), userID, req.Username)
	if err != nil {
		handleServiceError(w, r, err, "UpdateProfile")
		return
	}
	u := resp.GetUser()
	respondJSON(w, http.StatusOK, responses.ProfileResponse{
		ID: u.GetId(),
		Username: u.GetUsername(),
		Email: u.GetEmail(),
		AvatarURL: u.GetAvatarUrl(),
		CreatedAt: u.GetCreatedAtUnix(),
	})
}

// @Summary Initiate email change (sends verification code to new email)
// @Tags profile
// @Accept json
// @Produce json
// @Param body body requests.ChangeEmailRequest true "New email"
// @Success 200 {object} responses.ChangeEmailResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 409 {object} responses.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/profile/change-email [post]
func (h *AuthHandler) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	var req requests.ChangeEmailRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	resp, err := h.authSvc.ChangeEmail(r.Context(), userID, req.NewEmail)
	if err != nil {
		handleServiceError(w, r, err, "ChangeEmail")
		return
	}
	respondJSON(w, http.StatusOK, responses.ChangeEmailResponse{Success: resp.GetSuccess()})
}

// @Summary Confirm email change with verification code
// @Tags profile
// @Accept json
// @Produce json
// @Param body body requests.ConfirmEmailChangeRequest true "Verification code"
// @Success 200 {object} responses.ProfileResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/profile/confirm-email [post]
func (h *AuthHandler) ConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
	var req requests.ConfirmEmailChangeRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	resp, err := h.authSvc.ConfirmEmailChange(r.Context(), userID, req.VerificationCode)
	if err != nil {
		handleServiceError(w, r, err, "ConfirmEmailChange")
		return
	}
	u := resp.GetUser()
	respondJSON(w, http.StatusOK, responses.ProfileResponse{
		ID: u.GetId(),
		Username: u.GetUsername(),
		Email: u.GetEmail(),
		AvatarURL: u.GetAvatarUrl(),
		CreatedAt: u.GetCreatedAtUnix(),
	})
}

// @Summary Request presigned URL for avatar upload
// @Tags profile
// @Accept json
// @Produce json
// @Param body body requests.RequestAvatarUploadRequest true "Filename and content type"
// @Success 200 {object} responses.AvatarUploadResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/profile/avatar/upload [post]
func (h *AuthHandler) RequestAvatarUpload(w http.ResponseWriter, r *http.Request) {
	var req requests.RequestAvatarUploadRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	resp, err := h.authSvc.RequestAvatarUpload(r.Context(), userID, req.Filename, req.ContentType)
	if err != nil {
		handleServiceError(w, r, err, "RequestAvatarUpload")
		return
	}
	respondJSON(w, http.StatusOK, responses.AvatarUploadResponse{
		UploadURL: resp.GetUploadUrl(),
		S3Key: resp.GetS3Key(),
	})
}

// @Summary Confirm avatar upload after uploading to S3
// @Tags profile
// @Accept json
// @Produce json
// @Param body body requests.ConfirmAvatarUploadRequest true "S3 key"
// @Success 200 {object} responses.ProfileResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/profile/avatar/confirm [post]
func (h *AuthHandler) ConfirmAvatarUpload(w http.ResponseWriter, r *http.Request) {
	var req requests.ConfirmAvatarUploadRequest
	if err := decodeJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	resp, err := h.authSvc.ConfirmAvatarUpload(r.Context(), userID, req.S3Key)
	if err != nil {
		handleServiceError(w, r, err, "ConfirmAvatarUpload")
		return
	}
	u := resp.GetUser()
	respondJSON(w, http.StatusOK, responses.ProfileResponse{
		ID: u.GetId(),
		Username: u.GetUsername(),
		Email: u.GetEmail(),
		AvatarURL: u.GetAvatarUrl(),
		CreatedAt: u.GetCreatedAtUnix(),
	})
}

// @Summary Delete avatar
// @Description Removes the user's avatar. Deletes the file from S3 and clears avatar_url.
// @Tags profile
// @Produce json
// @Success 200 {object} responses.ProfileResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/profile/avatar [delete]
func (h *AuthHandler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	resp, err := h.authSvc.DeleteAvatar(r.Context(), userID)
	if err != nil {
		handleServiceError(w, r, err, "DeleteAvatar")
		return
	}
	u := resp.GetUser()
	respondJSON(w, http.StatusOK, responses.ProfileResponse{
		ID: u.GetId(),
		Username: u.GetUsername(),
		Email: u.GetEmail(),
		AvatarURL: u.GetAvatarUrl(),
		CreatedAt: u.GetCreatedAtUnix(),
	})
}

// @Summary Delete account
// @Description Permanently deletes the user account. Pins and media in trips are preserved.
// @Tags profile
// @Produce json
// @Success 200 {object} responses.DeleteAccountResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/profile [delete]
func (h *AuthHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	resp, err := h.authSvc.DeleteAccount(r.Context(), userID)
	if err != nil {
		handleServiceError(w, r, err, "DeleteAccount")
		return
	}
	respondJSON(w, http.StatusOK, responses.DeleteAccountResponse{Success: resp.GetSuccess()})
}
