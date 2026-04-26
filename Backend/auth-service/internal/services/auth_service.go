package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/auth-service/internal/models"
	"pinz/backend/auth-service/internal/utils"
	pb "pinz/backend/auth-service/pkg/proto"
)

// pendingUser implements webauthn.User for a not-yet-persisted registration.
type pendingUser struct {
	id []byte
	name string
	displayName string
	credentials []webauthn.Credential
}

func (u *pendingUser) WebAuthnID() []byte { return u.id }
func (u *pendingUser) WebAuthnName() string { return u.name }
func (u *pendingUser) WebAuthnDisplayName() string { return u.displayName }
func (u *pendingUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// existingUser implements webauthn.User for a fully-persisted user.
type existingUser struct {
	id []byte
	name string
	displayName string
	credentials []webauthn.Credential
}

func (u *existingUser) WebAuthnID() []byte { return u.id }
func (u *existingUser) WebAuthnName() string { return u.name }
func (u *existingUser) WebAuthnDisplayName() string { return u.displayName }
func (u *existingUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// regSession is stored in Redis between PasskeyRegisterBegin and PasskeyRegisterFinish.
type regSession struct {
	PendingUserID string `json:"pending_user_id"`
	Username string `json:"username"`
	SessionData webauthn.SessionData `json:"session_data"`
}

// loginSession is stored in Redis between PasskeyLoginBegin and PasskeyLoginFinish.
type loginSession struct {
	UserID string `json:"user_id"`
	SessionData webauthn.SessionData `json:"session_data"`
}

type S3Uploader interface {
	PresignedUploadURL(ctx context.Context, s3Key, contentType string) (string, error)
	ReadURL(ctx context.Context, s3Key string) (string, error)
	DeleteObject(ctx context.Context, s3Key string) error
}

type AuthService struct {
	pb.UnimplementedAuthServiceServer
	userRepo UserRepositoryInterface
	credRepo CredentialRepositoryInterface
	redisRepo RedisRepositoryInterface
	desiredPlaceRepo DesiredPlaceRepositoryInterface
	validator *validator.Validate
	wa *webauthn.WebAuthn
	s3 S3Uploader

	tracer trace.Tracer
	loginCounter metric.Int64Counter
	registrationCounter metric.Int64Counter
	tokenRefreshCounter metric.Int64Counter
}

func NewAuthService(
	userRepo UserRepositoryInterface,
	credRepo CredentialRepositoryInterface,
	redisRepo RedisRepositoryInterface,
	desiredPlaceRepo DesiredPlaceRepositoryInterface,
	validator *validator.Validate,
	wa *webauthn.WebAuthn,
	s3 S3Uploader,
) *AuthService {
	tracer := otel.Tracer("auth-service")
	meter := otel.Meter("auth-service")

	loginCounter, _ := meter.Int64Counter("auth.logins.total",
		metric.WithDescription("Total login attempts by method and status"),
	)
	registrationCounter, _ := meter.Int64Counter("auth.registrations.total",
		metric.WithDescription("Total registration attempts"),
	)
	tokenRefreshCounter, _ := meter.Int64Counter("auth.token_refresh.total",
		metric.WithDescription("Total token refresh operations"),
	)

	return &AuthService{
		userRepo: userRepo,
		credRepo: credRepo,
		redisRepo: redisRepo,
		desiredPlaceRepo: desiredPlaceRepo,
		validator: validator,
		wa: wa,
		s3: s3,
		tracer: tracer,
		loginCounter: loginCounter,
		registrationCounter: registrationCounter,
		tokenRefreshCounter: tokenRefreshCounter,
	}
}

func (s *AuthService) SubmitEmail(ctx context.Context, req *pb.SubmitEmailRequest) (*pb.SubmitEmailResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.SubmitEmail")
	defer span.End()

	email := strings.TrimSpace(req.GetEmail())
	if email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	if err := s.validator.Var(email, "required,email"); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email: %v", err)
	}

	_, err := s.userRepo.GetUserByEmail(email)
	if err == nil {
		span.SetAttributes(attribute.Bool("auth.user_exists", true))
		return &pb.SubmitEmailResponse{IsRegistered: true, RegistrationKey: ""}, nil
	}
	if err != sql.ErrNoRows {
		slog.ErrorContext(ctx, "SubmitEmail: get user by email", "error", err)
		return nil, status.Error(codes.Internal, "failed to check user existence")
	}

	span.SetAttributes(attribute.Bool("auth.user_exists", false))
	registrationID := uuid.New().String()

	code := utils.GenerateVerificationCode()
	slog.InfoContext(ctx, "verification code generated", "registration_id", registrationID)

	redisKey := "registration:" + registrationID
	if err := s.redisRepo.HSet(ctx, redisKey, "email", email, "code", code); err != nil {
		slog.ErrorContext(ctx, "SubmitEmail: redis HSet", "key", redisKey, "error", err)
		return nil, status.Error(codes.Internal, "failed to store registration data")
	}
	if err := s.redisRepo.Expire(ctx, redisKey, 15*time.Minute); err != nil {
		slog.WarnContext(ctx, "SubmitEmail: redis Expire", "key", redisKey, "error", err)
	}

	if err := s.redisRepo.XAdd(ctx, "pinz:auth:email:tasks", map[string]interface{}{
		"email": email,
		"code": code,
		"registration_id": registrationID,
	}); err != nil {
		slog.ErrorContext(ctx, "SubmitEmail: failed to enqueue email task", "registration_id", registrationID, "error", err)
	}

	return &pb.SubmitEmailResponse{IsRegistered: false, RegistrationKey: registrationID}, nil
}

func (s *AuthService) VerifyEmailCode(ctx context.Context, req *pb.VerifyEmailCodeRequest) (*pb.VerifyEmailCodeResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.VerifyEmailCode",
		trace.WithAttributes(attribute.String("auth.flow", "email_verification")),
	)
	defer span.End()

	rid := req.GetRegistrationId()
	code := req.GetVerificationCode()
	if rid == "" || code == "" {
		return nil, status.Error(codes.InvalidArgument, "registration_id and verification_code are required")
	}

	redisKey := "registration:" + rid
	data, err := s.redisRepo.HGetAll(ctx, redisKey)
	if err != nil || len(data) == 0 {
		return nil, status.Error(codes.NotFound, "invalid or expired registration id")
	}
	if data["code"] != code {
		span.SetAttributes(attribute.Bool("auth.code_valid", false))
		return &pb.VerifyEmailCodeResponse{Success: false}, status.Error(codes.InvalidArgument, "invalid verification code")
	}

	span.SetAttributes(attribute.Bool("auth.code_valid", true))
	verifiedKey := "verified:" + rid
	if err := s.redisRepo.SetEX(ctx, verifiedKey, data["email"], 2*time.Hour); err != nil {
		return nil, status.Error(codes.Internal, "failed to finalize verification")
	}
	_ = s.redisRepo.Del(ctx, redisKey)

	return &pb.VerifyEmailCodeResponse{Success: true}, nil
}

