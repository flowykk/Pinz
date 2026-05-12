package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"

	"pinz/backend/trip-service/internal/metrics"
)

const (
	tripEventsStream = "pinz:trip:events"
	statsEventsStream = "pinz:stats:events"
	mlTasksStream = "pinz:trip:ml:tasks"
	mlResultsStream = "pinz:trip:ml:results"
	mlContextPrefix = "pinz:trip:ml:context:"
	privacyEventsStream = "pinz:trip:privacy:events"
	pinUploadTasksStream = "pinz:trip:pin_upload:tasks"
	PinUploadMLTasksStream = "pinz:trip:pin_upload:ml:tasks"
	PinUploadMLResultsStream = "pinz:trip:pin_upload:ml:results"

	tripEventsChannelPrefix = "pinz:trip:"
	tripEventsChannelSuffix = ":events"

	// wsStreamMaxLen ограничивает длину per-trip и per-user WS-стримов, чтобы
	// они не росли бесконечно. Approx=true у XADD допускает небольшой оверхед.
	wsStreamMaxLen = 100
	// wsStreamTTL — TTL на stream key, ставится после каждого XADD. Фикcит
	// orphan streams на случай, если DeleteTripEventStream не был вызван.
	wsStreamTTL = 1 * time.Hour
)

// константы event-type для add-media флоу. Публикуются через
// PublishTripEventWS на per-trip WS-канале, клиент разбирает по "type" в payload.
const (
	EventTripStatusChanged = "TRIP_STATUS_CHANGED"
	EventTripProcessingCompleted = "TRIP_PROCESSING_COMPLETED"
	EventAddMediaProgress = "ADD_MEDIA_PROGRESS"
	EventAddMediaInitiatorChanged = "ADD_MEDIA_INITIATOR_CHANGED"
	EventAddMediaSessionCompleted = "ADD_MEDIA_SESSION_COMPLETED"
	EventPinUploadProcessingCompleted = "PIN_UPLOAD_PROCESSING_COMPLETED"
)

// RedisRepository provides Redis client and trip event streaming for Notification/Statistics
// services, as well as Pub/Sub channels for WebSocket notifications via API Gateway.
type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{client: client}
}

// InitRedisClient creates a Redis client. Returns (nil, nil) if REDIS_ADDR and REDIS_URL are both empty (optional Redis).
func InitRedisClient() (*redis.Client, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		if u := os.Getenv("REDIS_URL"); u != "" {
			opt, err := redis.ParseURL(u)
			if err != nil {
				return nil, err
			}
			client := redis.NewClient(opt)
			if err := client.Ping(context.Background()).Err(); err != nil {
				return nil, fmt.Errorf("redis ping: %w", err)
			}
			instrumentRedis(client)
			return client, nil
		}
		return nil, nil
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	instrumentRedis(client)
	return client, nil
}

func instrumentRedis(client *redis.Client) {
	if err := redisotel.InstrumentTracing(client); err != nil {
		slog.Warn("redis tracing instrumentation failed", "error", err)
	}
	if err := redisotel.InstrumentMetrics(client); err != nil {
		slog.Warn("redis metrics instrumentation failed", "error", err)
	}
}

// PublishStatsEvent отправляет событие в stream pinz:stats:events для statistics-service.
// Формат: event_type (string), trip_id (string, опционально), user_ids (JSON []string),
// payload (JSON map[string]any). Если Redis не настроен — no-op.
func (r *RedisRepository) PublishStatsEvent(ctx context.Context, eventType, tripID string, userIDs []string, payload map[string]any) error {
	if r == nil || r.client == nil {
		return nil
	}
	vals := map[string]interface{}{
		"event_type": eventType,
	}
	if tripID != "" {
		vals["trip_id"] = tripID
	}
	if len(userIDs) > 0 {
		if b, err := json.Marshal(userIDs); err == nil {
			vals["user_ids"] = string(b)
		}
	}
	if len(payload) > 0 {
		if b, err := json.Marshal(payload); err == nil {
			vals["payload"] = string(b)
		}
	}
	if err := r.client.XAdd(ctx, &redis.XAddArgs{Stream: statsEventsStream, Values: vals}).Err(); err != nil {
		slog.WarnContext(ctx, "PublishStatsEvent failed", "event", eventType, "trip_id", tripID, "error", err)
		metrics.StreamPublished(ctx, statsEventsStream, eventType, "error")
		return err
	}
	metrics.StreamPublished(ctx, statsEventsStream, eventType, "success")
	return nil
}

