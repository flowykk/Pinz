package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestRequireJWT(t *testing.T) {
	handlerNeverCalled := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	cases := map[string]func(t *testing.T){
		"missing_authorization_header_returns_401": func(t *testing.T) {
			h := RequireJWT(handlerNeverCalled)
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			h.ServeHTTP(rr, req)
			require.Equal(t, http.StatusUnauthorized, rr.Code)
			require.Contains(t, rr.Body.String(), "unauthorized")
		},
		"empty_bearer_returns_401": func(t *testing.T) {
			h := RequireJWT(handlerNeverCalled)
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			req.Header.Set("Authorization", "Bearer ")
			h.ServeHTTP(rr, req)
			require.Equal(t, http.StatusUnauthorized, rr.Code)
		},
		"missing_jwt_secret_returns_500": func(t *testing.T) {
			t.Setenv("JWT_SECRET_KEY", "")
			h := RequireJWT(handlerNeverCalled)
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			req.Header.Set("Authorization", "Bearer abc.def.ghi")
			h.ServeHTTP(rr, req)
			require.Equal(t, http.StatusInternalServerError, rr.Code)
			require.Contains(t, rr.Body.String(), "server misconfiguration")
		},
		"invalid_token_returns_401": func(t *testing.T) {
			t.Setenv("JWT_SECRET_KEY", "secret")
			h := RequireJWT(handlerNeverCalled)
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			req.Header.Set("Authorization", "Bearer not-a-jwt")
			h.ServeHTTP(rr, req)
			require.Equal(t, http.StatusUnauthorized, rr.Code)
		},
		"token_without_user_id_claim_returns_401": func(t *testing.T) {
			secret := "secret"
			t.Setenv("JWT_SECRET_KEY", secret)
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"username": "u"})
			s, err := token.SignedString([]byte(secret))
			require.NoError(t, err)
			h := RequireJWT(handlerNeverCalled)
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			req.Header.Set("Authorization", "Bearer "+s)
			h.ServeHTTP(rr, req)
			require.Equal(t, http.StatusUnauthorized, rr.Code)
		},
		"valid_token_calls_next_and_sets_user_id_in_context": func(t *testing.T) {
			secret := "secret"
			userID := "user-123"
			t.Setenv("JWT_SECRET_KEY", secret)
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"user_id":  userID,
				"username": "u",
			})
			s, err := token.SignedString([]byte(secret))
			require.NoError(t, err)
			called := false
			h := RequireJWT(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				require.Equal(t, userID, UserIDFromContext(r.Context()))
				w.WriteHeader(http.StatusOK)
			}))
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			req.Header.Set("Authorization", "Bearer "+s)
			h.ServeHTTP(rr, req)
			require.True(t, called)
			require.Equal(t, http.StatusOK, rr.Code)
		},
	}
	for name, fn := range cases {
		t.Run(name, fn)
	}
}
