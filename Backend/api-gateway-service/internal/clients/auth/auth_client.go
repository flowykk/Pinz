package auth

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "pinz/backend/api-gateway-service/pkg/proto"
)

type Client struct {
	conn *grpc.ClientConn
	client pb.AuthServiceClient
}

func NewClient() (*Client, error) {
	addr := os.Getenv("AUTH_SERVICE_GRPC_ADDRESS")
	if addr == "" {
		addr = "localhost:50051"
		slog.Warn("AUTH_SERVICE_GRPC_ADDRESS not set, using localhost:50051")
	}
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("auth gRPC client: %w", err)
	}
	return &Client{
		conn: conn,
		client: pb.NewAuthServiceClient(conn),
	}, nil
}

func (c *Client) SubmitEmail(ctx context.Context, req *pb.SubmitEmailRequest) (*pb.SubmitEmailResponse, error) {
	return c.client.SubmitEmail(ctx, req)
}

func (c *Client) VerifyEmailCode(ctx context.Context, req *pb.VerifyEmailCodeRequest) (*pb.VerifyEmailCodeResponse, error) {
	return c.client.VerifyEmailCode(ctx, req)
}

func (c *Client) PasskeyRegisterBegin(ctx context.Context, req *pb.PasskeyRegisterBeginRequest) (*pb.PasskeyRegisterBeginResponse, error) {
	return c.client.PasskeyRegisterBegin(ctx, req)
}

func (c *Client) PasskeyRegisterFinish(ctx context.Context, req *pb.PasskeyRegisterFinishRequest) (*pb.PasskeyRegisterFinishResponse, error) {
	return c.client.PasskeyRegisterFinish(ctx, req)
}

func (c *Client) PasskeyLoginBegin(ctx context.Context, req *pb.PasskeyLoginBeginRequest) (*pb.PasskeyLoginBeginResponse, error) {
	return c.client.PasskeyLoginBegin(ctx, req)
}

func (c *Client) PasskeyLoginFinish(ctx context.Context, req *pb.PasskeyLoginFinishRequest) (*pb.PasskeyLoginFinishResponse, error) {
	return c.client.PasskeyLoginFinish(ctx, req)
}

func (c *Client) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	return c.client.RefreshToken(ctx, req)
}

func (c *Client) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	return c.client.Logout(ctx, req)
}

func (c *Client) DevLogin(ctx context.Context, req *pb.DevLoginRequest) (*pb.DevLoginResponse, error) {
	return c.client.DevLogin(ctx, req)
}

func (c *Client) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	return c.client.GetProfile(ctx, req)
}

func (c *Client) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error) {
	return c.client.UpdateProfile(ctx, req)
}

func (c *Client) ChangeEmail(ctx context.Context, req *pb.ChangeEmailRequest) (*pb.ChangeEmailResponse, error) {
	return c.client.ChangeEmail(ctx, req)
}

func (c *Client) ConfirmEmailChange(ctx context.Context, req *pb.ConfirmEmailChangeRequest) (*pb.ConfirmEmailChangeResponse, error) {
	return c.client.ConfirmEmailChange(ctx, req)
}

func (c *Client) RequestAvatarUpload(ctx context.Context, req *pb.RequestAvatarUploadRequest) (*pb.RequestAvatarUploadResponse, error) {
	return c.client.RequestAvatarUpload(ctx, req)
}

func (c *Client) ConfirmAvatarUpload(ctx context.Context, req *pb.ConfirmAvatarUploadRequest) (*pb.ConfirmAvatarUploadResponse, error) {
	return c.client.ConfirmAvatarUpload(ctx, req)
}

func (c *Client) DeleteAvatar(ctx context.Context, req *pb.DeleteAvatarRequest) (*pb.DeleteAvatarResponse, error) {
	return c.client.DeleteAvatar(ctx, req)
}

func (c *Client) DeleteAccount(ctx context.Context, req *pb.DeleteAccountRequest) (*pb.DeleteAccountResponse, error) {
	return c.client.DeleteAccount(ctx, req)
}

func (c *Client) GetUsersProfiles(ctx context.Context, req *pb.GetUsersProfilesRequest) (*pb.GetUsersProfilesResponse, error) {
	return c.client.GetUsersProfiles(ctx, req)
}

func (c *Client) ListDesiredPlaces(ctx context.Context, req *pb.ListDesiredPlacesRequest) (*pb.ListDesiredPlacesResponse, error) {
	return c.client.ListDesiredPlaces(ctx, req)
}

func (c *Client) CreateDesiredPlace(ctx context.Context, req *pb.CreateDesiredPlaceRequest) (*pb.CreateDesiredPlaceResponse, error) {
	return c.client.CreateDesiredPlace(ctx, req)
}

func (c *Client) UpdateDesiredPlace(ctx context.Context, req *pb.UpdateDesiredPlaceRequest) (*pb.UpdateDesiredPlaceResponse, error) {
	return c.client.UpdateDesiredPlace(ctx, req)
}

func (c *Client) DeleteDesiredPlace(ctx context.Context, req *pb.DeleteDesiredPlaceRequest) (*pb.DeleteDesiredPlaceResponse, error) {
	return c.client.DeleteDesiredPlace(ctx, req)
}

func (c *Client) RequestDesiredPlaceImageUpload(ctx context.Context, req *pb.RequestDesiredPlaceImageUploadRequest) (*pb.RequestDesiredPlaceImageUploadResponse, error) {
	return c.client.RequestDesiredPlaceImageUpload(ctx, req)
}

func (c *Client) DeleteDesiredPlaceImage(ctx context.Context, req *pb.DeleteDesiredPlaceImageRequest) (*pb.DeleteDesiredPlaceImageResponse, error) {
	return c.client.DeleteDesiredPlaceImage(ctx, req)
}

func (c *Client) GetPublicUserProfile(ctx context.Context, req *pb.GetPublicUserProfileRequest) (*pb.GetPublicUserProfileResponse, error) {
	return c.client.GetPublicUserProfile(ctx, req)
}

func (c *Client) Close() error {
	return c.conn.Close()
}
