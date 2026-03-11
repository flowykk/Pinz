package server

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	pb "pinz/backend/auth-service/pkg/proto"
)

func RunGRPCServer(authService pb.AuthServiceServer) error {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = ":50051"
		log.Println("GRPC_PORT not set, using :50051")
	}
	lis, err := net.Listen("tcp", port)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	srv := grpc.NewServer()
	pb.RegisterAuthServiceServer(srv, authService)

	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthSrv)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	log.Printf("gRPC server listening on %s", port)

	go func() {
		if err := srv.Serve(lis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down gRPC server...")
	// Announce NOT_SERVING before draining so Kubernetes stops routing traffic
	// to this pod before connections are forcibly closed.
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	srv.GracefulStop()
	return nil
}
