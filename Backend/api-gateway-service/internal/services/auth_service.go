package services

import (
	"context"

	pb "pinz/backend/api-gateway-service/pkg/proto"
)

type AuthServiceInterface interface {
	SubmitEmail(ctx context.Context, email string) (*pb.SubmitEmailResponse, error)
	VerifyEmailCode(ctx context.Context, registrationID, verificationCode string) (*pb.VerifyEmailCodeResponse, error)
	SetPasswordAndUsername(ctx context.Context, registrationID, password, username string) (*pb.SetPasswordAndUsernameResponse, error)
	Login(ctx context.Context, email, password string) (*pb.LoginResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*pb.RefreshTokenResponse, error)
	Logout(ctx context.Context, refreshToken string) (*pb.LogoutResponse, error)
}

type AuthClient interface {
	SubmitEmail(ctx context.Context, req *pb.SubmitEmailRequest) (*pb.SubmitEmailResponse, error)
	VerifyEmailCode(ctx context.Context, req *pb.VerifyEmailCodeRequest) (*pb.VerifyEmailCodeResponse, error)
	SetPasswordAndUsername(ctx context.Context, req *pb.SetPasswordAndUsernameRequest) (*pb.SetPasswordAndUsernameResponse, error)
	Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error)
	RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error)
	Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error)
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

func (s *AuthService) SetPasswordAndUsername(ctx context.Context, registrationID, password, username string) (*pb.SetPasswordAndUsernameResponse, error) {
	return s.authClient.SetPasswordAndUsername(ctx, &pb.SetPasswordAndUsernameRequest{
		RegistrationId: registrationID,
		Password:       password,
		Username:       username,
	})
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*pb.LoginResponse, error) {
	return s.authClient.Login(ctx, &pb.LoginRequest{Email: email, Password: password})
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*pb.RefreshTokenResponse, error) {
	return s.authClient.RefreshToken(ctx, &pb.RefreshTokenRequest{RefreshToken: refreshToken})
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) (*pb.LogoutResponse, error) {
	return s.authClient.Logout(ctx, &pb.LogoutRequest{RefreshToken: refreshToken})
}
