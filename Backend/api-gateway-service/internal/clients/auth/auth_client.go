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
	conn   *grpc.ClientConn
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
		conn:   conn,
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

func (c *Client) Close() error {
	return c.conn.Close()
}
