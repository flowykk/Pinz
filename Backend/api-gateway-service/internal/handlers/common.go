package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/api-gateway-service/internal/responses"
)

// Reason-коды, выставляемые trip-service (add-media ошибки, PINZ-172).
// Синхронизировано с trip-service/internal/services/add_media_errors.go.
const (
	reasonSessionStale  = "SESSION_STALE"  // устаревший session_id → 410 Gone
	reasonWrongStatus   = "WRONG_STATUS"   // неподходящий trip.status → 412
	reasonNotInitiator  = "NOT_INITIATOR"  // мутация не-ведущим до истечения часа → 403
	reasonLimitExceeded = "LIMIT_EXCEEDED" // ТЗ 4.1.2 (500 медиа, 50 видео) → 422
)

func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

func decodeJSONBody(r *http.Request, dst interface{}) error {
	if r.Header.Get("Content-Type") != "application/json" {
		return fmt.Errorf("content-type must be application/json")
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	_ = r.Body.Close()
	return json.Unmarshal(body, dst)
}

func respondJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	data, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal error"}`))
		return
	}
	w.WriteHeader(statusCode)
	_, _ = w.Write(data)
}

func respondError(w http.ResponseWriter, statusCode int, message string) {
	respondJSON(w, statusCode, responses.ErrorResponse{Error: message})
}

func handleServiceError(w http.ResponseWriter, r *http.Request, err error, action string) {
	st, ok := status.FromError(err)
	if !ok {
		slog.ErrorContext(r.Context(), "service error", "action", action, "error", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	slog.WarnContext(r.Context(), "grpc error", "action", action, "code", st.Code().String(), "msg", st.Message())

	// Сначала смотрим на prикреплённый ErrorInfo.Reason — если сервис явно указал
	// семантическую причину (SESSION_STALE/WRONG_STATUS/NOT_INITIATOR/LIMIT_EXCEEDED),
	// мапим в точный HTTP-код и прикладываем metadata в тело.
	if reason, metadata := extractErrorInfo(st); reason != "" {
		if httpCode, ok := reasonToHTTPStatus[reason]; ok {
			respondJSON(w, httpCode, responses.ErrorResponse{
				Error:   st.Message(),
				Reason:  reason,
				Details: metadata,
			})
			return
		}
	}

	switch st.Code() {
	case codes.InvalidArgument:
		respondError(w, http.StatusBadRequest, st.Message())
	case codes.Unauthenticated:
		respondError(w, http.StatusUnauthorized, st.Message())
	case codes.PermissionDenied:
		respondError(w, http.StatusForbidden, st.Message())
	case codes.NotFound:
		respondError(w, http.StatusNotFound, st.Message())
	case codes.AlreadyExists:
		respondError(w, http.StatusConflict, st.Message())
	case codes.Unavailable:
		respondError(w, http.StatusServiceUnavailable, "service unavailable")
	case codes.DeadlineExceeded:
		respondError(w, http.StatusGatewayTimeout, st.Message())
	case codes.Unimplemented:
		respondError(w, http.StatusNotImplemented, st.Message())
	case codes.FailedPrecondition:
		respondError(w, http.StatusConflict, st.Message())
	case codes.ResourceExhausted:
		respondError(w, http.StatusTooManyRequests, st.Message())
	default:
		respondError(w, http.StatusInternalServerError, st.Message())
	}
}

// reasonToHTTPStatus — маппинг известных ErrorInfo.Reason в HTTP-код. Соответствует
// документу addMediaParallelFlow.md (корнер-кейсы).
var reasonToHTTPStatus = map[string]int{
	reasonSessionStale:  http.StatusGone,               // 410
	reasonWrongStatus:   http.StatusPreconditionFailed, // 412
	reasonNotInitiator:  http.StatusForbidden,          // 403
	reasonLimitExceeded: http.StatusUnprocessableEntity, // 422
}

// extractErrorInfo достаёт первый ErrorInfo из gRPC status.Details().
// Возвращает ("", nil), если ErrorInfo не прикреплён.
func extractErrorInfo(st *status.Status) (reason string, metadata map[string]string) {
	for _, d := range st.Details() {
		if info, ok := d.(*errdetails.ErrorInfo); ok {
			return info.GetReason(), info.GetMetadata()
		}
	}
	return "", nil
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
		"service": "api-gateway",
		"timestamp": fmt.Sprintf("%d", time.Now().Unix()),
	})
}

// AppleAppSiteAssociation serves the AASA file required by Apple CDN on app install.
// Must be reachable at /.well-known/apple-app-site-association with no redirects.
func AppleAppSiteAssociation(w http.ResponseWriter, r *http.Request) {
	const aasa = `{
 "applinks": {
 "apps": [],
 "details": [
 {
 "appID": "ABNY5S6RA5.io.tuist.Pinz",
 "paths": [
 "/join/*",
 "/reset-password*"
 ]
 }
 ]
 },
 "webcredentials": {
 "apps": ["ABNY5S6RA5.io.tuist.Pinz"]
 }
}`
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(aasa))
}

// unixToRFC3339 конвертирует unix timestamp (секунды) в RFC3339-строку UTC.
// Если unix == 0 — возвращает пустую строку (DTO с omitempty не сериализует поле).
func unixToRFC3339(unix int64) string {
	if unix == 0 {
		return ""
	}
	return time.Unix(unix, 0).UTC().Format(time.RFC3339)
}
