package services

import (
	"context"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"

	"pinz/backend/auth-service/internal/models"
)

// UserRepositoryInterface is used by AuthService for user and session persistence.
// *repositories.UserRepository implements this interface.
type UserRepositoryInterface interface {
	GetUserByEmail(email string) (*models.User, error)
	CreateUser(u *models.User) error
	AddSession(userID, token string, expiresAt interface{}) error
	GetRefreshToken(token string) (*models.RefreshToken, error)
	GetUserByID(userID string) (*models.User, error)
	GetUsersByIDs(userIDs []string) ([]*models.User, error)
	DeleteRefreshToken(id string) error
	DeleteUserRefreshTokens(userID string) error
	UpdateUsername(userID, username string) (*models.User, error)
	UpdateEmail(userID, email string) (*models.User, error)
	UpdateAvatarURL(userID, avatarURL string) (*models.User, error)
	DeleteUser(userID string) error
}

// RedisRepositoryInterface is used by AuthService for registration and session cache.
// *repositories.RedisRepository implements this interface.
type RedisRepositoryInterface interface {
	HSet(ctx context.Context, key string, values ...interface{}) error
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Get(ctx context.Context, key string) (string, error)
	SetEX(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	XAdd(ctx context.Context, stream string, values map[string]interface{}) error
}

// CredentialRepositoryInterface is used by AuthService for WebAuthn credentials.
// *repositories.CredentialRepository implements this interface.
type CredentialRepositoryInterface interface {
	CreateCredential(userID string, cred *webauthn.Credential) error
	GetCredentialsByUserID(userID string) ([]webauthn.Credential, error)
	UpdateCredential(cred *webauthn.Credential) error
}
