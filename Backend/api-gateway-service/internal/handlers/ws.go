package handlers

import (
	"context"
	"encoding/json"
	"errors"
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
	redis *redis.Client
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
	wsPingPeriod = 30 * time.Second
	// wsXReadBlock — сколько XREAD блокирует при отсутствии новых сообщений.
	// Короче ping-period, чтобы пинги не задерживались из-за блокирующего чтения
	// (XREAD идёт в отдельной goroutine, так что это не критично, но полезно
	// для cooperative cancellation при ctx.Done).
	wsXReadBlock = 15 * time.Second
	// wsSubChanBuffer — буфер канала между XRead-goroutine и pumpWS, чтобы
	// медленный клиент не блокировал чтение из Redis.
	wsSubChanBuffer = 16
)

type wsEvent struct {
	Event string `json:"event"`
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

	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	streamKey := "pinz:trip:" + tripID + ":events"
	msgs := subscribeStream(subCtx, h.redis, streamKey)

	pumpWS(ctx, conn, msgs, nil)
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

	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	streamKey := "pinz:user:" + userID + ":events"
	msgs := subscribeStream(subCtx, h.redis, streamKey)

	pumpWS(ctx, conn, msgs, allow)
}

// subscribeStream запускает goroutine, читающую Redis Stream через XRead без
// consumer-group (broadcast-семантика, совместимая со старым Pub/Sub). Первый
// XREAD идёт от "0-0" — забирает backfill, что устраняет гонку publish-до-
// subscribe. Канал закрывается при отмене ctx.
func subscribeStream(ctx context.Context, client *redis.Client, key string) <-chan []byte {
	out := make(chan []byte, wsSubChanBuffer)
	go func() {
		defer close(out)
		lastID := "0-0"
		for {
			if ctx.Err() != nil {
				return
			}
			res, err := client.XRead(ctx, &redis.XReadArgs{
				Streams: []string{key, lastID},
				Block: wsXReadBlock,
				Count: wsSubChanBuffer,
			}).Result()
			if err != nil {
				if errors.Is(err, redis.Nil) {
					continue
				}
				if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}
				slog.WarnContext(ctx, "ws: XRead error", "stream", key, "error", err)
				continue
			}
			for _, stream := range res {
				for _, m := range stream.Messages {
					data, _ := m.Values["data"].(string)
					if data == "" {
						lastID = m.ID
						continue
					}
					select {
					case out <- []byte(data):
					case <-ctx.Done():
						return
					}
					lastID = m.ID
				}
			}
		}
	}()
	return out
}

// pumpWS runs the read/write loop for a single WebSocket connection: forwards filtered
// WS-stream payloads, heartbeats with ping/pong, and detects remote close via reader.
func pumpWS(ctx context.Context, conn *websocket.Conn, sub <-chan []byte, allow func(wsEvent) bool) {
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
		case payload, ok := <-sub:
			if !ok {
				return
			}
			if allow != nil {
				var ev wsEvent
				if err := json.Unmarshal(payload, &ev); err != nil {
					continue
				}
				if !allow(ev) {
					continue
				}
			}
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
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
