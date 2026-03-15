package trip

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

// withUserIDMetadata adds x-user-id from context to outgoing gRPC metadata.
func withUserIDMetadata(ctx context.Context) context.Context {
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, metadataUserIDKey, userID)
}

func (c *Client) CreateTrip(ctx context.Context, req *pb.CreateTripRequest) (*pb.CreateTripResponse, error) {
	return c.client.CreateTrip(withUserIDMetadata(ctx), req)
}

func (c *Client) GetTrip(ctx context.Context, req *pb.GetTripRequest) (*pb.GetTripResponse, error) {
	return c.client.GetTrip(withUserIDMetadata(ctx), req)
}

func (c *Client) ListUserTrips(ctx context.Context, req *pb.ListUserTripsRequest) (*pb.ListUserTripsResponse, error) {
	return c.client.ListUserTrips(withUserIDMetadata(ctx), req)
}

func (c *Client) UpdateTrip(ctx context.Context, req *pb.UpdateTripRequest) (*pb.UpdateTripResponse, error) {
	return c.client.UpdateTrip(withUserIDMetadata(ctx), req)
}

func (c *Client) DeleteTrip(ctx context.Context, req *pb.DeleteTripRequest) (*pb.DeleteTripResponse, error) {
	return c.client.DeleteTrip(withUserIDMetadata(ctx), req)
}

func (c *Client) GenerateInviteLink(ctx context.Context, req *pb.GenerateInviteLinkRequest) (*pb.GenerateInviteLinkResponse, error) {
	return c.client.GenerateInviteLink(withUserIDMetadata(ctx), req)
}

func (c *Client) JoinTripByToken(ctx context.Context, req *pb.JoinTripByTokenRequest) (*pb.JoinTripByTokenResponse, error) {
	return c.client.JoinTripByToken(withUserIDMetadata(ctx), req)
}

func (c *Client) RemoveParticipant(ctx context.Context, req *pb.RemoveParticipantRequest) (*pb.RemoveParticipantResponse, error) {
	return c.client.RemoveParticipant(withUserIDMetadata(ctx), req)
}

func (c *Client) LeaveTrip(ctx context.Context, req *pb.LeaveTripRequest) (*pb.LeaveTripResponse, error) {
	return c.client.LeaveTrip(withUserIDMetadata(ctx), req)
}

func (c *Client) TransferAdmin(ctx context.Context, req *pb.TransferAdminRequest) (*pb.TransferAdminResponse, error) {
	return c.client.TransferAdmin(withUserIDMetadata(ctx), req)
}

func (c *Client) ProcessMediaGrouping(ctx context.Context, req *pb.ProcessMediaGroupingRequest) (*pb.ProcessMediaGroupingResponse, error) {
	return c.client.ProcessMediaGrouping(withUserIDMetadata(ctx), req)
}

func (c *Client) ApplyGroupsAndProcess(ctx context.Context, req *pb.ApplyGroupsAndProcessRequest) (*pb.ApplyGroupsAndProcessResponse, error) {
	return c.client.ApplyGroupsAndProcess(withUserIDMetadata(ctx), req)
}

func (c *Client) GetTripReview(ctx context.Context, req *pb.GetTripReviewRequest) (*pb.GetTripReviewResponse, error) {
	return c.client.GetTripReview(withUserIDMetadata(ctx), req)
}

func (c *Client) FinalizeTrip(ctx context.Context, req *pb.FinalizeTripRequest) (*pb.FinalizeTripResponse, error) {
	return c.client.FinalizeTrip(withUserIDMetadata(ctx), req)
}

func (c *Client) Close() error {
	return c.conn.Close()
}
