package services

import (
	"context"

	pb "pinz/backend/api-gateway-service/pkg/proto"
)

type AuthServiceInterface interface {
	SubmitEmail(ctx context.Context, email string) (*pb.SubmitEmailResponse, error)
	VerifyEmailCode(ctx context.Context, registrationID, verificationCode string) (*pb.VerifyEmailCodeResponse, error)
	PasskeyRegisterBegin(ctx context.Context, registrationID, username string) (*pb.PasskeyRegisterBeginResponse, error)
	PasskeyRegisterFinish(ctx context.Context, registrationID string, credentialJSON []byte) (*pb.PasskeyRegisterFinishResponse, error)
	PasskeyLoginBegin(ctx context.Context, email string) (*pb.PasskeyLoginBeginResponse, error)
	PasskeyLoginFinish(ctx context.Context, email string, credentialJSON []byte) (*pb.PasskeyLoginFinishResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*pb.RefreshTokenResponse, error)
	Logout(ctx context.Context, refreshToken string) (*pb.LogoutResponse, error)
	DevLogin(ctx context.Context, email string) (*pb.DevLoginResponse, error)
	GetProfile(ctx context.Context, userID string) (*pb.GetProfileResponse, error)
	UpdateProfile(ctx context.Context, userID, username string) (*pb.UpdateProfileResponse, error)
	ChangeEmail(ctx context.Context, userID, newEmail string) (*pb.ChangeEmailResponse, error)
	ConfirmEmailChange(ctx context.Context, userID, code string) (*pb.ConfirmEmailChangeResponse, error)
	RequestAvatarUpload(ctx context.Context, userID, filename, contentType string) (*pb.RequestAvatarUploadResponse, error)
	ConfirmAvatarUpload(ctx context.Context, userID, s3Key string) (*pb.ConfirmAvatarUploadResponse, error)
	DeleteAccount(ctx context.Context, userID string) (*pb.DeleteAccountResponse, error)
}

type AuthClient interface {
	SubmitEmail(ctx context.Context, req *pb.SubmitEmailRequest) (*pb.SubmitEmailResponse, error)
	VerifyEmailCode(ctx context.Context, req *pb.VerifyEmailCodeRequest) (*pb.VerifyEmailCodeResponse, error)
	PasskeyRegisterBegin(ctx context.Context, req *pb.PasskeyRegisterBeginRequest) (*pb.PasskeyRegisterBeginResponse, error)
	PasskeyRegisterFinish(ctx context.Context, req *pb.PasskeyRegisterFinishRequest) (*pb.PasskeyRegisterFinishResponse, error)
	PasskeyLoginBegin(ctx context.Context, req *pb.PasskeyLoginBeginRequest) (*pb.PasskeyLoginBeginResponse, error)
	PasskeyLoginFinish(ctx context.Context, req *pb.PasskeyLoginFinishRequest) (*pb.PasskeyLoginFinishResponse, error)
	RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error)
	Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error)
	DevLogin(ctx context.Context, req *pb.DevLoginRequest) (*pb.DevLoginResponse, error)
	GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error)
	UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error)
	ChangeEmail(ctx context.Context, req *pb.ChangeEmailRequest) (*pb.ChangeEmailResponse, error)
	ConfirmEmailChange(ctx context.Context, req *pb.ConfirmEmailChangeRequest) (*pb.ConfirmEmailChangeResponse, error)
	RequestAvatarUpload(ctx context.Context, req *pb.RequestAvatarUploadRequest) (*pb.RequestAvatarUploadResponse, error)
	ConfirmAvatarUpload(ctx context.Context, req *pb.ConfirmAvatarUploadRequest) (*pb.ConfirmAvatarUploadResponse, error)
	DeleteAccount(ctx context.Context, req *pb.DeleteAccountRequest) (*pb.DeleteAccountResponse, error)
}

type AuthService struct {
	authClient AuthClient
}

func NewAuthService(authClient AuthClient) *AuthService {
	return &AuthService{authClient: authClient}
}

func (s *AuthService) SubmitEmail(ctx context.Context, email string) (*pb.SubmitEmailResponse, error) {
	return s.authClient.SubmitEmail(ctx, &pb.SubmitEmailRequest{Email: email})
}

