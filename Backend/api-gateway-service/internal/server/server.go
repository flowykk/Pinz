package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMW "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"pinz/backend/api-gateway-service/internal/di"
	"pinz/backend/api-gateway-service/internal/handlers"

	_ "pinz/backend/api-gateway-service/docs"
)

type Server struct {
	http.Server
}

func NewServer(deps *di.Dependencies) *Server {
	r := chi.NewRouter()
	r.Use(chiMW.RequestID)
	r.Use(chiMW.Logger)
	r.Use(chiMW.Recoverer)
	r.Use(chiMW.Timeout(10 * time.Second))

	// Health check endpoint
	r.Get("/health", handlers.HealthCheck)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/email", deps.AuthHandler.SubmitEmail)
		r.Post("/auth/verify-email", deps.AuthHandler.VerifyEmailCode)
		r.Post("/auth/passkey/register/begin", deps.AuthHandler.PasskeyRegisterBegin)
		r.Post("/auth/passkey/register/finish", deps.AuthHandler.PasskeyRegisterFinish)
		r.Post("/auth/passkey/login/begin", deps.AuthHandler.PasskeyLoginBegin)
		r.Post("/auth/passkey/login/finish", deps.AuthHandler.PasskeyLoginFinish)
		r.Post("/auth/refresh", deps.AuthHandler.RefreshToken)
		r.Post("/auth/logout", deps.AuthHandler.Logout)
		r.Post("/auth/dev-login", deps.AuthHandler.DevLogin)
	})

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
	))

	port := os.Getenv("API_GATEWAY_PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	return &Server{
		Server: http.Server{
			Addr:         addr,
			Handler:      r,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
		},
	}
}

func (s *Server) Run() error {
	go func() {
		log.Printf("API Gateway listening on %s", s.Addr)
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("ListenAndServe: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	log.Println("Server stopped")
	return nil
}
