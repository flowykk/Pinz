package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	redis "github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	"pinz/backend/api-gateway-service/internal/middleware"
	"pinz/backend/api-gateway-service/pkg/proto"
)

// WSTripClient is the minimal slice of TripClient the WS handler needs to
// authorize participation before subscribing to a per-trip Pub/Sub channel.
type WSTripClient interface {
	GetTrip(ctx context.Context, req *proto.GetTripRequest) (*proto.GetTripResponse, error)
}

type WSHandler struct {
	redis      *redis.Client
	tripClient WSTripClient
}

func NewWSHandler(redisClient *redis.Client, tripClient WSTripClient) *WSHandler {
	return &WSHandler{redis: redisClient, tripClient: tripClient}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

const (
	wsWriteTimeout = 10 * time.Second
	wsReadDeadline = 60 * time.Second
	wsPingPeriod   = 30 * time.Second
)

type wsEvent struct {
	Event   string                 `json:"event"`
	Payload map[string]interface{} `json:"payload"`
}

// ServeWS upgrades the HTTP connection to WebSocket and streams every message from
// pinz:user:{user_id}:events with no filtering. Used for cross-cutting per-user
// notifications not bound to a specific trip.
func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	h.serveUserWS(w, r, nil)
}

// ServeTripCreationReviewWS serves the review-stage WebSocket for the trip creation flow.
// Only messages with payload.trip_id == {id} reach the client.
func (h *WSHandler) ServeTripCreationReviewWS(w http.ResponseWriter, r *http.Request) {
	h.serveTripWS(w, r, chi.URLParam(r, "id"))
}

// ServeTripAddMediaReviewWS serves the review-stage WebSocket for the add-media flow (ТЗ 5.3).
// Contract is identical to ServeTripCreationReviewWS.
func (h *WSHandler) ServeTripAddMediaReviewWS(w http.ResponseWriter, r *http.Request) {
	h.serveTripWS(w, r, chi.URLParam(r, "id"))
}

func (h *WSHandler) serveTripWS(w http.ResponseWriter, r *http.Request, tripID string) {
	if tripID == "" {
		http.Error(w, "trip_id required", http.StatusBadRequest)
		return
	}
	if h.redis == nil {
		http.Error(w, "websocket not available", http.StatusServiceUnavailable)
		return
	}
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if h.tripClient == nil {
		http.Error(w, "trip service unavailable", http.StatusServiceUnavailable)
		return
	}
	// Authorize: fail early if the user is not a participant of this trip. GetTrip
	// returns PermissionDenied for non-participants, NotFound for missing trips.
	if _, err := h.tripClient.GetTrip(ctx, &proto.GetTripRequest{TripId: tripID, UserId: userID}); err != nil {
		status, msg := tripAccessError(err)
		http.Error(w, msg, status)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.WarnContext(ctx, "ws: upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	channel := "pinz:trip:" + tripID + ":events"
	pubsub := h.redis.Subscribe(ctx, channel)
	defer pubsub.Close()

	pumpWS(ctx, conn, pubsub.Channel(), nil)
}

func tripAccessError(err error) (int, string) {
	st, ok := grpcstatus.FromError(err)
	if !ok {
		return http.StatusInternalServerError, "trip lookup failed"
	}
	switch st.Code() {
	case codes.PermissionDenied:
		return http.StatusForbidden, "not a trip participant"
	case codes.NotFound:
		return http.StatusNotFound, "trip not found"
	case codes.Unauthenticated:
		return http.StatusUnauthorized, "unauthorized"
	case codes.InvalidArgument:
		return http.StatusBadRequest, st.Message()
	default:
		return http.StatusBadGateway, "trip service error"
	}
}

func (h *WSHandler) serveUserWS(w http.ResponseWriter, r *http.Request, allow func(wsEvent) bool) {
	if h.redis == nil {
		http.Error(w, "websocket not available", http.StatusServiceUnavailable)
		return
	}
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.WarnContext(ctx, "ws: upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	channel := "pinz:user:" + userID + ":events"
	pubsub := h.redis.Subscribe(ctx, channel)
	defer pubsub.Close()

	pumpWS(ctx, conn, pubsub.Channel(), allow)
}

// pumpWS runs the read/write loop for a single WebSocket connection: forwards filtered
// Redis Pub/Sub payloads, heartbeats with ping/pong, and detects remote close via reader.
func pumpWS(ctx context.Context, conn *websocket.Conn, sub <-chan *redis.Message, allow func(wsEvent) bool) {
	_ = conn.SetReadDeadline(time.Now().Add(wsReadDeadline))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsReadDeadline))
	})

	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ping := time.NewTicker(wsPingPeriod)
	defer ping.Stop()

	for {
		select {
		case msg, ok := <-sub:
			if !ok {
				return
			}
			if allow != nil {
				var ev wsEvent
				if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
					continue
				}
				if !allow(ev) {
					continue
				}
			}
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
				slog.WarnContext(ctx, "ws: write failed", "error", err)
				return
			}
		case <-ping.C:
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(wsWriteTimeout)); err != nil {
				slog.WarnContext(ctx, "ws: ping failed", "error", err)
				return
			}
		case <-readerDone:
			return
		case <-ctx.Done():
			return
		}
	}
}
