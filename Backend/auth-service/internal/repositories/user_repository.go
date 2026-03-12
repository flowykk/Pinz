package repositories

import (
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"

	"pinz/backend/auth-service/internal/models"
)

var (
	psq = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	q := psq.Select("id", "email", "username", "avatar_url", "created_at").
		From("users").
		Where(sq.Eq{"email": email})
	row := q.RunWith(r.db).QueryRow()
	var u models.User
	var avatarURL sql.NullString
	err := row.Scan(&u.ID, &u.Email, &u.Username, &avatarURL, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	if avatarURL.Valid {
		u.AvatarURL = avatarURL.String
	}
	return &u, nil
}

func (r *UserRepository) CreateUser(u *models.User) error {
	q := psq.Insert("users").
		Columns("id", "email", "username", "avatar_url").
		Values(u.ID, u.Email, u.Username, u.AvatarURL).
		Suffix("RETURNING created_at")
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return err
	}
	row := r.db.QueryRow(sqlStr, args...)
	if err := row.Scan(&u.CreatedAt); err != nil {
		return err
	}
	return nil
}

func (r *UserRepository) AddSession(userID, token string, expiresAt interface{}) error {
	q := psq.Insert("refresh_tokens").
		Columns("user_id", "token", "expires_at").
		Values(userID, token, expiresAt)
	_, err := q.RunWith(r.db).Exec()
	return err
}

func (r *UserRepository) GetRefreshToken(token string) (*models.RefreshToken, error) {
	q := psq.Select("id", "user_id", "token", "expires_at").
		From("refresh_tokens").
		Where(sq.Eq{"token": token})
	row := q.RunWith(r.db).QueryRow()
	var rt models.RefreshToken
	err := row.Scan(&rt.ID, &rt.UserID, &rt.Token, &rt.ExpiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &rt, nil
}

func (r *UserRepository) GetUserByID(userID string) (*models.User, error) {
	q := psq.Select("id", "email", "username", "avatar_url", "created_at").
		From("users").
		Where(sq.Eq{"id": userID})
	row := q.RunWith(r.db).QueryRow()
	var u models.User
	var avatarURL sql.NullString
	err := row.Scan(&u.ID, &u.Email, &u.Username, &avatarURL, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	if avatarURL.Valid {
		u.AvatarURL = avatarURL.String
	}
	return &u, nil
}

func (r *UserRepository) DeleteRefreshToken(id string) error {
	q := psq.Delete("refresh_tokens").Where(sq.Eq{"id": id})
	_, err := q.RunWith(r.db).Exec()
	return err
}

func (r *UserRepository) DeleteUserRefreshTokens(userID string) error {
	q := psq.Delete("refresh_tokens").Where(sq.Eq{"user_id": userID})
	_, err := q.RunWith(r.db).Exec()
	return err
}
