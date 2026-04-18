package statistics

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
	conn   *grpc.ClientConn
	client pb.StatisticsServiceClient
}

func NewClient() (*Client, error) {
	addr := os.Getenv("STATISTICS_SERVICE_GRPC_ADDRESS")
	if addr == "" {
		addr = "localhost:50053"
		slog.Warn("STATISTICS_SERVICE_GRPC_ADDRESS not set, using localhost:50053")
	}
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("statistics gRPC client: %w", err)
	}
	return &Client{
		conn:   conn,
		client: pb.NewStatisticsServiceClient(conn),
	}, nil
}

func withUserIDMetadata(ctx context.Context) context.Context {
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, metadataUserIDKey, userID)
}

func (c *Client) GetUserStats(ctx context.Context, req *pb.GetUserStatsRequest) (*pb.GetUserStatsResponse, error) {
	return c.client.GetUserStats(withUserIDMetadata(ctx), req)
}

func (c *Client) GetVisitedLocations(ctx context.Context, req *pb.GetVisitedLocationsRequest) (*pb.GetVisitedLocationsResponse, error) {
	return c.client.GetVisitedLocations(withUserIDMetadata(ctx), req)
}

func (c *Client) Close() error {
	return c.conn.Close()
}
