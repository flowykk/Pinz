package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/auth-service/internal/mocks"
	"pinz/backend/auth-service/internal/models"
	pb "pinz/backend/auth-service/pkg/proto"
)

func TestIsUniqueViolation(t *testing.T) {
	cases := map[string]struct {
		err error
		want bool
	}{
		"nil": {nil, false},
		"plain_error": {errors.New("plain"), false},
		"pg_error_code_23505": {&pgconn.PgError{Code: "23505"}, true},
		"pg_error_code_23503": {&pgconn.PgError{Code: "23503"}, false},
		"pg_error_code_other": {&pgconn.PgError{Code: "23001"}, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := isUniqueViolation(tc.err)
			require.Equal(t, tc.want, got)
		})
	}
}

// authServiceForValidation returns AuthService with nil repos; only validation paths are tested.
func authServiceForValidation(t *testing.T) *AuthService {
	t.Helper()
	return NewAuthService(nil, nil, nil, nil, validator.New(), nil, nil)
}

func TestDevLogin_ValidationErrors(t *testing.T) {
	svc := authServiceForValidation(t)
	ctx := context.Background()
	cases := map[string]struct {
		req *pb.DevLoginRequest
		code codes.Code
	}{
		"empty_email": {req: &pb.DevLoginRequest{Email: ""}, code: codes.InvalidArgument},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.DevLogin(ctx, tc.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tc.code, st.Code())
		})
	}
}

func TestSubmitEmail_ValidationErrors(t *testing.T) {
	svc := authServiceForValidation(t)
	ctx := context.Background()
	cases := map[string]struct {
		req *pb.SubmitEmailRequest
		code codes.Code
	}{
		"empty_email": {req: &pb.SubmitEmailRequest{Email: ""}, code: codes.InvalidArgument},
		"invalid_email": {req: &pb.SubmitEmailRequest{Email: "not-an-email"}, code: codes.InvalidArgument},
		"whitespace_only": {req: &pb.SubmitEmailRequest{Email: " "}, code: codes.InvalidArgument},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.SubmitEmail(ctx, tc.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tc.code, st.Code())
		})
	}
}

func TestRefreshToken_ValidationErrors(t *testing.T) {
	svc := authServiceForValidation(t)
	ctx := context.Background()
	cases := map[string]struct {
		req *pb.RefreshTokenRequest
		code codes.Code
	}{
		"empty_token": {req: &pb.RefreshTokenRequest{RefreshToken: ""}, code: codes.InvalidArgument},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.RefreshToken(ctx, tc.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tc.code, st.Code())
		})
	}
}

func TestLogout_ValidationErrors(t *testing.T) {
	svc := authServiceForValidation(t)
	ctx := context.Background()
	cases := map[string]struct {
		req *pb.LogoutRequest
		code codes.Code
	}{
		"empty_token": {req: &pb.LogoutRequest{RefreshToken: ""}, code: codes.InvalidArgument},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.Logout(ctx, tc.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tc.code, st.Code())
		})
	}
}

func TestVerifyEmailCode_ValidationErrors(t *testing.T) {
	svc := authServiceForValidation(t)
	ctx := context.Background()
	cases := map[string]struct {
		req *pb.VerifyEmailCodeRequest
		code codes.Code
	}{
		"empty_registration_id": {
			req: &pb.VerifyEmailCodeRequest{RegistrationId: "", VerificationCode: "123456"},
			code: codes.InvalidArgument,
		},
		"empty_code": {
			req: &pb.VerifyEmailCodeRequest{RegistrationId: "reg-1", VerificationCode: ""},
			code: codes.InvalidArgument,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.VerifyEmailCode(ctx, tc.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tc.code, st.Code())
		})
	}
}