func (s *AuthService) PasskeyRegisterBegin(ctx context.Context, req *pb.PasskeyRegisterBeginRequest) (*pb.PasskeyRegisterBeginResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.PasskeyRegisterBegin",
		trace.WithAttributes(attribute.String("auth.flow", "passkey_registration")),
	)
	defer span.End()

	rid := req.GetRegistrationId()
	username := req.GetUsername()
	if rid == "" || username == "" {
		return nil, status.Error(codes.InvalidArgument, "registration_id and username are required")
	}
	if err := s.validator.Var(username, "required,min=4,max=20"); err != nil {
		return nil, status.Error(codes.InvalidArgument, "username must be 4–20 characters")
	}
	if !validateUsernameFormat(username) {
		return nil, status.Error(codes.InvalidArgument, "username must contain only letters, digits, hyphens and underscores")
	}

	verifiedKey := "verified:" + rid
	if _, err := s.redisRepo.Get(ctx, verifiedKey); err != nil {
		return nil, status.Error(codes.NotFound, "invalid or expired registration id")
	}

	pendingUserID := uuid.New()
	user := &pendingUser{
		id: pendingUserID[:],
		name: username,
		displayName: username,
	}

	creation, session, err := s.wa.BeginRegistration(user,
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		webauthn.WithExtensions(map[string]any{"credProps": true}),
	)
	if err != nil {
		slog.ErrorContext(ctx, "PasskeyRegisterBegin: begin registration", "error", err)
		return nil, status.Error(codes.Internal, "failed to begin passkey registration")
	}

	rs := regSession{
		PendingUserID: pendingUserID.String(),
		Username: username,
		SessionData: *session,
	}
	rsJSON, err := json.Marshal(rs)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to marshal registration session")
	}
	sessionKey := "webauthn:reg:" + rid
	if err := s.redisRepo.SetEX(ctx, sessionKey, rsJSON, 5*time.Minute); err != nil {
		return nil, status.Error(codes.Internal, "failed to store registration session")
	}

	_ = s.redisRepo.Expire(ctx, verifiedKey, 5*time.Minute)

	optionsJSON, err := json.Marshal(creation)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to marshal creation options")
	}
	return &pb.PasskeyRegisterBeginResponse{OptionsJson: optionsJSON}, nil
}

