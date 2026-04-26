package services

import (
	"context"
	"testing"

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

// TestDesiredPlaces_Integration проверяет полный CRUD-цикл и GetPublicUserProfile
// против реальной БД (testcontainers Postgres). S3 не подключаем — image_url
// в ответах будет пустым (PresignedUploadURL не вызывается).
func TestDesiredPlaces_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	testinfra.WithAuthPostgres(t)

	sqlDB, err := db.InitDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	userRepo := repositories.NewUserRepository(sqlDB)
	dpRepo := repositories.NewDesiredPlaceRepository(sqlDB)

	owner := &models.User{
		ID:       uuid.NewString(),
		Email:    "owner@example.com",
		Username: "owner",
	}
	require.NoError(t, userRepo.CreateUser(owner))
	stranger := &models.User{
		ID:       uuid.NewString(),
		Email:    "stranger@example.com",
		Username: "stranger",
	}
	require.NoError(t, userRepo.CreateUser(stranger))

	svc := NewAuthService(userRepo, nil, nil, dpRepo, validator.New(), nil, nil)
	ctx := context.Background()

	// 1. Create
	create1, err := svc.CreateDesiredPlace(ctx, &pb.CreateDesiredPlaceRequest{
		UserId:      owner.ID,
		Name:        "Eiffel Tower",
		Description: "Want to visit",
	})
	require.NoError(t, err)
	require.NotEmpty(t, create1.GetPlace().GetId())

	create2, err := svc.CreateDesiredPlace(ctx, &pb.CreateDesiredPlaceRequest{
		UserId:      owner.ID,
		Name:        "Colosseum",
		Description: "Roman arena",
	})
	require.NoError(t, err)

	// 2. List — DESC by created_at, поэтому второй (более новый) идёт первым.
	list, err := svc.ListDesiredPlaces(ctx, &pb.ListDesiredPlacesRequest{UserId: owner.ID})
	require.NoError(t, err)
	require.Len(t, list.GetPlaces(), 2)
	require.Equal(t, create2.GetPlace().GetId(), list.GetPlaces()[0].GetId())

	// 3. Update content (без image)
	upd, err := svc.UpdateDesiredPlace(ctx, &pb.UpdateDesiredPlaceRequest{
		UserId:      owner.ID,
		PlaceId:     create1.GetPlace().GetId(),
		Name:        "Eiffel — updated",
		Description: "Edited",
	})
	require.NoError(t, err)
	require.Equal(t, "Eiffel — updated", upd.GetPlace().GetName())

	// 4. Stranger не может обновлять/удалять чужое — отдаём 404 (а не 403),
	// чтобы не палить существование.
	_, err = svc.UpdateDesiredPlace(ctx, &pb.UpdateDesiredPlaceRequest{
		UserId:      stranger.ID,
		PlaceId:     create1.GetPlace().GetId(),
		Name:        "evil",
		Description: "evil",
	})
	require.Error(t, err)
	st, _ := status.FromError(err)
	require.Equal(t, codes.NotFound, st.Code())

	_, err = svc.DeleteDesiredPlace(ctx, &pb.DeleteDesiredPlaceRequest{
		UserId:  stranger.ID,
		PlaceId: create1.GetPlace().GetId(),
	})
	require.Error(t, err)

	// 5. GetPublicUserProfile — возвращает publicный профиль и весь wishlist.
	pub, err := svc.GetPublicUserProfile(ctx, &pb.GetPublicUserProfileRequest{UserId: owner.ID})
	require.NoError(t, err)
	require.Equal(t, owner.ID, pub.GetProfile().GetUserId())
	require.Equal(t, "owner", pub.GetProfile().GetUsername())
	require.NotZero(t, pub.GetProfile().GetCreatedAtUnix())
	require.Len(t, pub.GetDesiredPlaces(), 2)

	// 6. Delete — удаляем одно, второе остаётся.
	delResp, err := svc.DeleteDesiredPlace(ctx, &pb.DeleteDesiredPlaceRequest{
		UserId:  owner.ID,
		PlaceId: create1.GetPlace().GetId(),
	})
	require.NoError(t, err)
	require.True(t, delResp.GetSuccess())

	listAfter, err := svc.ListDesiredPlaces(ctx, &pb.ListDesiredPlacesRequest{UserId: owner.ID})
	require.NoError(t, err)
	require.Len(t, listAfter.GetPlaces(), 1)
	require.Equal(t, create2.GetPlace().GetId(), listAfter.GetPlaces()[0].GetId())

	// 7. Cascade ON DELETE: удаление пользователя уносит его желаемые места.
	require.NoError(t, userRepo.DeleteUser(owner.ID))
	listGone, err := svc.ListDesiredPlaces(ctx, &pb.ListDesiredPlacesRequest{UserId: owner.ID})
	require.NoError(t, err)
	require.Empty(t, listGone.GetPlaces())
}
