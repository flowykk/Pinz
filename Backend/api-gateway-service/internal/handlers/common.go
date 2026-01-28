package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/api-gateway-service/internal/responses"
)

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

func handleServiceError(w http.ResponseWriter, err error, action string) {
	st, ok := status.FromError(err)
	if !ok {
		log.Printf("%s: %v", action, err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	log.Printf("%s (gRPC %s): %s", action, st.Code().String(), st.Message())
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
	default:
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}