func (s *AuthService) PasskeyRegisterFinish(ctx context.Context, req *pb.PasskeyRegisterFinishRequest) (*pb.PasskeyRegisterFinishResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.PasskeyRegisterFinish",
		trace.WithAttributes(attribute.String("auth.flow", "passkey_registration")),
	)
	defer span.End()

	rid := req.GetRegistrationId()
	credJSON := req.GetCredentialJson()
	if rid == "" || len(credJSON) == 0 {
		return nil, status.Error(codes.InvalidArgument, "registration_id and credential_json are required")
	}

	verifiedKey := "verified:" + rid
	email, err := s.redisRepo.Get(ctx, verifiedKey)
	if err != nil {
		return nil, status.Error(codes.NotFound, "invalid or expired registration id")
	}

	sessionKey := "webauthn:reg:" + rid
	sessionRaw, err := s.redisRepo.Get(ctx, sessionKey)
	if err != nil {
		return nil, status.Error(codes.NotFound, "registration session not found or expired")
	}

	var rs regSession
	if err := json.Unmarshal([]byte(sessionRaw), &rs); err != nil {
		return nil, status.Error(codes.Internal, "failed to parse registration session")
	}

	pendingUID, err := uuid.Parse(rs.PendingUserID)
	if err != nil {
		return nil, status.Error(codes.Internal, "invalid pending user id")
	}
	user := &pendingUser{
		id: pendingUID[:],
		name: rs.Username,
		displayName: rs.Username,
	}

	parsedCred, err := protocol.ParseCredentialCreationResponseBytes(credJSON)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse credential: %v", err)
	}
	credential, err := s.wa.CreateCredential(user, rs.SessionData, parsedCred)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to verify credential: %v", err)
	}

	u := &models.User{
		ID: rs.PendingUserID,
		Email: email,
		Username: rs.Username,
	}
	if err := s.userRepo.CreateUser(u); err != nil {
		if isUniqueViolation(err) {
			s.registrationCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "conflict")))
			return nil, status.Error(codes.AlreadyExists, "user with this email or username already exists")
		}
		slog.ErrorContext(ctx, "PasskeyRegisterFinish: create user", "error", err)
		s.registrationCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "error")))
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	if err := s.credRepo.CreateCredential(u.ID, credential); err != nil {
		slog.ErrorContext(ctx, "PasskeyRegisterFinish: save credential", "error", err)
		s.registrationCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "error")))
		return nil, status.Error(codes.Internal, "failed to save passkey credential")
	}

	_ = s.redisRepo.Del(ctx, verifiedKey, sessionKey)
	s.registrationCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "success")))
	slog.InfoContext(ctx, "user registered", "username", u.Username)

	return s.issueTokens(ctx, u)
}

