package services

import (
	"strconv"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Add-media-specific gRPC error helpers. Каждый возвращает status.Error с
// прикреплённым ErrorInfo (Reason + Metadata), который api-gateway снимает
// в handleServiceError и превращает в HTTP-код + structured JSON-body.
//
// Коды gRPC остаются общими (FailedPrecondition/PermissionDenied/InvalidArgument),
// различение по Reason идёт в клиентском коде и в api-gateway. Это позволяет
// сохранять семантику gRPC и при этом отдавать фронту точный 410/412/403/422.

// ErrorReason — значения ErrorInfo.Reason, идентифицирующие add-media ошибки.
const (
	ErrorReasonSessionStale  = "SESSION_STALE"  // → HTTP 410 Gone
	ErrorReasonWrongStatus   = "WRONG_STATUS"   // → HTTP 412 Precondition Failed
	ErrorReasonNotInitiator  = "NOT_INITIATOR"  // → HTTP 403 Forbidden
	ErrorReasonLimitExceeded = "LIMIT_EXCEEDED" // → HTTP 422 Unprocessable Entity

	ErrorDomain = "trip-service.add_media"
)

func withErrorInfo(c codes.Code, msg, reason string, metadata map[string]string) error {
	st := status.New(c, msg)
	info := &errdetails.ErrorInfo{
		Reason:   reason,
		Domain:   ErrorDomain,
		Metadata: metadata,
	}
	if augmented, err := st.WithDetails(info); err == nil {
		return augmented.Err()
	}
	return st.Err()
}

// errSessionStale — переданный session_id не совпадает с активной сессией трипа.
// currentSessionID может быть пустым, если сессия только что закрыта.
func errSessionStale(tripID, providedSessionID, currentSessionID string) error {
	return withErrorInfo(codes.FailedPrecondition,
		"session_id does not match active session",
		ErrorReasonSessionStale,
		map[string]string{
			"trip_id":             tripID,
			"provided_session_id": providedSessionID,
			"current_session_id":  currentSessionID,
		})
}

// errNoActiveSession — на трипе нет активной add-media сессии. Используется
// когда ручка ожидает существующую сессию (request-upload-urls, commit-upload,
// review, confirm, cancel и т.п.).
func errNoActiveSession(tripID string) error {
	return withErrorInfo(codes.FailedPrecondition,
		"no active add-media session",
		ErrorReasonSessionStale,
		map[string]string{"trip_id": tripID})
}

// errWrongStatus — trip.status не подходит для запрошенной операции.
// Клиент получает 412 + текущий статус, делает GET /trips/{id} и перерисовывается.
func errWrongStatus(expected, actual string) error {
	return withErrorInfo(codes.FailedPrecondition,
		"trip status does not allow this operation",
		ErrorReasonWrongStatus,
		map[string]string{
			"expected_status": expected,
			"current_status":  actual,
		})
}

// errNotInitiator — мутирующий запрос от не-ведущего до истечения часа.
// takeoverAvailableAt — момент, когда у participant'а появится право перехвата.
func errNotInitiator(currentInitiator string, takeoverAvailableAt time.Time) error {
	md := map[string]string{
		"current_initiator_user_id": currentInitiator,
	}
	if !takeoverAvailableAt.IsZero() {
		md["takeover_available_at"] = takeoverAvailableAt.UTC().Format(time.RFC3339)
	}
	return withErrorInfo(codes.PermissionDenied,
		"not the current initiator",
		ErrorReasonNotInitiator,
		md)
}

// errLimitExceeded — превышение лимитов ТЗ 4.1.2 (500 медиа, 50 видео).
// Клиент получает 422 + remaining_slots, чтобы точно сказать пользователю
// сколько файлов можно докинуть.
func errLimitExceeded(kind string, limit, current int) error {
	remaining := limit - current
	if remaining < 0 {
		remaining = 0
	}
	return withErrorInfo(codes.ResourceExhausted,
		"media limit exceeded",
		ErrorReasonLimitExceeded,
		map[string]string{
			"kind":            kind, // "media" | "video"
			"limit":           strconv.Itoa(limit),
			"current":         strconv.Itoa(current),
			"remaining_slots": strconv.Itoa(remaining),
		})
}
