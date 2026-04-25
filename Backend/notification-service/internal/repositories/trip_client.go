package repositories

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "pinz/backend/notification-service/pkg/proto"
)

// TripClientInterface — часть trip-service RPC, нужная worker/scheduler-у.
type TripClientInterface interface {
	ListTripParticipantIDs(ctx context.Context, tripID string) ([]string, error)
	GetNotificationSettings(ctx context.Context, tripID string, userIDs []string) (map[string]bool, error)
	ListAnniversaryTrips(ctx context.Context, todayUnix int64) ([]*pb.NotificationTrip, error)
	ListEndedMonthAgoTrips(ctx context.Context, todayUnix int64) ([]*pb.NotificationTrip, error)
	Close() error
}

// TripClient — gRPC-клиент trip-service, делает проксирование методов +
// OTel-инструментирование. Использует addr из env TRIP_SERVICE_GRPC_ADDRESS.
type TripClient struct {
	conn *grpc.ClientConn
	cli pb.TripServiceClient
}

func NewTripClient() (*TripClient, error) {
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
	return &TripClient{conn: conn, cli: pb.NewTripServiceClient(conn)}, nil
}

// ListTripParticipantIDs — список user_id всех участников трипа.
func (c *TripClient) ListTripParticipantIDs(ctx context.Context, tripID string) ([]string, error) {
	resp, err := c.cli.ListTripParticipants(ctx, &pb.ListTripParticipantsRequest{TripId: tripID})
	if err != nil {
		return nil, err
	}
	return resp.GetUserIds(), nil
}

func (c *TripClient) GetNotificationSettings(ctx context.Context, tripID string, userIDs []string) (map[string]bool, error) {
	if len(userIDs) == 0 {
		return map[string]bool{}, nil
	}
	resp, err := c.cli.GetNotificationSettings(ctx, &pb.GetNotificationSettingsRequest{TripId: tripID, UserIds: userIDs})
	if err != nil {
		return nil, err
	}
	out := resp.GetNotificationsEnabled()
	if out == nil {
		out = map[string]bool{}
	}
	// По умолчанию (нет записи в trip_settings) — notifications_enabled=true.
	for _, uid := range userIDs {
		if _, ok := out[uid]; !ok {
			out[uid] = true
		}
	}
	return out, nil
}

func (c *TripClient) ListAnniversaryTrips(ctx context.Context, todayUnix int64) ([]*pb.NotificationTrip, error) {
	resp, err := c.cli.ListAnniversaryTrips(ctx, &pb.ListAnniversaryTripsRequest{TodayUnix: todayUnix})
	if err != nil {
		return nil, err
	}
	return resp.GetTrips(), nil
}

func (c *TripClient) ListEndedMonthAgoTrips(ctx context.Context, todayUnix int64) ([]*pb.NotificationTrip, error) {
	resp, err := c.cli.ListEndedMonthAgoTrips(ctx, &pb.ListEndedMonthAgoTripsRequest{TodayUnix: todayUnix})
	if err != nil {
		return nil, err
	}
	return resp.GetTrips(), nil
}

func (c *TripClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
