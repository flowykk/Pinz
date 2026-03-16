package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
	redis "github.com/redis/go-redis/v9"

	"pinz/backend/api-gateway-service/internal/middleware"
)

type WSHandler struct {
	redis *redis.Client
}

func NewWSHandler(redisClient *redis.Client) *WSHandler {
	return &WSHandler{redis: redisClient}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ServeWS upgrades HTTP connection to WebSocket and subscribes the client to
// per-user Redis Pub/Sub channel pinz:user:{user_id}:events.
func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
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

	ch := pubsub.Channel()

	// Writer loop: forward Redis messages to WebSocket client.
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
				slog.WarnContext(ctx, "ws: write failed", "error", err)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
