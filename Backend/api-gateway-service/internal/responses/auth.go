package responses

type SubmitEmailResponse struct {
	IsRegistered   bool   `json:"is_registered"`
	RegistrationID string `json:"registration_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type VerifyEmailCodeResponse struct {
	Success bool `json:"success"`
}

// PasskeyRegisterBeginResponse contains the PublicKeyCredentialCreationOptions to pass to the authenticator.
type PasskeyRegisterBeginResponse struct {
	OptionsJSON []byte `json:"options_json" swaggertype:"string" format:"base64"`
}

type PasskeyRegisterFinishResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// PasskeyLoginBeginResponse contains the PublicKeyCredentialRequestOptions to pass to the authenticator.
type PasskeyLoginBeginResponse struct {
	OptionsJSON []byte `json:"options_json" swaggertype:"string" format:"base64"`
}

type PasskeyLoginFinishResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
}

type LogoutResponse struct {
	Success bool `json:"success"`
}

type DevLoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type ProfileResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url,omitempty"`
	CreatedAt int64  `json:"created_at"`
}

type ChangeEmailResponse struct {
	Success bool `json:"success"`
}

type AvatarUploadResponse struct {
	UploadURL string `json:"upload_url"`
	S3Key     string `json:"s3_key"`
}

type DeleteAccountResponse struct {
	Success bool `json:"success"`
}
