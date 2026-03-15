package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
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
			respondUnauthorized(w)
			return
		}
		tokenStr := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if tokenStr == "" {
			respondUnauthorized(w)
			return
		}
		secret := os.Getenv("JWT_SECRET_KEY")
		if secret == "" {
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
			respondUnauthorized(w)
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			respondUnauthorized(w)
			return
		}
		userID, _ := claims["user_id"].(string)
		if userID == "" {
			respondUnauthorized(w)
			return
		}
		ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func respondUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
}