func (s *AuthService) PasskeyLoginBegin(ctx context.Context, req *pb.PasskeyLoginBeginRequest) (*pb.PasskeyLoginBeginResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.PasskeyLoginBegin",
		trace.WithAttributes(attribute.String("auth.flow", "passkey_login")),
	)
	defer span.End()

	email := req.GetEmail()
	if email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	u, err := s.userRepo.GetUserByEmail(email)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "PasskeyLoginBegin: get user", "error", err)
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	creds, err := s.credRepo.GetCredentialsByUserID(u.ID)
	if err != nil {
		slog.ErrorContext(ctx, "PasskeyLoginBegin: get credentials", "error", err)
		return nil, status.Error(codes.Internal, "failed to get credentials")
	}
	if len(creds) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "no passkey registered for this user")
	}

	uid, err := uuid.Parse(u.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "invalid user id")
	}
	waUser := &existingUser{
		id: uid[:],
		name: u.Username,
		displayName: u.Username,
		credentials: creds,
	}

	assertion, session, err := s.wa.BeginLogin(waUser)
	if err != nil {
		slog.ErrorContext(ctx, "PasskeyLoginBegin: begin login", "error", err)
		return nil, status.Error(codes.Internal, "failed to begin passkey login")
	}

	ls := loginSession{UserID: u.ID, SessionData: *session}
	lsJSON, err := json.Marshal(ls)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to marshal login session")
	}
	sessionKey := "webauthn:login:" + u.ID
	if err := s.redisRepo.SetEX(ctx, sessionKey, lsJSON, 5*time.Minute); err != nil {
		return nil, status.Error(codes.Internal, "failed to store login session")
	}

	optionsJSON, err := json.Marshal(assertion)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to marshal assertion options")
	}
	return &pb.PasskeyLoginBeginResponse{OptionsJson: optionsJSON}, nil
}

func (s *AuthService) PasskeyLoginFinish(ctx context.Context, req *pb.PasskeyLoginFinishRequest) (*pb.PasskeyLoginFinishResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.PasskeyLoginFinish",
		trace.WithAttributes(attribute.String("auth.flow", "passkey_login")),
	)
	defer span.End()

	email := req.GetEmail()
	credJSON := req.GetCredentialJson()
	if email == "" || len(credJSON) == 0 {
		return nil, status.Error(codes.InvalidArgument, "email and credential_json are required")
	}

	u, err := s.userRepo.GetUserByEmail(email)
	if err == sql.ErrNoRows {
		s.loginCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("method", "passkey"),
			attribute.String("status", "not_found"),
		))
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	if err != nil {
		slog.ErrorContext(ctx, "PasskeyLoginFinish: get user", "error", err)
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	sessionKey := "webauthn:login:" + u.ID
	sessionRaw, err := s.redisRepo.Get(ctx, sessionKey)
	if err != nil {
		return nil, status.Error(codes.NotFound, "login session not found or expired")
	}

	var ls loginSession
	if err := json.Unmarshal([]byte(sessionRaw), &ls); err != nil {
		return nil, status.Error(codes.Internal, "failed to parse login session")
	}

	creds, err := s.credRepo.GetCredentialsByUserID(u.ID)
	if err != nil {
		slog.ErrorContext(ctx, "PasskeyLoginFinish: get credentials", "error", err)
		return nil, status.Error(codes.Internal, "failed to get credentials")
	}

	uid, err := uuid.Parse(u.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "invalid user id")
	}
	waUser := &existingUser{
		id: uid[:],
		name: u.Username,
		displayName: u.Username,
		credentials: creds,
	}

	parsedAssertion, err := protocol.ParseCredentialRequestResponseBytes(credJSON)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse assertion: %v", err)
	}
	updatedCred, err := s.wa.ValidateLogin(waUser, ls.SessionData, parsedAssertion)
	if err != nil {
		s.loginCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("method", "passkey"),
			attribute.String("status", "invalid"),
		))
		return nil, status.Errorf(codes.Unauthenticated, "passkey verification failed: %v", err)
	}

	if err := s.credRepo.UpdateCredential(updatedCred); err != nil {
		slog.WarnContext(ctx, "PasskeyLoginFinish: update credential", "error", err)
	}
	_ = s.redisRepo.Del(ctx, sessionKey)

	s.loginCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("method", "passkey"),
		attribute.String("status", "success"),
	))
	slog.InfoContext(ctx, "user logged in", "username", u.Username)

	resp, err := s.issueTokens(ctx, u)
	if err != nil {
		return nil, err
	}
	return &pb.PasskeyLoginFinishResponse{
		AccessToken: resp.AccessToken,
		RefreshToken: resp.RefreshToken,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.RefreshToken")
	defer span.End()

	rt := req.GetRefreshToken()
	if rt == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}

	rec, err := s.userRepo.GetRefreshToken(rt)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}
	if err != nil {
		slog.ErrorContext(ctx, "RefreshToken: get token", "error", err)
		return nil, status.Error(codes.Internal, "failed to get refresh token")
	}
	if time.Now().After(rec.ExpiresAt) {
		return nil, status.Error(codes.Unauthenticated, "refresh token expired")
	}

	u, err := s.userRepo.GetUserByID(rec.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "RefreshToken: get user", "error", err)
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		return nil, status.Error(codes.Internal, "JWT_SECRET_KEY not set")
	}
	accessToken, err := utils.GenerateAccessToken(u.ID, u.Username, secret)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate access token")
	}

	s.tokenRefreshCounter.Add(ctx, 1)
	return &pb.RefreshTokenResponse{AccessToken: accessToken}, nil
}

