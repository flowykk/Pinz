package auth

import (
	"context"
	"fmt"
	"log"
	"os"

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
		log.Println("AUTH_SERVICE_GRPC_ADDRESS not set, using localhost:50051")
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

func (c *Client) SetPasswordAndUsername(ctx context.Context, req *pb.SetPasswordAndUsernameRequest) (*pb.SetPasswordAndUsernameResponse, error) {
	return c.client.SetPasswordAndUsername(ctx, req)
}

func (c *Client) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	return c.client.Login(ctx, req)
}

func (c *Client) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	return c.client.RefreshToken(ctx, req)
}

func (c *Client) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	return c.client.Logout(ctx, req)
}

func (c *Client) Close() error {
	return c.conn.Close()
}
