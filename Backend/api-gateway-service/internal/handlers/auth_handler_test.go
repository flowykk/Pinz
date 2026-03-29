package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/api-gateway-service/internal/mocks"
	"pinz/backend/api-gateway-service/pkg/proto"
)

func TestAuthHandler_DevLogin_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	authSvc := mocks.NewMockAuthServiceInterface(ctrl)
	authSvc.EXPECT().
		DevLogin(gomock.Any(), "user@example.com").
		Return(&proto.DevLoginResponse{
			AccessToken:  "access-123",
			RefreshToken: "refresh-456",
		}, nil)

	h := NewAuthHandler(authSvc)
	body := `{"email":"user@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/dev-login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.DevLogin(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	var dst struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &dst))
	require.Equal(t, "access-123", dst.AccessToken)
	require.Equal(t, "refresh-456", dst.RefreshToken)
}

func TestAuthHandler_DevLogin_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	authSvc := mocks.NewMockAuthServiceInterface(ctrl)
	authSvc.EXPECT().
		DevLogin(gomock.Any(), "nobody@example.com").
		Return(nil, status.Error(codes.NotFound, "user not found"))

	h := NewAuthHandler(authSvc)
	body := `{"email":"nobody@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/dev-login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.DevLogin(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
	require.Contains(t, rr.Body.String(), "user not found")
}
