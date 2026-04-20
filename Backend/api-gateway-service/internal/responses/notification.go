package responses

// DeviceTokenRegisterRequest — тело POST /api/v1/profile/device-tokens.
type DeviceTokenRegisterRequest struct {
	ApnsToken string `json:"apns_token"`
}

// DeviceTokenResponse — ответ регистрации.
type DeviceTokenResponse struct {
	TokenID string `json:"token_id"`
}

// DeviceTokenUnregisterRequest — тело DELETE /api/v1/profile/device-tokens.
type DeviceTokenUnregisterRequest struct {
	ApnsToken string `json:"apns_token"`
}

type DeviceTokenUnregisterResponse struct {
	Success bool `json:"success"`
}
