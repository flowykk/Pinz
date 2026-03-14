package trip

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
	client pb.TripServiceClient
}

func NewClient() (*Client, error) {
	addr := os.Getenv("TRIP_SERVICE_GRPC_ADDRESS")
	if addr == "" {
		addr = "localhost:50052"
		slog.Warn("TRIP_SERVICE_GRPC_ADDRESS not set, using localhost:50052")
	}
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("trip gRPC client: %w", err)
	}
	return &Client{
		conn:   conn,
		client: pb.NewTripServiceClient(conn),
	}, nil
}

func (c *Client) CreateTrip(ctx context.Context, req *pb.CreateTripRequest) (*pb.CreateTripResponse, error) {
	return c.client.CreateTrip(ctx, req)
}

func (c *Client) GetTrip(ctx context.Context, req *pb.GetTripRequest) (*pb.GetTripResponse, error) {
	return c.client.GetTrip(ctx, req)
}

func (c *Client) ListUserTrips(ctx context.Context, req *pb.ListUserTripsRequest) (*pb.ListUserTripsResponse, error) {
	return c.client.ListUserTrips(ctx, req)
}

func (c *Client) UpdateTrip(ctx context.Context, req *pb.UpdateTripRequest) (*pb.UpdateTripResponse, error) {
	return c.client.UpdateTrip(ctx, req)
}

func (c *Client) DeleteTrip(ctx context.Context, req *pb.DeleteTripRequest) (*pb.DeleteTripResponse, error) {
	return c.client.DeleteTrip(ctx, req)
}

func (c *Client) Close() error {
	return c.conn.Close()
}