func (s *AuthService) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.Logout")
	defer span.End()

	rt := req.GetRefreshToken()
	if rt == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}

	rec, err := s.userRepo.GetRefreshToken(rt)
	if err == sql.ErrNoRows {
		return &pb.LogoutResponse{Success: true}, nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "Logout: get token", "error", err)
		return nil, status.Error(codes.Internal, "failed to get refresh token")
	}
	if err := s.userRepo.DeleteRefreshToken(rec.ID); err != nil {
		slog.ErrorContext(ctx, "Logout: delete token", "error", err)
		return nil, status.Error(codes.Internal, "failed to delete refresh token")
	}
	return &pb.LogoutResponse{Success: true}, nil
}

// issueTokens creates an access+refresh token pair and persists the session.
func (s *AuthService) issueTokens(ctx context.Context, u *models.User) (*pb.PasskeyRegisterFinishResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.issueTokens")
	defer span.End()

	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		return nil, status.Error(codes.Internal, "JWT_SECRET_KEY not set")
	}
	accessToken, err := utils.GenerateAccessToken(u.ID, u.Username, secret)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate access token")
	}
	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate refresh token")
	}
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	if err := s.userRepo.AddSession(u.ID, refreshToken, expiresAt); err != nil {
		slog.ErrorContext(ctx, "issueTokens: add session", "error", err)
		return nil, status.Error(codes.Internal, "failed to save session")
	}
	return &pb.PasskeyRegisterFinishResponse{
		AccessToken: accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) DevLogin(ctx context.Context, req *pb.DevLoginRequest) (*pb.DevLoginResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.DevLogin",
		trace.WithAttributes(attribute.String("auth.flow", "dev_login")),
	)
	defer span.End()

	email := req.GetEmail()
	if email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	u, err := s.userRepo.GetUserByEmail(email)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "DevLogin: get user", "error", err)
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	s.loginCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("method", "dev"),
		attribute.String("status", "success"),
	))

	resp, err := s.issueTokens(ctx, u)
	if err != nil {
		return nil, err
	}
	return &pb.DevLoginResponse{
		AccessToken: resp.AccessToken,
		RefreshToken: resp.RefreshToken,
	}, nil
}

