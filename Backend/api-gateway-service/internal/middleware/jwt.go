package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"pinz/backend/api-gateway-service/internal/metrics"
)

type contextKey string

const UserIDContextKey contextKey = "user_id"

// UserIDFromContext returns the user ID set by RequireJWT (empty if not set).
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(UserIDContextKey).(string)
	return v
}

// RequireJWT parses the Bearer token from Authorization header, validates it with JWT_SECRET_KEY,
// and sets user_id from claims into the request context. Returns 401 if missing or invalid.
func RequireJWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(strings.TrimSpace(auth), "Bearer ") {
			metrics.JWTFailure(r.Context(), "missing")
			respondUnauthorized(w)
			return
		}
		tokenStr := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if tokenStr == "" {
			metrics.JWTFailure(r.Context(), "empty")
			respondUnauthorized(w)
			return
		}
		secret := os.Getenv("JWT_SECRET_KEY")
		if secret == "" {
			metrics.JWTFailure(r.Context(), "no_secret")
			http.Error(w, `{"error":"server misconfiguration"}`, http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			return
		}
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			metrics.JWTFailure(r.Context(), classifyJWTErr(err))
			respondUnauthorized(w)
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			metrics.JWTFailure(r.Context(), "claims_type")
			respondUnauthorized(w)
			return
		}
		userID, _ := claims["user_id"].(string)
		if userID == "" {
			metrics.JWTFailure(r.Context(), "no_user_id")
			respondUnauthorized(w)
			return
		}
		ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func classifyJWTErr(err error) string {
	if err == nil {
		return "invalid"
	}
	switch {
	case strings.Contains(err.Error(), "expired"):
		return "expired"
	case strings.Contains(err.Error(), "signature"):
		return "bad_signature"
	case strings.Contains(err.Error(), "malformed"):
		return "malformed"
	}
	return "invalid"
}

func respondUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
}