func (s *AuthService) VerifyEmailCode(ctx context.Context, registrationID, verificationCode string) (*pb.VerifyEmailCodeResponse, error) {
	return s.authClient.VerifyEmailCode(ctx, &pb.VerifyEmailCodeRequest{
		RegistrationId:   registrationID,
		VerificationCode: verificationCode,
	})
}

func (s *AuthService) PasskeyRegisterBegin(ctx context.Context, registrationID, username string) (*pb.PasskeyRegisterBeginResponse, error) {
	return s.authClient.PasskeyRegisterBegin(ctx, &pb.PasskeyRegisterBeginRequest{
		RegistrationId: registrationID,
		Username:       username,
	})
}

func (s *AuthService) PasskeyRegisterFinish(ctx context.Context, registrationID string, credentialJSON []byte) (*pb.PasskeyRegisterFinishResponse, error) {
	return s.authClient.PasskeyRegisterFinish(ctx, &pb.PasskeyRegisterFinishRequest{
		RegistrationId: registrationID,
		CredentialJson: credentialJSON,
	})
}

func (s *AuthService) PasskeyLoginBegin(ctx context.Context, email string) (*pb.PasskeyLoginBeginResponse, error) {
	return s.authClient.PasskeyLoginBegin(ctx, &pb.PasskeyLoginBeginRequest{Email: email})
}

func (s *AuthService) PasskeyLoginFinish(ctx context.Context, email string, credentialJSON []byte) (*pb.PasskeyLoginFinishResponse, error) {
	return s.authClient.PasskeyLoginFinish(ctx, &pb.PasskeyLoginFinishRequest{
		Email:          email,
		CredentialJson: credentialJSON,
	})
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*pb.RefreshTokenResponse, error) {
	return s.authClient.RefreshToken(ctx, &pb.RefreshTokenRequest{RefreshToken: refreshToken})
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) (*pb.LogoutResponse, error) {
	return s.authClient.Logout(ctx, &pb.LogoutRequest{RefreshToken: refreshToken})
}

func (s *AuthService) DevLogin(ctx context.Context, email string) (*pb.DevLoginResponse, error) {
	return s.authClient.DevLogin(ctx, &pb.DevLoginRequest{Email: email})
}

func (s *AuthService) GetProfile(ctx context.Context, userID string) (*pb.GetProfileResponse, error) {
	return s.authClient.GetProfile(ctx, &pb.GetProfileRequest{UserId: userID})
}

func (s *AuthService) UpdateProfile(ctx context.Context, userID, username string) (*pb.UpdateProfileResponse, error) {
	return s.authClient.UpdateProfile(ctx, &pb.UpdateProfileRequest{UserId: userID, Username: username})
}

func (s *AuthService) ChangeEmail(ctx context.Context, userID, newEmail string) (*pb.ChangeEmailResponse, error) {
	return s.authClient.ChangeEmail(ctx, &pb.ChangeEmailRequest{UserId: userID, NewEmail: newEmail})
}

func (s *AuthService) ConfirmEmailChange(ctx context.Context, userID, code string) (*pb.ConfirmEmailChangeResponse, error) {
	return s.authClient.ConfirmEmailChange(ctx, &pb.ConfirmEmailChangeRequest{UserId: userID, VerificationCode: code})
}

func (s *AuthService) RequestAvatarUpload(ctx context.Context, userID, filename, contentType string) (*pb.RequestAvatarUploadResponse, error) {
	return s.authClient.RequestAvatarUpload(ctx, &pb.RequestAvatarUploadRequest{
		UserId:      userID,
		Filename:    filename,
		ContentType: contentType,
	})
}

func (s *AuthService) ConfirmAvatarUpload(ctx context.Context, userID, s3Key string) (*pb.ConfirmAvatarUploadResponse, error) {
	return s.authClient.ConfirmAvatarUpload(ctx, &pb.ConfirmAvatarUploadRequest{UserId: userID, S3Key: s3Key})
}

func (s *AuthService) DeleteAccount(ctx context.Context, userID string) (*pb.DeleteAccountResponse, error) {
	return s.authClient.DeleteAccount(ctx, &pb.DeleteAccountRequest{UserId: userID})
}