func (s *AuthService) userToProto(ctx context.Context, u *models.User) *pb.User {
	avatar := ""
	if u.AvatarURL != "" && s.s3 != nil {
		url, err := s.s3.ReadURL(ctx, u.AvatarURL)
		if err != nil {
			slog.WarnContext(ctx, "userToProto: presign avatar GET", "key", u.AvatarURL, "error", err)
		} else {
			avatar = url
		}
	}
	return &pb.User{
		Id: u.ID,
		Username: u.Username,
		Email: u.Email,
		AvatarUrl: avatar,
		CreatedAtUnix: u.CreatedAt.Unix(),
	}
}

func (s *AuthService) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.GetProfile")
	defer span.End()

	userID := req.GetUserId()
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	u, err := s.userRepo.GetUserByID(userID)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "GetProfile: get user", "error", err)
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	return &pb.GetProfileResponse{User: s.userToProto(ctx, u)}, nil
}

// GetUsersProfiles — batched публичные профили (без email). Используется api-gateway
// для обогащения списков участников / current_initiator в ответах trip-service.
// avatar_url presigned; если пользователь не найден — в ответе остаётся пустой
// объект с заполненным user_id, клиент фильтрует по непустому username.
func (s *AuthService) GetUsersProfiles(ctx context.Context, req *pb.GetUsersProfilesRequest) (*pb.GetUsersProfilesResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.GetUsersProfiles")
	defer span.End()

	userIDs := req.GetUserIds()
	if len(userIDs) == 0 {
		return &pb.GetUsersProfilesResponse{Profiles: nil}, nil
	}
	users, err := s.userRepo.GetUsersByIDs(userIDs)
	if err != nil {
		slog.ErrorContext(ctx, "GetUsersProfiles: GetUsersByIDs", "error", err)
		return nil, status.Error(codes.Internal, "failed to load users")
	}
	byID := make(map[string]*models.User, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}
	profiles := make([]*pb.PublicUserProfile, 0, len(userIDs))
	for _, id := range userIDs {
		u, ok := byID[id]
		if !ok {
			profiles = append(profiles, &pb.PublicUserProfile{UserId: id})
			continue
		}
		avatar := ""
		if u.AvatarURL != "" && s.s3 != nil {
			if url, perr := s.s3.ReadURL(ctx, u.AvatarURL); perr == nil {
				avatar = url
			}
		}
		profiles = append(profiles, &pb.PublicUserProfile{
			UserId:        u.ID,
			Username:      u.Username,
			AvatarUrl:     avatar,
			CreatedAtUnix: u.CreatedAt.Unix(),
		})
	}
	return &pb.GetUsersProfilesResponse{Profiles: profiles}, nil
}

// GetPublicUserProfile возвращает публичный профиль другого пользователя (без email)
// и его список желаемых мест за один gRPC round-trip (decomp #16, ТЗ 1.7.2).
// Не используем GetProfile, чтобы email не покидал auth-service для публичного запроса.
func (s *AuthService) GetPublicUserProfile(ctx context.Context, req *pb.GetPublicUserProfileRequest) (*pb.GetPublicUserProfileResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.GetPublicUserProfile")
	defer span.End()

	userID := req.GetUserId()
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	u, err := s.userRepo.GetUserByID(userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "GetPublicUserProfile: get user", "error", err)
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	avatar := ""
	if u.AvatarURL != "" && s.s3 != nil {
		if url, perr := s.s3.ReadURL(ctx, u.AvatarURL); perr == nil {
			avatar = url
		} else {
			slog.WarnContext(ctx, "GetPublicUserProfile: presign avatar", "key", u.AvatarURL, "error", perr)
		}
	}

	places, err := s.listDesiredPlacesProto(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &pb.GetPublicUserProfileResponse{
		Profile: &pb.PublicUserProfile{
			UserId:        u.ID,
			Username:      u.Username,
			AvatarUrl:     avatar,
			CreatedAtUnix: u.CreatedAt.Unix(),
		},
		DesiredPlaces: places,
	}, nil
}

func (s *AuthService) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.UpdateProfile")
	defer span.End()

	userID := req.GetUserId()
	username := req.GetUsername()
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if err := s.validator.Var(username, "required,min=4,max=20"); err != nil {
		return nil, status.Error(codes.InvalidArgument, "username must be 4–20 characters")
	}
	if !validateUsernameFormat(username) {
		return nil, status.Error(codes.InvalidArgument, "username must contain only letters, digits, hyphens and underscores")
	}

	u, err := s.userRepo.UpdateUsername(userID, username)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, status.Error(codes.AlreadyExists, "username already taken")
		}
		slog.ErrorContext(ctx, "UpdateProfile: update username", "error", err)
		return nil, status.Error(codes.Internal, "failed to update username")
	}

	return &pb.UpdateProfileResponse{User: s.userToProto(ctx, u)}, nil
}

