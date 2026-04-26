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

	r.Get("/health", handlers.HealthCheck)
	r.Get("/.well-known/apple-app-site-association", handlers.AppleAppSiteAssociation)

	// WebSocket endpoints — long-lived; must stay outside chiMW.Timeout, which otherwise
	// cancels the request context after 10s and kills the connection.
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireJWT)
		r.Get("/v1/ws", deps.WSHandler.ServeWS)
		r.Get("/api/v1/trips/creation/{id}/review/ws", deps.WSHandler.ServeTripCreationReviewWS)
		r.Get("/api/v1/trips/{id}/media/add/review/ws", deps.WSHandler.ServeTripAddMediaReviewWS)
	})

	r.Group(func(r chi.Router) {
		r.Use(chiMW.Timeout(10 * time.Second))

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

			r.Route("/profile", func(r chi.Router) {
				r.Use(middleware.RequireJWT)
				r.Get("/", deps.AuthHandler.GetProfile)
				r.Patch("/", deps.AuthHandler.UpdateProfile)
				r.Delete("/", deps.AuthHandler.DeleteAccount)
				r.Post("/change-email", deps.AuthHandler.ChangeEmail)
				r.Post("/confirm-email", deps.AuthHandler.ConfirmEmailChange)
				r.Post("/avatar/upload", deps.AuthHandler.RequestAvatarUpload)
				r.Post("/avatar/confirm", deps.AuthHandler.ConfirmAvatarUpload)
				r.Delete("/avatar", deps.AuthHandler.DeleteAvatar)
				r.Get("/stats", deps.StatisticsHandler.GetProfileStats)
				r.Get("/visited-locations", deps.StatisticsHandler.GetProfileVisitedLocations)
				r.Post("/device-tokens", deps.NotificationHandler.RegisterDeviceToken)
				r.Delete("/device-tokens", deps.NotificationHandler.UnregisterDeviceToken)
				// Желаемые места (ТЗ 1.13).
				r.Get("/desired-places", deps.AuthHandler.ListDesiredPlaces)
				r.Post("/desired-places", deps.AuthHandler.CreateDesiredPlace)
				r.Post("/desired-places/upload-url", deps.AuthHandler.RequestDesiredPlaceImageUpload)
				r.Patch("/desired-places/{place_id}", deps.AuthHandler.UpdateDesiredPlace)
				r.Delete("/desired-places/{place_id}", deps.AuthHandler.DeleteDesiredPlace)
				r.Delete("/desired-places/{place_id}/image", deps.AuthHandler.DeleteDesiredPlaceImage)
			})

			// Публичный профиль другого пользователя (ТЗ 1.7.2): username, avatar, created_at + wishlist.
			r.Route("/users", func(r chi.Router) {
				r.Use(middleware.RequireJWT)
				r.Get("/{id}", deps.AuthHandler.GetPublicUserProfile)
			})

			// Основные операции над путешествиями.
			r.Route("/trips", func(r chi.Router) {
				r.Use(middleware.RequireJWT)
				r.Get("/", deps.TripHandler.ListTrips)
				r.Get("/favourites", deps.TripHandler.ListFavourites)
				r.Post("/", deps.TripHandler.CreateTrip)
				r.Post("/join", deps.TripHandler.JoinTripByToken)
				r.Post("/creation/start", deps.TripHandler.CreateTrip)
				r.Post("/creation/{id}/media/process-grouping", deps.TripHandler.ProcessMediaGrouping)
				r.Post("/creation/{id}/apply-groups-and-process", deps.TripHandler.ApplyGroupsAndProcess)
				r.Get("/creation/{id}/review", deps.TripHandler.GetTripReview)
				r.Post("/creation/{id}/finalize", deps.TripHandler.FinalizeTrip)
				r.Get("/{id}", deps.TripHandler.GetTrip)
				r.Patch("/{id}", deps.TripHandler.UpdateTrip)
				r.Delete("/{id}", deps.TripHandler.DeleteTrip)
				r.Post("/{id}/cover/upload", deps.TripHandler.RequestTripCoverUpload)
				r.Post("/{id}/cover/confirm", deps.TripHandler.ConfirmTripCoverUpload)
				r.Delete("/{id}/cover", deps.TripHandler.DeleteTripCover)
				r.Patch("/{id}/settings", deps.TripHandler.UpdateTripSettings)
				// per-user приватность (ТЗ 6.4-6.7).
				r.Put("/{id}/privacy", deps.TripHandler.UpdateTripPrivacy)
				r.Put("/{id}/pins/{pin_id}/privacy", deps.TripHandler.UpdatePinPrivacy)
				r.Put("/{id}/media/{media_id}/privacy", deps.TripHandler.UpdateMediaPrivacy)
				r.Post("/{id}/invite", deps.TripHandler.GenerateInviteLink)
				r.Post("/{id}/leave", deps.TripHandler.LeaveTrip)
				r.Post("/{id}/publish", deps.TripHandler.PublishTrip)
				r.Post("/{id}/like", deps.TripHandler.LikeTrip)
				r.Post("/{id}/dislike", deps.TripHandler.DislikeTrip)
				r.Post("/{id}/favourite", deps.TripHandler.AddToFavourites)
				r.Delete("/{id}/favourite", deps.TripHandler.RemoveFromFavourites)
				r.Delete("/{id}/participants/{user_id}", deps.TripHandler.RemoveParticipant)
				// кооперативное добавление медиа в READY-трип (ТЗ 5.1-5.3).
				r.Post("/{id}/media/add/start", deps.TripHandler.AddMediaStart)
				r.Post("/{id}/media/add/request-upload-urls", deps.TripHandler.AddMediaRequestUploadUrls)
				r.Post("/{id}/media/add/commit-upload", deps.TripHandler.AddMediaCommitUpload)
				r.Get("/{id}/media/add/session-media", deps.TripHandler.AddMediaGetSessionMedia)
				r.Post("/{id}/media/add/process-grouping", deps.TripHandler.AddMediaProcessGrouping)
				r.Get("/{id}/media/add/grouping", deps.TripHandler.AddMediaGetGrouping)
				r.Post("/{id}/media/add/apply-groups-and-process", deps.TripHandler.AddMediaApplyGroupsAndProcess)
				r.Get("/{id}/media/add/review", deps.TripHandler.AddMediaGetReview)
				r.Post("/{id}/media/add/confirm", deps.TripHandler.AddMediaConfirm)
				r.Post("/{id}/media/add/cancel", deps.TripHandler.AddMediaCancel)
				r.Post("/{id}/media/add/takeover", deps.TripHandler.AddMediaTakeover)
				// фотобатлы и лучшие воспоминания (ТЗ 8).
				r.Post("/{id}/battles", deps.TripHandler.StartBattle)
				r.Post("/{id}/battles/{battle_id}/result", deps.TripHandler.SubmitBattleResult)
				r.Get("/{id}/best-memories", deps.TripHandler.GetBestMemories)
				// Pin RUD + add/remove media (ТЗ 4.2-4.5)
				r.Get("/{id}/pins/{pin_id}", deps.TripHandler.GetPin)
				r.Patch("/{id}/pins/{pin_id}", deps.TripHandler.UpdatePin)
				r.Delete("/{id}/pins/{pin_id}", deps.TripHandler.DeletePin)
				r.Post("/{id}/pins/{pin_id}/media/sessions/start", deps.TripHandler.AddMediaToPinStart)
				r.Post("/{id}/pins/{pin_id}/media/sessions/{sid}/upload-urls", deps.TripHandler.RequestPinMediaUploadUrls)
				r.Post("/{id}/pins/{pin_id}/media/sessions/{sid}/commit-upload", deps.TripHandler.CommitPinMediaUpload)
				r.Post("/{id}/pins/{pin_id}/media/sessions/{sid}/process", deps.TripHandler.ProcessPinMediaAddition)
				r.Get("/{id}/pins/{pin_id}/media/sessions/{sid}/review", deps.TripHandler.GetPinMediaAdditionReview)
				r.Post("/{id}/pins/{pin_id}/media/sessions/{sid}/finalize", deps.TripHandler.FinalizePinMediaAddition)
				r.Post("/{id}/pins/{pin_id}/media/sessions/{sid}/cancel", deps.TripHandler.CancelPinMediaAddition)
				r.Delete("/{id}/pins/{pin_id}/media/{media_id}", deps.TripHandler.RemoveMediaFromPin)
				// Sessioned создание одиночного пина (ТЗ 4.1, 4.6-4.11)
				r.Post("/{id}/pins/creation/start", deps.TripHandler.CreatePinStart)
				r.Post("/{id}/pins/creation/sessions/{sid}/upload-urls", deps.TripHandler.RequestPinCreationUploadUrls)
				r.Post("/{id}/pins/creation/sessions/{sid}/commit-upload", deps.TripHandler.CommitPinCreationUpload)
				r.Post("/{id}/pins/creation/sessions/{sid}/process", deps.TripHandler.ProcessPinCreation)
				r.Get("/{id}/pins/creation/sessions/{sid}/review", deps.TripHandler.GetPinCreationReview)
				r.Post("/{id}/pins/creation/sessions/{sid}/finalize", deps.TripHandler.FinalizePinCreation)
				r.Post("/{id}/pins/creation/sessions/{sid}/cancel", deps.TripHandler.CancelPinCreation)
			})

			r.Route("/feed", func(r chi.Router) {
				r.Use(middleware.RequireJWT)
				r.Get("/", deps.TripHandler.ListFeed)
			})

			// рекомендательная система (ТЗ 9): карта популярных мест по городу/стране.
			r.Route("/recommendations", func(r chi.Router) {
				r.Use(middleware.RequireJWT)
				r.Get("/", deps.TripHandler.GetRecommendations)
				r.Post("/save", deps.TripHandler.SaveRecommendation)
			})

			// текстовый поиск пинов в трипах авторизованного пользователя.
			r.Route("/pins", func(r chi.Router) {
				r.Use(middleware.RequireJWT)
				r.Get("/search", deps.TripHandler.SearchPins)
			})
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
			Addr: ":" + port,
			Handler: r,
			ReadTimeout: 15 * time.Second,
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
