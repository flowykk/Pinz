package responses

type SubmitEmailResponse struct {
	IsRegistered    bool   `json:"is_registered"`
	RegistrationKey string `json:"registration_key,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type VerifyEmailCodeResponse struct {
	Success bool `json:"success"`
}

type SetPasswordAndUsernameResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
}

type LogoutResponse struct {
	Success bool `json:"success"`
}