func (s *AuthService) ChangeEmail(ctx context.Context, req *pb.ChangeEmailRequest) (*pb.ChangeEmailResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.ChangeEmail")
	defer span.End()

	userID := req.GetUserId()
	newEmail := strings.TrimSpace(req.GetNewEmail())
	if userID == "" || newEmail == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and new_email are required")
	}
	if err := s.validator.Var(newEmail, "required,email"); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email: %v", err)
	}

	existing, err := s.userRepo.GetUserByEmail(newEmail)
	if err == nil && existing.ID != userID {
		return nil, status.Error(codes.AlreadyExists, "email already in use")
	}
	if err != nil && err != sql.ErrNoRows {
		slog.ErrorContext(ctx, "ChangeEmail: check existing email", "error", err)
		return nil, status.Error(codes.Internal, "failed to check email")
	}

	code := utils.GenerateVerificationCode()
	redisKey := "email_change:" + userID
	if err := s.redisRepo.HSet(ctx, redisKey, "email", newEmail, "code", code); err != nil {
		slog.ErrorContext(ctx, "ChangeEmail: redis HSet", "error", err)
		return nil, status.Error(codes.Internal, "failed to store email change data")
	}
	if err := s.redisRepo.Expire(ctx, redisKey, 15*time.Minute); err != nil {
		slog.WarnContext(ctx, "ChangeEmail: redis Expire", "error", err)
	}

	if err := s.redisRepo.XAdd(ctx, "pinz:auth:email:tasks", map[string]interface{}{
		"email": newEmail,
		"code": code,
		"user_id": userID,
	}); err != nil {
		slog.ErrorContext(ctx, "ChangeEmail: failed to enqueue email task", "error", err)
	}

	slog.InfoContext(ctx, "email change initiated", "user_id", userID)
	return &pb.ChangeEmailResponse{Success: true}, nil
}

func (s *AuthService) ConfirmEmailChange(ctx context.Context, req *pb.ConfirmEmailChangeRequest) (*pb.ConfirmEmailChangeResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.ConfirmEmailChange")
	defer span.End()

	userID := req.GetUserId()
	code := req.GetVerificationCode()
	if userID == "" || code == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and verification_code are required")
	}

	redisKey := "email_change:" + userID
	data, err := s.redisRepo.HGetAll(ctx, redisKey)
	if err != nil || len(data) == 0 {
		return nil, status.Error(codes.NotFound, "no pending email change or expired")
	}
	if data["code"] != code {
		return nil, status.Error(codes.InvalidArgument, "invalid verification code")
	}

	u, err := s.userRepo.UpdateEmail(userID, data["email"])
	if err != nil {
		if isUniqueViolation(err) {
			return nil, status.Error(codes.AlreadyExists, "email already in use")
		}
		slog.ErrorContext(ctx, "ConfirmEmailChange: update email", "error", err)
		return nil, status.Error(codes.Internal, "failed to update email")
	}

	_ = s.redisRepo.Del(ctx, redisKey)
	slog.InfoContext(ctx, "email changed", "user_id", userID, "new_email", data["email"])
	return &pb.ConfirmEmailChangeResponse{User: s.userToProto(ctx, u)}, nil
}