// PublishGeoRequest публикует PIN_LOCATIONS_REQUESTED в pinz:stats:events.
// statistics-service consumer'ит это событие, идёт в BigDataCloud, заполняет
// master geo_registry/trip_locations и публикует PIN_LOCATIONS_RESOLVED
// Геокодинг — некритичный путь: при недоступности Redis/statistics событие
// просто остаётся в стриме, трип создаётся с пустым location_name.
func (r *RedisRepository) PublishGeoRequest(ctx context.Context, tripID string, pins []GeoRequestPin) error {
	if r == nil || r.client == nil {
		return nil
	}
	if tripID == "" || len(pins) == 0 {
		return nil
	}
	payload := map[string]any{"pins": pins}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	vals := map[string]any{
		"event_type": "PIN_LOCATIONS_REQUESTED",
		"trip_id":    tripID,
		"payload":    string(b),
	}
	if err := r.client.XAdd(ctx, &redis.XAddArgs{Stream: statsEventsStream, Values: vals}).Err(); err != nil {
		slog.WarnContext(ctx, "PublishGeoRequest failed", "trip_id", tripID, "error", err)
		metrics.StreamPublished(ctx, statsEventsStream, "PIN_LOCATIONS_REQUESTED", "error")
		return err
	}
	metrics.StreamPublished(ctx, statsEventsStream, "PIN_LOCATIONS_REQUESTED", "success")
	return nil
}

// PublishTripEvent adds an event to the trip events stream for Notification/Statistics services.
func (r *RedisRepository) PublishTripEvent(ctx context.Context, eventType string, tripID, userID string) error {
	vals := map[string]interface{}{
		"event_type": eventType,
		"trip_id": tripID,
	}
	if userID != "" {
		vals["user_id"] = userID
	}
	err := r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: tripEventsStream,
		Values: vals,
	}).Err()
	if err != nil {
		slog.WarnContext(ctx, "PublishTripEvent failed", "event", eventType, "trip_id", tripID, "error", err)
		metrics.StreamPublished(ctx, tripEventsStream, eventType, "error")
		return err
	}
	metrics.StreamPublished(ctx, tripEventsStream, eventType, "success")
	return nil
}

func (r *RedisRepository) ReadMLResults(ctx context.Context, group, consumer string, count int64, blockMs int64) ([]redis.XStream, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: group,
		Consumer: consumer,
		Streams: []string{mlResultsStream, ">"},
		Count: count,
		Block: time.Duration(blockMs) * time.Millisecond,
	}).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	return streams, nil
}

// AddPinUploadTask публикует задачу async ML pin-upload сессии.
func (r *RedisRepository) AddPinUploadTask(ctx context.Context, tripID, sessionID string, targetPinID *string, initiatorUserID string) error {
	if r == nil || r.client == nil {
		return nil
	}
	vals := map[string]interface{}{
		"trip_id":           tripID,
		"session_id":        sessionID,
		"initiator_user_id": initiatorUserID,
	}
	if targetPinID != nil {
		vals["target_pin_id"] = *targetPinID
	}
	if err := r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: pinUploadTasksStream,
		Values: vals,
	}).Err(); err != nil {
		slog.WarnContext(ctx, "AddPinUploadTask failed", "session_id", sessionID, "trip_id", tripID, "error", err)
		metrics.StreamPublished(ctx, pinUploadTasksStream, "pin_upload_task", "error")
		return err
	}
	metrics.StreamPublished(ctx, pinUploadTasksStream, "pin_upload_task", "success")
	return nil
}

