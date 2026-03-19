package services

import (
	"context"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/auth-service/internal/db"
	"pinz/backend/auth-service/internal/models"
	"pinz/backend/auth-service/internal/repositories"
	"pinz/backend/auth-service/internal/testinfra"
	pb "pinz/backend/auth-service/pkg/proto"
)

func TestAuthService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	testinfra.WithAuthPostgres(t)
	t.Setenv("JWT_SECRET_KEY", "test-secret")

	sqlDB, err := db.InitDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	userRepo := repositories.NewUserRepository(sqlDB)
	u := &models.User{
		ID:       uuid.New().String(),
		Email:    "user@example.com",
		Username: "user",
	}
	require.NoError(t, userRepo.CreateUser(u))

	svc := NewAuthService(userRepo, nil, nil, validator.New(), nil)
	ctx := context.Background()

	env := &struct {
		accessToken  string
		refreshToken string
	}{}

	cases := map[string]func(t *testing.T){
		"DevLogin_user_not_found": func(t *testing.T) {
			_, err := svc.DevLogin(ctx, &pb.DevLoginRequest{Email: "nonexistent@example.com"})
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, codes.NotFound, st.Code())
		},
		"RefreshToken_unknown_token_unauthenticated": func(t *testing.T) {
			_, err := svc.RefreshToken(ctx, &pb.RefreshTokenRequest{RefreshToken: "not-a-valid-refresh-token-" + uuid.New().String()})
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, codes.Unauthenticated, st.Code())
		},
		"DevLogin_returns_tokens": func(t *testing.T) {
			resp, err := svc.DevLogin(ctx, &pb.DevLoginRequest{Email: u.Email})
			require.NoError(t, err)
			require.NotEmpty(t, resp.GetAccessToken())
			require.NotEmpty(t, resp.GetRefreshToken())
			env.accessToken = resp.GetAccessToken()
			env.refreshToken = resp.GetRefreshToken()
		},
		"DevLogin_sets_created_at": func(t *testing.T) {
			require.WithinDuration(t, time.Now(), u.CreatedAt, 5*time.Second)
		},
		"RefreshToken_valid_returns_new_access": func(t *testing.T) {
			resp, err := svc.RefreshToken(ctx, &pb.RefreshTokenRequest{RefreshToken: env.refreshToken})
			require.NoError(t, err)
			require.NotEmpty(t, resp.GetAccessToken())
		},
		"Logout_revokes_refresh": func(t *testing.T) {
			resp, err := svc.Logout(ctx, &pb.LogoutRequest{RefreshToken: env.refreshToken})
			require.NoError(t, err)
			require.True(t, resp.GetSuccess())
		},
		"RefreshToken_after_logout_fails": func(t *testing.T) {
			_, err := svc.RefreshToken(ctx, &pb.RefreshTokenRequest{RefreshToken: env.refreshToken})
			require.Error(t, err)
		},
		"Logout_double_returns_success": func(t *testing.T) {
			resp, err := svc.Logout(ctx, &pb.LogoutRequest{RefreshToken: env.refreshToken})
			require.NoError(t, err)
			require.True(t, resp.GetSuccess())
		},
	}

	order := []string{
		"DevLogin_user_not_found", "RefreshToken_unknown_token_unauthenticated", "DevLogin_returns_tokens", "DevLogin_sets_created_at",
		"RefreshToken_valid_returns_new_access", "Logout_revokes_refresh",
		"RefreshToken_after_logout_fails", "Logout_double_returns_success",
	}
	for _, name := range order {
		t.Run(name, cases[name])
	}
}