func (s *AuthService) RequestAvatarUpload(ctx context.Context, req *pb.RequestAvatarUploadRequest) (*pb.RequestAvatarUploadResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.RequestAvatarUpload")
	defer span.End()

	userID := req.GetUserId()
	filename := req.GetFilename()
	contentType := req.GetContentType()
	if userID == "" || filename == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and filename are required")
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".jpg"
	}
	switch ext {
	case ".jpg", ".jpeg", ".png", ".heic":
		// allowed
	default:
		return nil, status.Error(codes.InvalidArgument, "avatar must be .jpg, .jpeg, .png or .heic")
	}

	if s.s3 == nil {
		return nil, status.Error(codes.Unavailable, "avatar upload is not configured")
	}
	s3Key := fmt.Sprintf("avatars/%s/%s%s", userID, uuid.NewString(), ext)

	uploadURL, err := s.s3.PresignedUploadURL(ctx, s3Key, contentType)
	if err != nil {
		slog.ErrorContext(ctx, "RequestAvatarUpload: presign", "error", err)
		return nil, status.Error(codes.Internal, "failed to generate upload URL")
	}

	return &pb.RequestAvatarUploadResponse{
		UploadUrl: uploadURL,
		S3Key: s3Key,
	}, nil
}

func (s *AuthService) ConfirmAvatarUpload(ctx context.Context, req *pb.ConfirmAvatarUploadRequest) (*pb.ConfirmAvatarUploadResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.ConfirmAvatarUpload")
	defer span.End()

	userID := req.GetUserId()
	s3Key := req.GetS3Key()
	if userID == "" || s3Key == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and s3_key are required")
	}

	oldUser, err := s.userRepo.GetUserByID(userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		slog.ErrorContext(ctx, "ConfirmAvatarUpload: get user", "error", err)
	}
	if oldUser != nil && oldUser.AvatarURL != "" && oldUser.AvatarURL != s3Key && s.s3 != nil {
		if err := s.s3.DeleteObject(ctx, oldUser.AvatarURL); err != nil {
			slog.ErrorContext(ctx, "ConfirmAvatarUpload: delete old avatar (best-effort)", "key", oldUser.AvatarURL, "error", err)
		}
	}

	u, err := s.userRepo.UpdateAvatarURL(userID, s3Key)
	if err != nil {
		slog.ErrorContext(ctx, "ConfirmAvatarUpload: update avatar", "error", err)
		return nil, status.Error(codes.Internal, "failed to update avatar")
	}

	return &pb.ConfirmAvatarUploadResponse{User: s.userToProto(ctx, u)}, nil
}

func (s *AuthService) DeleteAvatar(ctx context.Context, req *pb.DeleteAvatarRequest) (*pb.DeleteAvatarResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.DeleteAvatar")
	defer span.End()

	userID := req.GetUserId()
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	u, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		slog.ErrorContext(ctx, "DeleteAvatar: get user", "error", err)
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	if u.AvatarURL != "" && s.s3 != nil {
		if err := s.s3.DeleteObject(ctx, u.AvatarURL); err != nil {
			slog.ErrorContext(ctx, "DeleteAvatar: s3 delete (best-effort)", "key", u.AvatarURL, "error", err)
		}
	}

	u, err = s.userRepo.UpdateAvatarURL(userID, "")
	if err != nil {
		slog.ErrorContext(ctx, "DeleteAvatar: clear avatar_url", "error", err)
		return nil, status.Error(codes.Internal, "failed to delete avatar")
	}

	return &pb.DeleteAvatarResponse{User: s.userToProto(ctx, u)}, nil
}

func (s *AuthService) DeleteAccount(ctx context.Context, req *pb.DeleteAccountRequest) (*pb.DeleteAccountResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.DeleteAccount")
	defer span.End()

	userID := req.GetUserId()
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	if err := s.userRepo.DeleteUser(userID); err != nil {
		slog.ErrorContext(ctx, "DeleteAccount: delete user", "error", err)
		return nil, status.Error(codes.Internal, "failed to delete account")
	}

	slog.InfoContext(ctx, "account deleted", "user_id", userID)
	return &pb.DeleteAccountResponse{Success: true}, nil
}

func isUniqueViolation(err error) bool {
	var e *pgconn.PgError
	if errors.As(err, &e) && e.Code == "23505" {
		return true
	}
	return false
}