// AddMLTask adds a task to the ML/processing stream (for worker: apply-groups-and-process flow).
func (r *RedisRepository) AddMLTask(ctx context.Context, tripID string) error {
	err := r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: mlTasksStream,
		Values: map[string]interface{}{"trip_id": tripID},
	}).Err()
	if err != nil {
		slog.WarnContext(ctx, "AddMLTask failed", "trip_id", tripID, "error", err)
		metrics.StreamPublished(ctx, mlTasksStream, "ml_task", "error")
		return err
	}
	metrics.StreamPublished(ctx, mlTasksStream, "ml_task", "success")
	return nil
}

// AddMLTaskWithFlow adds a task with flow marker and optional new pin ids (for add-media).
func (r *RedisRepository) AddMLTaskWithFlow(ctx context.Context, tripID, flow string, newPinIDs []string) error {
	vals := map[string]interface{}{"trip_id": tripID}
	if flow != "" {
		vals["flow"] = flow
	}
	if len(newPinIDs) > 0 {
		if b, err := json.Marshal(newPinIDs); err == nil {
			vals["new_pin_ids"] = string(b)
		}
	}
	err := r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: mlTasksStream,
		Values: vals,
	}).Err()
	if err != nil {
		metrics.StreamPublished(ctx, mlTasksStream, "ml_task", "error")
		return err
	}
	metrics.StreamPublished(ctx, mlTasksStream, "ml_task", "success")
	return nil
}

// SetMLContext stores flow-scoped context for later filtering ML results (TTL).
func (r *RedisRepository) SetMLContext(ctx context.Context, tripID, flow string, newPinIDs []string, ttl time.Duration) error {
	if tripID == "" {
		return nil
	}
	vals := map[string]interface{}{}
	if flow != "" {
		vals["flow"] = flow
	}
	if len(newPinIDs) > 0 {
		if b, err := json.Marshal(newPinIDs); err == nil {
			vals["new_pin_ids"] = string(b)
		}
	}
	if len(vals) == 0 {
		return nil
	}
	key := mlContextPrefix + tripID
	if err := r.client.HSet(ctx, key, vals).Err(); err != nil {
		return err
	}
	if ttl > 0 {
		_ = r.client.Expire(ctx, key, ttl).Err()
	}
	return nil
}

// GetMLContext returns flow and new_pin_ids (if present) for the trip.
func (r *RedisRepository) GetMLContext(ctx context.Context, tripID string) (flow string, newPinIDs []string, err error) {
	if tripID == "" {
		return "", nil, nil
	}
	key := mlContextPrefix + tripID
	m, err := r.client.HGetAll(ctx, key).Result()
	if err != nil || len(m) == 0 {
		return "", nil, err
	}
	flow = m["flow"]
	if s := m["new_pin_ids"]; s != "" {
		_ = json.Unmarshal([]byte(s), &newPinIDs)
	}
	return flow, newPinIDs, nil
}

// PublishPrivacyEvent publishes a PRIVACY_CHANGED event for worker aggregation.
func (r *RedisRepository) PublishPrivacyEvent(ctx context.Context, objectType, objectID, tripID, userID, privacyLevel string) error {
	if r == nil || r.client == nil {
		return nil
	}
	err := r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: privacyEventsStream,
		Values: map[string]interface{}{
			"event_type": "PRIVACY_CHANGED",
			"object_type": objectType,
			"object_id": objectID,
			"trip_id": tripID,
			"user_id": userID,
			"privacy_level": privacyLevel,
		},
	}).Err()
	if err != nil {
		metrics.StreamPublished(ctx, privacyEventsStream, "PRIVACY_CHANGED", "error")
		return err
	}
	metrics.StreamPublished(ctx, privacyEventsStream, "PRIVACY_CHANGED", "success")
	return nil
}

// PublishTripEventWS — XADD в per-trip WS-stream pinz:trip:{id}:events.
func (r *RedisRepository) PublishTripEventWS(ctx context.Context, tripID string, eventType string, payload map[string]interface{}) error {
	if r == nil || r.client == nil || tripID == "" {
		return nil
	}
	if payload == nil {
		payload = map[string]interface{}{}
	}
	if _, ok := payload["trip_id"]; !ok {
		payload["trip_id"] = tripID
	}
	data, err := json.Marshal(map[string]interface{}{
		"event":   eventType,
		"payload": payload,
	})
	if err != nil {
		slog.WarnContext(ctx, "PublishTripEventWS marshal failed", "trip_id", tripID, "event", eventType, "error", err)
		return err
	}
	r.publishWSStream(ctx, tripEventsChannelPrefix+tripID+tripEventsChannelSuffix, data, eventType)
	return nil
}

