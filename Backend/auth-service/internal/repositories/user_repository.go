package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"pinz/backend/auth-service/internal/db/sqlcdb"
	"pinz/backend/auth-service/internal/models"
)

type UserRepository struct {
	q *sqlcdb.Queries
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{q: sqlcdb.New(db)}
}

func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	u, err := r.q.GetUserByEmail(context.Background(), email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return userFromSQLC(u), nil
}

func (r *UserRepository) CreateUser(u *models.User) error {
	id, err := uuid.Parse(u.ID)
	if err != nil {
		return err
	}
	avatar := sql.NullString{String: u.AvatarURL, Valid: u.AvatarURL != ""}
	createdAt, err := r.q.CreateUser(context.Background(), sqlcdb.CreateUserParams{
		ID: id,
		Email: u.Email,
		Username: u.Username,
		AvatarUrl: avatar,
	})
	if err != nil {
		return err
	}
	u.CreatedAt = createdAt
	return nil
}

func (r *UserRepository) AddSession(userID, token string, expiresAt interface{}) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	t, ok := expiresAt.(time.Time)
	if !ok {
		return errors.New("expiresAt must be time.Time")
	}
	return r.q.AddSession(context.Background(), sqlcdb.AddSessionParams{
		UserID: uid,
		Token: token,
		ExpiresAt: t,
	})
}

func (r *UserRepository) GetRefreshToken(token string) (*models.RefreshToken, error) {
	rt, err := r.q.GetRefreshToken(context.Background(), token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &models.RefreshToken{
		ID: rt.ID.String(),
		UserID: rt.UserID.String(),
		Token: rt.Token,
		ExpiresAt: rt.ExpiresAt,
	}, nil
}

func (r *UserRepository) GetUserByID(userID string) (*models.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	u, err := r.q.GetUserByID(context.Background(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return userFromSQLC(u), nil
}

// GetUsersByIDs — batched выборка пользователей по списку id для api-gateway
// enrichment (N2). Несуществующие id просто отсутствуют в ответе. Дубликаты в
// запросе не ошибочны — Postgres вернёт их один раз, вызывающая сторона
// строит map user_id → профиль и переиспользует.
func (r *UserRepository) GetUsersByIDs(userIDs []string) ([]*models.User, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	ids := make([]uuid.UUID, 0, len(userIDs))
	for _, s := range userIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.q.GetUsersByIDs(context.Background(), ids)
	if err != nil {
		return nil, err
	}
	out := make([]*models.User, 0, len(rows))
	for _, u := range rows {
		out = append(out, userFromSQLC(u))
	}
	return out, nil
}

func (r *UserRepository) DeleteRefreshToken(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.DeleteRefreshToken(context.Background(), uid)
}

func (r *UserRepository) DeleteUserRefreshTokens(userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.q.DeleteUserRefreshTokens(context.Background(), uid)
}

func (r *UserRepository) UpdateUsername(userID, username string) (*models.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	u, err := r.q.UpdateUsername(context.Background(), sqlcdb.UpdateUsernameParams{
		ID: id,
		Username: username,
	})
	if err != nil {
		return nil, err
	}
	return userFromSQLC(u), nil
}

func (r *UserRepository) UpdateEmail(userID, email string) (*models.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	u, err := r.q.UpdateEmail(context.Background(), sqlcdb.UpdateEmailParams{
		ID: id,
		Email: email,
	})
	if err != nil {
		return nil, err
	}
	return userFromSQLC(u), nil
}

func (r *UserRepository) UpdateAvatarURL(userID, avatarURL string) (*models.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	avatar := sql.NullString{String: avatarURL, Valid: avatarURL != ""}
	u, err := r.q.UpdateAvatarURL(context.Background(), sqlcdb.UpdateAvatarURLParams{
		ID: id,
		AvatarUrl: avatar,
	})
	if err != nil {
		return nil, err
	}
	return userFromSQLC(u), nil
}

func (r *UserRepository) DeleteUser(userID string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.q.DeleteUser(context.Background(), id)
}

func userFromSQLC(u sqlcdb.User) *models.User {
	out := &models.User{
		ID: u.ID.String(),
		Email: u.Email,
		Username: u.Username,
		CreatedAt: u.CreatedAt,
	}
	if u.AvatarUrl.Valid {
		out.AvatarURL = u.AvatarUrl.String
	}
	return out
}
