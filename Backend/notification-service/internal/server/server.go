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

	pb "pinz/backend/notification-service/pkg/proto"
)

func RunGRPCServer(svc pb.NotificationServiceServer) error {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = ":50054"
		slog.Warn("GRPC_PORT not set, using :50054")
	}
	slog.Info("gRPC: binding to port", "port", port)
	lis, err := net.Listen("tcp", port)
	if err != nil {
		slog.Error("gRPC: listen failed", "port", port, "error", err)
		return fmt.Errorf("listen: %w", err)
	}

	srv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(AuthUnaryInterceptor),
	)
	pb.RegisterNotificationServiceServer(srv, svc)

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
