package requests

type SubmitEmailRequest struct {
	Email string `json:"email" example:"user@example.com"`
}

type VerifyEmailCodeRequest struct {
	RegistrationID   string `json:"registration_id"`
	VerificationCode string `json:"verification_code"`
}

type PasskeyRegisterBeginRequest struct {
	RegistrationID string `json:"registration_id"`
	Username       string `json:"username"`
}

// PasskeyRegisterFinishRequest carries the raw attestation JSON from the authenticator.
type PasskeyRegisterFinishRequest struct {
	RegistrationID string `json:"registration_id"`
	// CredentialJSON is the PublicKeyCredential (attestation) object serialised to JSON by the client.
	CredentialJSON []byte `json:"credential_json" swaggertype:"string" format:"base64"`
}

type PasskeyLoginBeginRequest struct {
	Email string `json:"email" example:"user@example.com"`
}

// PasskeyLoginFinishRequest carries the raw assertion JSON from the authenticator.
type PasskeyLoginFinishRequest struct {
	Email string `json:"email" example:"user@example.com"`
	// CredentialJSON is the PublicKeyCredential (assertion) object serialised to JSON by the client.
	CredentialJSON []byte `json:"credential_json" swaggertype:"string" format:"base64"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type DevLoginRequest struct {
	Email string `json:"email" example:"user@example.com"`
}

type UpdateProfileRequest struct {
	Username string `json:"username" example:"johndoe"`
}

type ChangeEmailRequest struct {
	NewEmail string `json:"new_email" example:"new@example.com"`
}

type ConfirmEmailChangeRequest struct {
	VerificationCode string `json:"verification_code"`
}

type RequestAvatarUploadRequest struct {
	Filename    string `json:"filename" example:"avatar.jpg"`
	ContentType string `json:"content_type" example:"image/jpeg"`
}

type ConfirmAvatarUploadRequest struct {
	S3Key string `json:"s3_key"`
}
