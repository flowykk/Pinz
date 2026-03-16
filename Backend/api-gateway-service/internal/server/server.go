package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMW "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"pinz/backend/api-gateway-service/internal/di"
	"pinz/backend/api-gateway-service/internal/handlers"
	"pinz/backend/api-gateway-service/internal/middleware"

	_ "pinz/backend/api-gateway-service/docs"
)

type Server struct {
	http.Server
}

func NewServer(deps *di.Dependencies) *Server {
	r := chi.NewRouter()

	// OTel HTTP tracing — must be the outermost middleware so it wraps the full
	// request lifecycle. A second middleware (below) updates the span name with
	// the chi route pattern once routing is complete.
	r.Use(otelhttp.NewMiddleware("api-gateway"))

	// After routing, enrich the active span with the matched route pattern so
	// traces are grouped by endpoint rather than raw URL.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			span := trace.SpanFromContext(r.Context())
			next.ServeHTTP(w, r)
			if chiCtx := chi.RouteContext(r.Context()); chiCtx != nil {
				if pattern := chiCtx.RoutePattern(); pattern != "" {
					span.SetName(r.Method + " " + pattern)
					span.SetAttributes(attribute.String("http.route", pattern))
				}
			}
		})
	})

	r.Use(chiMW.RequestID)
	r.Use(chiMW.Logger)
	r.Use(chiMW.Recoverer)
	r.Use(chiMW.Timeout(10 * time.Second))

	r.Get("/health", handlers.HealthCheck)
	r.Get("/.well-known/apple-app-site-association", handlers.AppleAppSiteAssociation)

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

		r.Route("/trips", func(r chi.Router) {
			r.Use(middleware.RequireJWT)
			r.Get("/", deps.TripHandler.ListTrips)
			r.Post("/", deps.TripHandler.CreateTrip)
			r.Post("/join", deps.TripHandler.JoinTripByToken)
			r.Get("/{id}", deps.TripHandler.GetTrip)
			r.Patch("/{id}", deps.TripHandler.UpdateTrip)
			r.Delete("/{id}", deps.TripHandler.DeleteTrip)
			r.Patch("/{id}/settings", deps.TripHandler.UpdateTripSettings)
			r.Post("/{id}/invite", deps.TripHandler.GenerateInviteLink)
			r.Post("/{id}/leave", deps.TripHandler.LeaveTrip)
			r.Post("/{id}/transfer-admin", deps.TripHandler.TransferAdmin)
			r.Post("/{id}/like", deps.TripHandler.LikeTrip)
			r.Post("/{id}/dislike", deps.TripHandler.DislikeTrip)
			r.Post("/{id}/favourite", deps.TripHandler.AddToFavourites)
			r.Delete("/{id}/favourite", deps.TripHandler.RemoveFromFavourites)
			r.Delete("/{id}/participants/{user_id}", deps.TripHandler.RemoveParticipant)
			r.Post("/{id}/media/process-grouping", deps.TripHandler.ProcessMediaGrouping)
			r.Post("/{id}/apply-groups-and-process", deps.TripHandler.ApplyGroupsAndProcess)
			r.Get("/{id}/review", deps.TripHandler.GetTripReview)
			r.Post("/{id}/finalize", deps.TripHandler.FinalizeTrip)
		})
		r.Route("/feed", func(r chi.Router) {
			r.Use(middleware.RequireJWT)
			r.Get("/", deps.TripHandler.ListFeed)
		})
	})

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
	))

	port := os.Getenv("API_GATEWAY_PORT")
	if port == "" {
		port = "8080"
	}

	return &Server{
		Server: http.Server{
			Addr:         ":" + port,
			Handler:      r,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
		},
	}
}

func (s *Server) Run() error {
	go func() {
		slog.Info("API Gateway listening", "addr", s.Addr)
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("ListenAndServe error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	slog.Info("Server stopped")
	return nil
}
