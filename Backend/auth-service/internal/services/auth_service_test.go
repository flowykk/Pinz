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
	svc := NewAuthService(nil, nil, nil, nil, validator.New(), nil, nil, WithDevLogin(true))
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

	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil, WithDevLogin(true))
	ctx := context.Background()
	_, err := svc.DevLogin(ctx, &pb.DevLoginRequest{Email: "nobody@example.com"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, st.Code())
}

func TestDevLogin_DisabledReturnsUnimplemented(t *testing.T) {
	svc := NewAuthService(nil, nil, nil, nil, validator.New(), nil, nil)
	_, err := svc.DevLogin(context.Background(), &pb.DevLoginRequest{Email: "any@example.com"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Unimplemented, st.Code())
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

func TestDevLogin_HappyPath_IssuesTokens(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", "test-secret")
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	user := &models.User{ID: "u1", Email: "u@example.com", Username: "user"}
	userRepo.EXPECT().GetUserByEmail("u@example.com").Return(user, nil)
	userRepo.EXPECT().AddSession("u1", gomock.Any(), gomock.Any()).Return(nil)

	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil, WithDevLogin(true))
	resp, err := svc.DevLogin(context.Background(), &pb.DevLoginRequest{Email: user.Email})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetAccessToken())
	require.NotEmpty(t, resp.GetRefreshToken())
}

func TestDevLogin_NoJwtSecret_Internal(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", "")
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	userRepo.EXPECT().GetUserByEmail("u@example.com").Return(&models.User{ID: "u1", Email: "u@example.com"}, nil)
	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil, WithDevLogin(true))
	_, err := svc.DevLogin(context.Background(), &pb.DevLoginRequest{Email: "u@example.com"})
	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestDevLogin_AddSessionError_Internal(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", "test-secret")
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	userRepo.EXPECT().GetUserByEmail("u@example.com").Return(&models.User{ID: "u1", Email: "u@example.com"}, nil)
	userRepo.EXPECT().AddSession("u1", gomock.Any(), gomock.Any()).Return(errors.New("db down"))
	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil, WithDevLogin(true))
	_, err := svc.DevLogin(context.Background(), &pb.DevLoginRequest{Email: "u@example.com"})
	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestGetProfile_EmptyUserID(t *testing.T) {
	svc := authServiceForValidation(t)
	_, err := svc.GetProfile(context.Background(), &pb.GetProfileRequest{UserId: ""})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetProfile_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	userRepo.EXPECT().GetUserByID("missing").Return(nil, sql.ErrNoRows)
	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil)
	_, err := svc.GetProfile(context.Background(), &pb.GetProfileRequest{UserId: "missing"})
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetProfile_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	user := &models.User{ID: "u1", Email: "e@example.com", Username: "user", CreatedAt: time.Unix(1234, 0)}
	userRepo.EXPECT().GetUserByID("u1").Return(user, nil)
	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil)

	resp, err := svc.GetProfile(context.Background(), &pb.GetProfileRequest{UserId: "u1"})
	require.NoError(t, err)
	require.Equal(t, "u1", resp.GetUser().GetId())
	require.Equal(t, "e@example.com", resp.GetUser().GetEmail())
	require.Equal(t, int64(1234), resp.GetUser().GetCreatedAtUnix())
}

func TestUpdateProfile_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	updated := &models.User{ID: "u1", Email: "e@example.com", Username: "newname"}
	userRepo.EXPECT().UpdateUsername("u1", "newname").Return(updated, nil)
	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil)

	resp, err := svc.UpdateProfile(context.Background(), &pb.UpdateProfileRequest{UserId: "u1", Username: "newname"})
	require.NoError(t, err)
	require.Equal(t, "newname", resp.GetUser().GetUsername())
}