func TestPasskeyRegisterBegin_ValidationErrors(t *testing.T) {
	svc := authServiceForValidation(t)
	ctx := context.Background()
	cases := map[string]struct {
		req *pb.PasskeyRegisterBeginRequest
		code codes.Code
	}{
		"empty_registration_id": {
			req: &pb.PasskeyRegisterBeginRequest{RegistrationId: "", Username: "user"},
			code: codes.InvalidArgument,
		},
		"empty_username": {
			req: &pb.PasskeyRegisterBeginRequest{RegistrationId: "reg-1", Username: ""},
			code: codes.InvalidArgument,
		},
		"username_too_short": {
			req: &pb.PasskeyRegisterBeginRequest{RegistrationId: "reg-1", Username: "abc"},
			code: codes.InvalidArgument,
		},
		"username_too_long": {
			req: &pb.PasskeyRegisterBeginRequest{RegistrationId: "reg-1", Username: string(make([]byte, 21))},
			code: codes.InvalidArgument,
		},
		"username_invalid_chars": {
			req: &pb.PasskeyRegisterBeginRequest{RegistrationId: "reg-1", Username: "user@name"},
			code: codes.InvalidArgument,
		},
		"username_spaces": {
			req: &pb.PasskeyRegisterBeginRequest{RegistrationId: "reg-1", Username: "user name"},
			code: codes.InvalidArgument,
		},
		"username_cyrillic": {
			req: &pb.PasskeyRegisterBeginRequest{RegistrationId: "reg-1", Username: "Пользователь"},
			code: codes.InvalidArgument,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.PasskeyRegisterBegin(ctx, tc.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tc.code, st.Code())
		})
	}
}

func TestPasskeyRegisterFinish_ValidationErrors(t *testing.T) {
	svc := authServiceForValidation(t)
	ctx := context.Background()
	cases := map[string]struct {
		req *pb.PasskeyRegisterFinishRequest
		code codes.Code
	}{
		"empty_registration_id": {
			req: &pb.PasskeyRegisterFinishRequest{RegistrationId: "", CredentialJson: []byte("{}")},
			code: codes.InvalidArgument,
		},
		"empty_credential_json": {
			req: &pb.PasskeyRegisterFinishRequest{RegistrationId: "reg-1", CredentialJson: nil},
			code: codes.InvalidArgument,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.PasskeyRegisterFinish(ctx, tc.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tc.code, st.Code())
		})
	}
}

func TestPasskeyLoginBegin_ValidationErrors(t *testing.T) {
	svc := authServiceForValidation(t)
	ctx := context.Background()
	cases := map[string]struct {
		req *pb.PasskeyLoginBeginRequest
		code codes.Code
	}{
		"empty_email": {req: &pb.PasskeyLoginBeginRequest{Email: ""}, code: codes.InvalidArgument},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.PasskeyLoginBegin(ctx, tc.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tc.code, st.Code())
		})
	}
}

func TestPasskeyLoginFinish_ValidationErrors(t *testing.T) {
	svc := authServiceForValidation(t)
	ctx := context.Background()
	cases := map[string]struct {
		req *pb.PasskeyLoginFinishRequest
		code codes.Code
	}{
		"empty_email": {
			req: &pb.PasskeyLoginFinishRequest{Email: "", CredentialJson: []byte("{}")},
			code: codes.InvalidArgument,
		},
		"empty_credential_json": {
			req: &pb.PasskeyLoginFinishRequest{Email: "user@example.com", CredentialJson: nil},
			code: codes.InvalidArgument,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.PasskeyLoginFinish(ctx, tc.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tc.code, st.Code())
		})
	}
}

func TestSubmitEmail_user_exists(t *testing.T) {
	ctrl := gomock.NewController(t)
	existingUser := &models.User{ID: "u1", Email: "existing@example.com", Username: "user"}
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	userRepo.EXPECT().GetUserByEmail(existingUser.Email).Return(existingUser, nil)
	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil)
	ctx := context.Background()

	resp, err := svc.SubmitEmail(ctx, &pb.SubmitEmailRequest{Email: existingUser.Email})
	require.NoError(t, err)
	require.True(t, resp.GetIsRegistered())
	require.Empty(t, resp.GetRegistrationKey())
}

func TestRefreshToken_expired(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", "test-secret")
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	userRepo.EXPECT().GetRefreshToken("some-token").Return(&models.RefreshToken{
		ID: "rt1", UserID: "u1", Token: "some-token",
		ExpiresAt: time.Now().Add(-time.Hour),
	}, nil)
	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil)
	ctx := context.Background()

	_, err := svc.RefreshToken(ctx, &pb.RefreshTokenRequest{RefreshToken: "some-token"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Unauthenticated, st.Code())
}