// publishWSStream делает XADD + best-effort EXPIRE на WS-stream.
func (r *RedisRepository) publishWSStream(ctx context.Context, key string, data []byte, eventType string) {
	if err := r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: key,
		MaxLen: wsStreamMaxLen,
		Approx: true,
		Values: map[string]interface{}{"data": data},
	}).Err(); err != nil {
		slog.WarnContext(ctx, "PublishTripEventWS xadd failed", "stream", key, "event", eventType, "error", err)
		metrics.StreamPublished(ctx, "pinz:trip:*:events", eventType, "error")
		return
	}
	metrics.StreamPublished(ctx, "pinz:trip:*:events", eventType, "success")
	if err := r.client.Expire(ctx, key, wsStreamTTL).Err(); err != nil {
		slog.WarnContext(ctx, "PublishTripEventWS expire failed", "stream", key, "error", err)
	}
}

func (r *RedisRepository) AddMLTaskFull(ctx context.Context, tripID, flow, pinsJSON string, newPinIDs []string, presignExpiresAtUnix int64) error {
	if r == nil || r.client == nil {
		return nil
	}
	vals := map[string]interface{}{
		"trip_id": tripID,
		"pins":    pinsJSON,
		"presign_expires_at_unix": strconv.FormatInt(presignExpiresAtUnix, 10),
	}
	if flow != "" {
		vals["flow"] = flow
	}
	if len(newPinIDs) > 0 {
		if b, err := json.Marshal(newPinIDs); err == nil {
			vals["new_pin_ids"] = string(b)
		}
	}
	if err := r.client.XAdd(ctx, &redis.XAddArgs{Stream: mlTasksStream, Values: vals}).Err(); err != nil {
		metrics.StreamPublished(ctx, mlTasksStream, "ml_task", "error")
		return err
	}
	metrics.StreamPublished(ctx, mlTasksStream, "ml_task", "success")
	return nil
}

func (r *RedisRepository) AddPinUploadMLTask(ctx context.Context, tripID, sessionID, targetPinID, newMediaJSON, existingMediaJSON string, presignExpiresAtUnix int64) error {
	if r == nil || r.client == nil {
		return nil
	}
	vals := map[string]interface{}{
		"trip_id":    tripID,
		"session_id": sessionID,
		"new_media":  newMediaJSON,
		"presign_expires_at_unix": strconv.FormatInt(presignExpiresAtUnix, 10),
	}
	if targetPinID != "" {
		vals["target_pin_id"] = targetPinID
	}
	if existingMediaJSON != "" {
		vals["existing_media"] = existingMediaJSON
	}
	if err := r.client.XAdd(ctx, &redis.XAddArgs{Stream: PinUploadMLTasksStream, Values: vals}).Err(); err != nil {
		metrics.StreamPublished(ctx, PinUploadMLTasksStream, "ml_task", "error")
		return err
	}
	metrics.StreamPublished(ctx, PinUploadMLTasksStream, "ml_task", "success")
	return nil
}

func (r *RedisRepository) ReadPinUploadMLResults(ctx context.Context, group, consumer string, count int64, blockMs int64) ([]redis.XStream, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{PinUploadMLResultsStream, ">"},
		Count:    count,
		Block:    time.Duration(blockMs) * time.Millisecond,
	}).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	return streams, nil
}

// DeleteTripEventStream удаляет per-trip WS-stream. Вызывается из DeleteTrip,
// чтобы не копить orphan-ключи в Redis.
func (r *RedisRepository) DeleteTripEventStream(ctx context.Context, tripID string) error {
	if r == nil || r.client == nil || tripID == "" {
		return nil
	}
	key := tripEventsChannelPrefix + tripID + tripEventsChannelSuffix
	if err := r.client.Del(ctx, key).Err(); err != nil {
		slog.WarnContext(ctx, "DeleteTripEventStream failed", "stream", key, "error", err)
		return err
	}
	return nil
}
