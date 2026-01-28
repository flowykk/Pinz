package requests

type SubmitEmailRequest struct {
	Email string `json:"email" example:"user@example.com"`
}

type VerifyEmailCodeRequest struct {
	RegistrationID   string `json:"registration_id"`
	VerificationCode string `json:"verification_code"`
}

type SetPasswordAndUsernameRequest struct {
	RegistrationID string `json:"registration_id"`
	Password       string `json:"password"`
	Username       string `json:"username"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}