func TestUpdateProfile_RepoError_Internal(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	userRepo.EXPECT().UpdateUsername("u1", "newname").Return(nil, errors.New("db down"))
	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil)
	_, err := svc.UpdateProfile(context.Background(), &pb.UpdateProfileRequest{UserId: "u1", Username: "newname"})
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestChangeEmail_ValidationErrors(t *testing.T) {
	svc := authServiceForValidation(t)
	cases := map[string]*pb.ChangeEmailRequest{
		"empty_user_id":  {UserId: "", NewEmail: "x@example.com"},
		"empty_new_email": {UserId: "u1", NewEmail: "  "},
		"invalid_email":  {UserId: "u1", NewEmail: "not-an-email"},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.ChangeEmail(context.Background(), req)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestChangeEmail_AlreadyTakenByOtherUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	userRepo.EXPECT().GetUserByEmail("new@example.com").Return(&models.User{ID: "other"}, nil)
	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil)
	_, err := svc.ChangeEmail(context.Background(), &pb.ChangeEmailRequest{UserId: "u1", NewEmail: "new@example.com"})
	require.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestChangeEmail_HappyPath_StoresInRedisAndEnqueues(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	redisRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
	userRepo.EXPECT().GetUserByEmail("new@example.com").Return(nil, sql.ErrNoRows)
	redisRepo.EXPECT().HSet(gomock.Any(), "email_change:u1", "email", "new@example.com", "code", gomock.Any()).Return(nil)
	redisRepo.EXPECT().Expire(gomock.Any(), "email_change:u1", 15*time.Minute).Return(nil)
	redisRepo.EXPECT().XAdd(gomock.Any(), "pinz:auth:email:tasks", gomock.Any()).Return(nil)
	svc := NewAuthService(userRepo, nil, redisRepo, nil, validator.New(), nil, nil)
	resp, err := svc.ChangeEmail(context.Background(), &pb.ChangeEmailRequest{UserId: "u1", NewEmail: "new@example.com"})
	require.NoError(t, err)
	require.True(t, resp.GetSuccess())
}

func TestChangeEmail_HSetError_Internal(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	redisRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
	userRepo.EXPECT().GetUserByEmail("new@example.com").Return(nil, sql.ErrNoRows)
	redisRepo.EXPECT().HSet(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("redis down"))
	svc := NewAuthService(userRepo, nil, redisRepo, nil, validator.New(), nil, nil)
	_, err := svc.ChangeEmail(context.Background(), &pb.ChangeEmailRequest{UserId: "u1", NewEmail: "new@example.com"})
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestConfirmEmailChange_ValidationErrors(t *testing.T) {
	svc := authServiceForValidation(t)
	cases := map[string]*pb.ConfirmEmailChangeRequest{
		"empty_user_id": {UserId: "", VerificationCode: "123456"},
		"empty_code":    {UserId: "u1", VerificationCode: ""},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.ConfirmEmailChange(context.Background(), req)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestConfirmEmailChange_NoPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	redisRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
	redisRepo.EXPECT().HGetAll(gomock.Any(), "email_change:u1").Return(map[string]string{}, nil)
	svc := NewAuthService(nil, nil, redisRepo, nil, validator.New(), nil, nil)
	_, err := svc.ConfirmEmailChange(context.Background(), &pb.ConfirmEmailChangeRequest{UserId: "u1", VerificationCode: "123456"})
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestConfirmEmailChange_WrongCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	redisRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
	redisRepo.EXPECT().HGetAll(gomock.Any(), "email_change:u1").Return(map[string]string{
		"email": "new@example.com", "code": "999999",
	}, nil)
	svc := NewAuthService(nil, nil, redisRepo, nil, validator.New(), nil, nil)
	_, err := svc.ConfirmEmailChange(context.Background(), &pb.ConfirmEmailChangeRequest{UserId: "u1", VerificationCode: "111111"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestConfirmEmailChange_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	redisRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
	redisRepo.EXPECT().HGetAll(gomock.Any(), "email_change:u1").Return(map[string]string{
		"email": "new@example.com", "code": "123456",
	}, nil)
	userRepo.EXPECT().UpdateEmail("u1", "new@example.com").Return(&models.User{ID: "u1", Email: "new@example.com"}, nil)
	redisRepo.EXPECT().Del(gomock.Any(), "email_change:u1").Return(nil)
	svc := NewAuthService(userRepo, nil, redisRepo, nil, validator.New(), nil, nil)
	resp, err := svc.ConfirmEmailChange(context.Background(), &pb.ConfirmEmailChangeRequest{UserId: "u1", VerificationCode: "123456"})
	require.NoError(t, err)
	require.Equal(t, "new@example.com", resp.GetUser().GetEmail())
}

func TestConfirmEmailChange_UniqueViolationMaps(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	redisRepo := mocks.NewMockRedisRepositoryInterface(ctrl)
	redisRepo.EXPECT().HGetAll(gomock.Any(), "email_change:u1").Return(map[string]string{
		"email": "new@example.com", "code": "123456",
	}, nil)
	userRepo.EXPECT().UpdateEmail("u1", "new@example.com").Return(nil, &pgconn.PgError{Code: "23505"})
	svc := NewAuthService(userRepo, nil, redisRepo, nil, validator.New(), nil, nil)
	_, err := svc.ConfirmEmailChange(context.Background(), &pb.ConfirmEmailChangeRequest{UserId: "u1", VerificationCode: "123456"})
	require.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestDeleteAccount_EmptyUserID(t *testing.T) {
	svc := authServiceForValidation(t)
	_, err := svc.DeleteAccount(context.Background(), &pb.DeleteAccountRequest{UserId: ""})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestDeleteAccount_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	userRepo.EXPECT().DeleteUser("u1").Return(nil)
	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil)
	resp, err := svc.DeleteAccount(context.Background(), &pb.DeleteAccountRequest{UserId: "u1"})
	require.NoError(t, err)
	require.True(t, resp.GetSuccess())
}

func TestDeleteAccount_RepoError_Internal(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	userRepo.EXPECT().DeleteUser("u1").Return(errors.New("fk violation"))
	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil)
	_, err := svc.DeleteAccount(context.Background(), &pb.DeleteAccountRequest{UserId: "u1"})
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestUserToProto_PresignsAvatar(t *testing.T) {
	ctrl := gomock.NewController(t)
	s3Mock := mocks.NewMockS3Uploader(ctrl)
	s3Mock.EXPECT().ReadURL(gomock.Any(), "avatars/u1/a.jpg").Return("https://signed/a.jpg", nil)
	svc := NewAuthService(nil, nil, nil, nil, validator.New(), nil, s3Mock)
	out := svc.userToProto(context.Background(), &models.User{ID: "u1", AvatarURL: "avatars/u1/a.jpg", CreatedAt: time.Unix(1, 0)})
	require.Equal(t, "https://signed/a.jpg", out.AvatarUrl)
}

func TestUserToProto_PresignError_LeavesAvatarEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	s3Mock := mocks.NewMockS3Uploader(ctrl)
	s3Mock.EXPECT().ReadURL(gomock.Any(), "avatars/u1/a.jpg").Return("", errors.New("s3 down"))
	svc := NewAuthService(nil, nil, nil, nil, validator.New(), nil, s3Mock)
	out := svc.userToProto(context.Background(), &models.User{ID: "u1", AvatarURL: "avatars/u1/a.jpg"})
	require.Empty(t, out.AvatarUrl)
}

func TestUserToProto_NoAvatar(t *testing.T) {
	svc := authServiceForValidation(t)
	out := svc.userToProto(context.Background(), &models.User{ID: "u1", Email: "e@example.com", Username: "user", CreatedAt: time.Unix(42, 0)})
	require.Empty(t, out.AvatarUrl)
	require.Equal(t, "u1", out.Id)
	require.Equal(t, "e@example.com", out.Email)
	require.Equal(t, int64(42), out.CreatedAtUnix)
}

func TestGetUsersProfiles_EmptyReturnsEmpty(t *testing.T) {
	svc := authServiceForValidation(t)
	resp, err := svc.GetUsersProfiles(context.Background(), &pb.GetUsersProfilesRequest{})
	require.NoError(t, err)
	require.Empty(t, resp.GetProfiles())
}

func TestGetUsersProfiles_FillsKnownAndPlaceholdersUnknown(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	userRepo.EXPECT().GetUsersByIDs([]string{"u1", "missing"}).Return([]*models.User{
		{ID: "u1", Username: "alice", CreatedAt: time.Unix(1, 0)},
	}, nil)
	svc := NewAuthService(userRepo, nil, nil, nil, validator.New(), nil, nil)
	resp, err := svc.GetUsersProfiles(context.Background(), &pb.GetUsersProfilesRequest{UserIds: []string{"u1", "missing"}})
	require.NoError(t, err)
	require.Len(t, resp.GetProfiles(), 2)
	require.Equal(t, "alice", resp.GetProfiles()[0].GetUsername())
	require.Equal(t, "missing", resp.GetProfiles()[1].GetUserId())
	require.Empty(t, resp.GetProfiles()[1].GetUsername())
}
