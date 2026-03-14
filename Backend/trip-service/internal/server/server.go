package server

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	pb "pinz/backend/trip-service/pkg/proto"
)

func RunGRPCServer(tripService pb.TripServiceServer) error {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = ":50052"
		slog.Warn("GRPC_PORT not set, using :50052")
	}
	lis, err := net.Listen("tcp", port)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	srv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	pb.RegisterTripServiceServer(srv, tripService)

	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthSrv)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	slog.Info("gRPC server listening", "port", port)

	go func() {
		if err := srv.Serve(lis); err != nil {
			slog.Error("gRPC serve error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down gRPC server...")
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	srv.GracefulStop()
	return nil
}
