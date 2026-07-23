package notification

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"pinz/backend/api-gateway-service/internal/middleware"
	pb "pinz/backend/api-gateway-service/pkg/proto"
)

const metadataUserIDKey = "x-user-id"

type Client struct {
	conn *grpc.ClientConn
	client pb.NotificationServiceClient
}

func NewClient() (*Client, error) {
	addr := os.Getenv("NOTIFICATION_SERVICE_GRPC_ADDRESS")
	if addr == "" {
		addr = "localhost:50054"
		slog.Warn("NOTIFICATION_SERVICE_GRPC_ADDRESS not set, using localhost:50054")
	}
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("notification gRPC client: %w", err)
	}
	return &Client{conn: conn, client: pb.NewNotificationServiceClient(conn)}, nil
}

func withUserIDMetadata(ctx context.Context) context.Context {
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, metadataUserIDKey, userID)
}

func (c *Client) RegisterDeviceToken(ctx context.Context, req *pb.RegisterDeviceTokenRequest) (*pb.RegisterDeviceTokenResponse, error) {
	return c.client.RegisterDeviceToken(withUserIDMetadata(ctx), req)
}

func (c *Client) UnregisterDeviceToken(ctx context.Context, req *pb.UnregisterDeviceTokenRequest) (*pb.UnregisterDeviceTokenResponse, error) {
	return c.client.UnregisterDeviceToken(withUserIDMetadata(ctx), req)
}

func (c *Client) Close() error {
	return c.conn.Close()
}
