package repositories_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"pinz/backend/auth-service/internal/db"
	"pinz/backend/auth-service/internal/models"
	"pinz/backend/auth-service/internal/repositories"
	"pinz/backend/auth-service/internal/testinfra"
)

func TestUserRepository_CreateUserWithCredential(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	testinfra.WithAuthPostgres(t)

	sqlDB, err := db.InitDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	repo := repositories.NewUserRepository(sqlDB)
	ctx := context.Background()

	credID := []byte("cred-id-shared")

	t.Run("happy path persists both rows", func(t *testing.T) {
		u := &models.User{
			ID:       uuid.New().String(),
			Email:    "atomic-ok@example.com",
			Username: "atomic_ok",
		}
		cred := &webauthn.Credential{ID: credID}

		require.NoError(t, repo.CreateUserWithCredential(ctx, u, cred))
		require.False(t, u.CreatedAt.IsZero())

		got, err := repo.GetUserByEmail(u.Email)
		require.NoError(t, err)
		require.Equal(t, u.ID, got.ID)
	})

	t.Run("credential conflict rolls back user", func(t *testing.T) {
		u := &models.User{
			ID:       uuid.New().String(),
			Email:    "atomic-rollback@example.com",
			Username: "atomic_rollback",
		}
		cred := &webauthn.Credential{ID: credID}

		err := repo.CreateUserWithCredential(ctx, u, cred)
		require.Error(t, err)

		_, getErr := repo.GetUserByEmail(u.Email)
		require.True(t, errors.Is(getErr, sql.ErrNoRows), "user must be rolled back, got %v", getErr)
	})
}
