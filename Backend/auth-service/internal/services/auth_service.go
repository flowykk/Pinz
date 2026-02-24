package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/auth-service/internal/models"
	"pinz/backend/auth-service/internal/repositories"
	"pinz/backend/auth-service/internal/utils"
	pb "pinz/backend/auth-service/pkg/proto"
)

type AuthService struct {
	pb.UnimplementedAuthServiceServer
	userRepo  *repositories.UserRepository
	redisRepo *repositories.RedisRepository
	validator *validator.Validate
}

func NewAuthService(
	userRepo *repositories.UserRepository,
	redisRepo *repositories.RedisRepository,
	validator *validator.Validate,
) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		redisRepo: redisRepo,
		validator: validator,
	}
}

func (s *AuthService) SubmitEmail(ctx context.Context, req *pb.SubmitEmailRequest) (*pb.SubmitEmailResponse, error) {
	email := req.GetEmail()
	if email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	if err := s.validator.Var(email, "required,email"); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email: %v", err)
	}

	_, err := s.userRepo.GetUserByEmail(email)
	if err == nil {
		return &pb.SubmitEmailResponse{IsRegistered: true, RegistrationKey: ""}, nil
	}
	if err != sql.ErrNoRows {
		log.Printf("SubmitEmail: get user by email %s: %v", email, err)
		return nil, status.Error(codes.Internal, "failed to check user existence")
	}

	registrationID := uuid.New().String()
	code := utils.GenerateVerificationCode()
	fmt.Println(code)
	redisKey := "registration:" + registrationID
	if err := s.redisRepo.HSet(ctx, redisKey, "email", email, "code", code); err != nil {
		log.Printf("SubmitEmail: redis HSet %s: %v", redisKey, err)
		return nil, status.Error(codes.Internal, "failed to store registration data")
	}
	if err := s.redisRepo.Expire(ctx, redisKey, 15*time.Minute); err != nil {
		log.Printf("SubmitEmail: redis Expire %s: %v", redisKey, err)
	}

	return &pb.SubmitEmailResponse{IsRegistered: false, RegistrationKey: registrationID}, nil
}

func (s *AuthService) VerifyEmailCode(ctx context.Context, req *pb.VerifyEmailCodeRequest) (*pb.VerifyEmailCodeResponse, error) {
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
		return &pb.VerifyEmailCodeResponse{Success: false}, status.Error(codes.InvalidArgument, "invalid verification code")
	}

	verifiedKey := "verified:" + rid
	if err := s.redisRepo.SetEX(ctx, verifiedKey, data["email"], 2*time.Hour); err != nil {
		return nil, status.Error(codes.Internal, "failed to finalize verification")
	}
	_ = s.redisRepo.Del(ctx, redisKey)

	return &pb.VerifyEmailCodeResponse{Success: true}, nil
}

func (s *AuthService) SetPasswordAndUsername(ctx context.Context, req *pb.SetPasswordAndUsernameRequest) (*pb.SetPasswordAndUsernameResponse, error) {
	rid := req.GetRegistrationId()
	password := req.GetPassword()
	username := req.GetUsername()
	if rid == "" || password == "" || username == "" {
		return nil, status.Error(codes.InvalidArgument, "registration_id, password and username are required")
	}
	var err error
	if err = s.validator.Var(password, "required,min=6"); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "password must be at least 6 characters")
	}
	if err = s.validator.Var(username, "required,min=4,max=20"); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "username must be 4–20 characters")
	}

	verifiedKey := "verified:" + rid
	email, err := s.redisRepo.Get(ctx, verifiedKey)
	if err != nil {
		return nil, status.Error(codes.NotFound, "invalid or expired registration id")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to hash password")
	}

	u := &models.User{
		Email:        email,
		PasswordHash: string(hash),
		Username:     username,
	}
	if err := s.userRepo.CreateUser(u); err != nil {
		if isUniqueViolation(err) {
			return nil, status.Error(codes.AlreadyExists, "user with this email or username already exists")
		}
		log.Printf("SetPasswordAndUsername: create user: %v", err)
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	_ = s.redisRepo.Del(ctx, verifiedKey)

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
		log.Printf("SetPasswordAndUsername: add session: %v", err)
		return nil, status.Error(codes.Internal, "failed to save session")
	}

	return &pb.SetPasswordAndUsernameResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	email := req.GetEmail()
	password := req.GetPassword()
	if email == "" || password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	u, err := s.userRepo.GetUserByEmail(email)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	if err != nil {
		log.Printf("Login: get user: %v", err)
		return nil, status.Error(codes.Internal, "failed to get user")
	}
	if err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

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
		log.Printf("Login: add session: %v", err)
		return nil, status.Error(codes.Internal, "failed to save session")
	}

	return &pb.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	rt := req.GetRefreshToken()
	if rt == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}

	rec, err := s.userRepo.GetRefreshToken(rt)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}
	if err != nil {
		log.Printf("RefreshToken: get token: %v", err)
		return nil, status.Error(codes.Internal, "failed to get refresh token")
	}
	if time.Now().After(rec.ExpiresAt) {
		return nil, status.Error(codes.Unauthenticated, "refresh token expired")
	}

	u, err := s.userRepo.GetUserByID(rec.UserID)
	if err != nil {
		log.Printf("RefreshToken: get user: %v", err)
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

	return &pb.RefreshTokenResponse{AccessToken: accessToken}, nil
}

func (s *AuthService) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	rt := req.GetRefreshToken()
	if rt == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}

	rec, err := s.userRepo.GetRefreshToken(rt)
	if err == sql.ErrNoRows {
		return &pb.LogoutResponse{Success: true}, nil
	}
	if err != nil {
		log.Printf("Logout: get token: %v", err)
		return nil, status.Error(codes.Internal, "failed to get refresh token")
	}
	if err := s.userRepo.DeleteRefreshToken(rec.ID); err != nil {
		log.Printf("Logout: delete token: %v", err)
		return nil, status.Error(codes.Internal, "failed to delete refresh token")
	}
	return &pb.LogoutResponse{Success: true}, nil
}

func isUniqueViolation(err error) bool {
	var e *pgconn.PgError
	if errors.As(err, &e) && e.Code == "23505" {
		return true
	}
	return false
}