func TestRefreshToken_invalid(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	userRepo.EXPECT().GetRefreshToken("invalid-token").Return(nil, sql.ErrNoRows)
	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil)
	ctx := context.Background()

	_, err := svc.RefreshToken(ctx, &pb.RefreshTokenRequest{RefreshToken: "invalid-token"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Unauthenticated, st.Code())
}

func TestLogout_token_not_found(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	userRepo.EXPECT().GetRefreshToken("missing-token").Return(nil, sql.ErrNoRows)
	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil)
	ctx := context.Background()

	resp, err := svc.Logout(ctx, &pb.LogoutRequest{RefreshToken: "missing-token"})
	require.NoError(t, err)
	require.True(t, resp.GetSuccess())
}

// --- Unit tests with mocks (plan: auth-unit) ---

func TestSubmitEmail_new_user_returns_registration_key(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	redisRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
	userRepo.EXPECT().GetUserByEmail("new@example.com").Return(nil, sql.ErrNoRows)
	redisRepo.EXPECT().HSet(gomock.Any(), gomock.Any(), "email", "new@example.com", "code", gomock.Any()).Return(nil)
	redisRepo.EXPECT().Expire(gomock.Any(), gomock.Any(), 15*time.Minute).Return(nil)
	redisRepo.EXPECT().XAdd(gomock.Any(), "pinz:auth:email:tasks", gomock.Any()).Return(nil)

	svc := NewAuthService(userRepo, nil, redisRepo, nil, validator.New(), nil, nil)
	ctx := context.Background()
	resp, err := svc.SubmitEmail(ctx, &pb.SubmitEmailRequest{Email: "new@example.com"})
	require.NoError(t, err)
	require.False(t, resp.GetIsRegistered())
	require.NotEmpty(t, resp.GetRegistrationKey())
}

func TestSubmitEmail_email_enqueue_failure_still_succeeds(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	redisRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
	userRepo.EXPECT().GetUserByEmail("new@example.com").Return(nil, sql.ErrNoRows)
	redisRepo.EXPECT().HSet(gomock.Any(), gomock.Any(), "email", "new@example.com", "code", gomock.Any()).Return(nil)
	redisRepo.EXPECT().Expire(gomock.Any(), gomock.Any(), 15*time.Minute).Return(nil)
	redisRepo.EXPECT().XAdd(gomock.Any(), "pinz:auth:email:tasks", gomock.Any()).Return(errors.New("redis connection refused"))

	svc := NewAuthService(userRepo, nil, redisRepo, nil, validator.New(), nil, nil)
	ctx := context.Background()
	resp, err := svc.SubmitEmail(ctx, &pb.SubmitEmailRequest{Email: "new@example.com"})
	require.NoError(t, err)
	require.False(t, resp.GetIsRegistered())
	require.NotEmpty(t, resp.GetRegistrationKey())
}

func TestSubmitEmail_db_error_returns_internal(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	userRepo.EXPECT().GetUserByEmail("x@example.com").Return(nil, errors.New("db connection refused"))

	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil)
	ctx := context.Background()
	_, err := svc.SubmitEmail(ctx, &pb.SubmitEmailRequest{Email: "x@example.com"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Internal, st.Code())
}

func TestRefreshToken_success_returns_new_access_token(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", "test-secret")
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	userRepo.EXPECT().GetRefreshToken("valid-token").Return(&models.RefreshToken{
		ID: "r1", UserID: "u1", Token: "valid-token",
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil)
	userRepo.EXPECT().UpdateRefreshTokenExpiresAt("r1", gomock.Any()).Return(nil)
	userRepo.EXPECT().GetUserByID("u1").Return(&models.User{ID: "u1", Email: "u@example.com", Username: "user"}, nil)

	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil)
	ctx := context.Background()
	resp, err := svc.RefreshToken(ctx, &pb.RefreshTokenRequest{RefreshToken: "valid-token"})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetAccessToken())
}

