package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
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
	"pinz/backend/auth-service/internal/repositories"
	"pinz/backend/auth-service/internal/utils"
	pb "pinz/backend/auth-service/pkg/proto"
)

// pendingUser implements webauthn.User for a not-yet-persisted registration.
type pendingUser struct {
	id          []byte
	name        string
	displayName string
	credentials []webauthn.Credential
}

func (u *pendingUser) WebAuthnID() []byte                         { return u.id }
func (u *pendingUser) WebAuthnName() string                       { return u.name }
func (u *pendingUser) WebAuthnDisplayName() string                { return u.displayName }
func (u *pendingUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// existingUser implements webauthn.User for a fully-persisted user.
type existingUser struct {
	id          []byte
	name        string
	displayName string
	credentials []webauthn.Credential
}

func (u *existingUser) WebAuthnID() []byte                         { return u.id }
func (u *existingUser) WebAuthnName() string                       { return u.name }
func (u *existingUser) WebAuthnDisplayName() string                { return u.displayName }
func (u *existingUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// regSession is stored in Redis between PasskeyRegisterBegin and PasskeyRegisterFinish.
type regSession struct {
	PendingUserID string               `json:"pending_user_id"`
	Username      string               `json:"username"`
	SessionData   webauthn.SessionData `json:"session_data"`
}

// loginSession is stored in Redis between PasskeyLoginBegin and PasskeyLoginFinish.
type loginSession struct {
	UserID      string               `json:"user_id"`
	SessionData webauthn.SessionData `json:"session_data"`
}

type AuthService struct {
	pb.UnimplementedAuthServiceServer
	userRepo  *repositories.UserRepository
	credRepo  *repositories.CredentialRepository
	redisRepo *repositories.RedisRepository
	validator *validator.Validate
	wa        *webauthn.WebAuthn

	tracer              trace.Tracer
	loginCounter        metric.Int64Counter
	registrationCounter metric.Int64Counter
	tokenRefreshCounter metric.Int64Counter
}

func NewAuthService(
	userRepo *repositories.UserRepository,
	credRepo *repositories.CredentialRepository,
	redisRepo *repositories.RedisRepository,
	validator *validator.Validate,
	wa *webauthn.WebAuthn,
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
		userRepo:            userRepo,
		credRepo:            credRepo,
		redisRepo:           redisRepo,
		validator:           validator,
		wa:                  wa,
		tracer:              tracer,
		loginCounter:        loginCounter,
		registrationCounter: registrationCounter,
		tokenRefreshCounter: tokenRefreshCounter,
	}
}

func (s *AuthService) SubmitEmail(ctx context.Context, req *pb.SubmitEmailRequest) (*pb.SubmitEmailResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.SubmitEmail")
	defer span.End()

	email := req.GetEmail()
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
  
	code := "1111" // TODO: replace with utils.GenerateVerificationCode() when email sending is ready
	slog.InfoContext(ctx, "verification code generated", "registration_id", registrationID, "code", code)

	redisKey := "registration:" + registrationID
	if err := s.redisRepo.HSet(ctx, redisKey, "email", email, "code", code); err != nil {
		slog.ErrorContext(ctx, "SubmitEmail: redis HSet", "key", redisKey, "error", err)
		return nil, status.Error(codes.Internal, "failed to store registration data")
	}
	if err := s.redisRepo.Expire(ctx, redisKey, 15*time.Minute); err != nil {
		slog.WarnContext(ctx, "SubmitEmail: redis Expire", "key", redisKey, "error", err)
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

	verifiedKey := "verified:" + rid
	if _, err := s.redisRepo.Get(ctx, verifiedKey); err != nil {
		return nil, status.Error(codes.NotFound, "invalid or expired registration id")
	}

	pendingUserID := uuid.New()
	user := &pendingUser{
		id:          pendingUserID[:],
		name:        username,
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
		Username:      username,
		SessionData:   *session,
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
		id:          pendingUID[:],
		name:        rs.Username,
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
		ID:       rs.PendingUserID,
		Email:    email,
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
		id:          uid[:],
		name:        u.Username,
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
		id:          uid[:],
		name:        u.Username,
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
		AccessToken:  resp.AccessToken,
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
		AccessToken:  accessToken,
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
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
	}, nil
}

func isUniqueViolation(err error) bool {
	var e *pgconn.PgError
	if errors.As(err, &e) && e.Code == "23505" {
		return true
	}
	return false
}