func TestDevLogin_user_not_found(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	userRepo.EXPECT().GetUserByEmail("nobody@example.com").Return(nil, sql.ErrNoRows)

	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil)
	ctx := context.Background()
	_, err := svc.DevLogin(ctx, &pb.DevLoginRequest{Email: "nobody@example.com"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, st.Code())
}

func TestUpdateProfile_ValidationErrors(t *testing.T) {
	svc := authServiceForValidation(t)
	ctx := context.Background()
	cases := map[string]struct {
		req *pb.UpdateProfileRequest
		code codes.Code
	}{
		"empty_user_id": {
			req: &pb.UpdateProfileRequest{UserId: "", Username: "valid"},
			code: codes.InvalidArgument,
		},
		"username_too_short": {
			req: &pb.UpdateProfileRequest{UserId: "u1", Username: "abc"},
			code: codes.InvalidArgument,
		},
		"username_invalid_chars": {
			req: &pb.UpdateProfileRequest{UserId: "u1", Username: "user@name"},
			code: codes.InvalidArgument,
		},
		"username_cyrillic": {
			req: &pb.UpdateProfileRequest{UserId: "u1", Username: "Пользователь"},
			code: codes.InvalidArgument,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.UpdateProfile(ctx, tc.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tc.code, st.Code())
		})
	}
}

func TestDeleteAvatar_EmptyUserID(t *testing.T) {
	svc := authServiceForValidation(t)
	ctx := context.Background()
	_, err := svc.DeleteAvatar(ctx, &pb.DeleteAvatarRequest{UserId: ""})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
}

func TestDeleteAvatar_WithAvatar(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	s3Mock := mocks.NewMockS3Uploader(ctrl)

	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, s3Mock)
	ctx := context.Background()

	userWithAvatar := &models.User{
		ID: "user-1",
		Email: "test@example.com",
		Username: "test",
		AvatarURL: "avatars/user-1/avatar.jpg",
		CreatedAt: time.Now(),
	}
	userWithoutAvatar := &models.User{
		ID: "user-1",
		Email: "test@example.com",
		Username: "test",
		AvatarURL: "",
		CreatedAt: time.Now(),
	}

	userRepo.EXPECT().GetUserByID("user-1").Return(userWithAvatar, nil)
	s3Mock.EXPECT().DeleteObject(gomock.Any(), "avatars/user-1/avatar.jpg").Return(nil)
	userRepo.EXPECT().UpdateAvatarURL("user-1", "").Return(userWithoutAvatar, nil)

	resp, err := svc.DeleteAvatar(ctx, &pb.DeleteAvatarRequest{UserId: "user-1"})
	require.NoError(t, err)
	require.Empty(t, resp.GetUser().GetAvatarUrl())
}

func TestDeleteAvatar_WithoutAvatar(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)

	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil)
	ctx := context.Background()

	user := &models.User{
		ID: "user-1",
		Email: "test@example.com",
		Username: "test",
		AvatarURL: "",
		CreatedAt: time.Now(),
	}

	userRepo.EXPECT().GetUserByID("user-1").Return(user, nil)
	userRepo.EXPECT().UpdateAvatarURL("user-1", "").Return(user, nil)

	resp, err := svc.DeleteAvatar(ctx, &pb.DeleteAvatarRequest{UserId: "user-1"})
	require.NoError(t, err)
	require.Empty(t, resp.GetUser().GetAvatarUrl())
}

func TestDeleteAvatar_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)

	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil)
	ctx := context.Background()

	userRepo.EXPECT().GetUserByID("user-1").Return(nil, sql.ErrNoRows)

	_, err := svc.DeleteAvatar(ctx, &pb.DeleteAvatarRequest{UserId: "user-1"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, st.Code())
}

func TestRequestAvatarUpload_InvalidFormat(t *testing.T) {
	svc := authServiceForValidation(t)
	ctx := context.Background()
	_, err := svc.RequestAvatarUpload(ctx, &pb.RequestAvatarUploadRequest{
		UserId: "u1",
		Filename: "avatar.gif",
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
}
