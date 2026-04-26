package services

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/auth-service/internal/mocks"
	"pinz/backend/auth-service/internal/models"
	pb "pinz/backend/auth-service/pkg/proto"
)

func newDesiredPlaceSvc(t *testing.T, ctrl *gomock.Controller) (*AuthService, *mocks.MockDesiredPlaceRepositoryInterface, *mocks.MockS3Uploader) {
	t.Helper()
	dpRepo := mocks.NewMockDesiredPlaceRepositoryInterface(ctrl)
	s3 := mocks.NewMockS3Uploader(ctrl)
	svc := NewAuthService(nil, nil, nil, dpRepo, validator.New(), nil, s3)
	return svc, dpRepo, s3
}

func TestCreateDesiredPlace_Validation(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, _, _ := newDesiredPlaceSvc(t, ctrl)
	ctx := context.Background()

	cases := map[string]*pb.CreateDesiredPlaceRequest{
		"missing_user_id":   {Name: "n", Description: "d"},
		"empty_name":        {UserId: uuid.NewString(), Name: "  ", Description: "d"},
		"empty_description": {UserId: uuid.NewString(), Name: "n", Description: ""},
		"name_too_long":     {UserId: uuid.NewString(), Name: strings.Repeat("a", 201), Description: "d"},
		"description_too_long": {
			UserId: uuid.NewString(), Name: "n", Description: strings.Repeat("d", 1001),
		},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.CreateDesiredPlace(ctx, req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, codes.InvalidArgument, st.Code())
		})
	}
}

func TestCreateDesiredPlace_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, repo, _ := newDesiredPlaceSvc(t, ctrl)
	ctx := context.Background()

	userID := uuid.NewString()
	repo.EXPECT().Create(gomock.Any()).DoAndReturn(func(p *models.DesiredPlace) (*models.DesiredPlace, error) {
		require.Equal(t, userID, p.UserID)
		require.Equal(t, "Eiffel Tower", p.Name)
		require.Equal(t, "Want to visit", p.Description)
		require.Equal(t, "", p.ImageURL)
		return p, nil
	})

	resp, err := svc.CreateDesiredPlace(ctx, &pb.CreateDesiredPlaceRequest{
		UserId:      userID,
		Name:        "Eiffel Tower",
		Description: "Want to visit",
	})
	require.NoError(t, err)
	require.Equal(t, "Eiffel Tower", resp.GetPlace().GetName())
	require.Equal(t, "", resp.GetPlace().GetImageUrl())
}

func TestUpdateDesiredPlace_OwnershipMismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, repo, _ := newDesiredPlaceSvc(t, ctrl)
	ctx := context.Background()

	placeID := uuid.NewString()
	repo.EXPECT().GetByID(placeID).Return(&models.DesiredPlace{
		ID:          placeID,
		UserID:      uuid.NewString(), // другой owner
		Name:        "n",
		Description: "d",
	}, nil)

	_, err := svc.UpdateDesiredPlace(ctx, &pb.UpdateDesiredPlaceRequest{
		UserId:      uuid.NewString(),
		PlaceId:     placeID,
		Name:        "new",
		Description: "new",
	})
	require.Error(t, err)
	st, _ := status.FromError(err)
	require.Equal(t, codes.NotFound, st.Code(), "не палить чужие записи")
}

func TestUpdateDesiredPlace_ImageReplaceCleanupOldKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, repo, s3 := newDesiredPlaceSvc(t, ctrl)
	ctx := context.Background()

	userID := uuid.NewString()
	placeID := uuid.NewString()
	oldKey := "desired-places/x/old.jpg"
	newKey := "desired-places/x/new.jpg"

	gomock.InOrder(
		repo.EXPECT().GetByID(placeID).Return(&models.DesiredPlace{
			ID:          placeID,
			UserID:      userID,
			Name:        "old",
			Description: "old",
			ImageURL:    oldKey,
		}, nil),
		repo.EXPECT().Update(placeID, "new", "new", newKey).Return(&models.DesiredPlace{
			ID:          placeID,
			UserID:      userID,
			Name:        "new",
			Description: "new",
			ImageURL:    newKey,
		}, nil),
		s3.EXPECT().DeleteObject(gomock.Any(), oldKey).Return(nil),
	)
	s3.EXPECT().ReadURL(gomock.Any(), newKey).Return("https://s3/get/new", nil)

	_, err := svc.UpdateDesiredPlace(ctx, &pb.UpdateDesiredPlaceRequest{
		UserId:      userID,
		PlaceId:     placeID,
		Name:        "new",
		Description: "new",
		SetImageKey: true,
		S3Key:       newKey,
	})
	require.NoError(t, err)
}

func TestUpdateDesiredPlace_NoImageChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, repo, s3 := newDesiredPlaceSvc(t, ctrl)
	ctx := context.Background()

	userID := uuid.NewString()
	placeID := uuid.NewString()
	keepKey := "desired-places/x/keep.jpg"

	gomock.InOrder(
		repo.EXPECT().GetByID(placeID).Return(&models.DesiredPlace{
			ID: placeID, UserID: userID, Name: "old", Description: "old",
			ImageURL: keepKey,
		}, nil),
		repo.EXPECT().UpdateContent(placeID, "new", "new").Return(&models.DesiredPlace{
			ID: placeID, UserID: userID, Name: "new", Description: "new",
			ImageURL: keepKey,
		}, nil),
	)
	s3.EXPECT().ReadURL(gomock.Any(), keepKey).Return("https://s3/get/keep", nil)

	_, err := svc.UpdateDesiredPlace(ctx, &pb.UpdateDesiredPlaceRequest{
		UserId:      userID,
		PlaceId:     placeID,
		Name:        "new",
		Description: "new",
		// SetImageKey: false → image_url не трогается, S3 не вызывается
	})
	require.NoError(t, err)
}

func TestDeleteDesiredPlace_DeletesS3Object(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, repo, s3 := newDesiredPlaceSvc(t, ctrl)
	ctx := context.Background()

	userID := uuid.NewString()
	placeID := uuid.NewString()
	key := "desired-places/x/p.jpg"

	gomock.InOrder(
		repo.EXPECT().GetByID(placeID).Return(&models.DesiredPlace{
			ID: placeID, UserID: userID, ImageURL: key,
		}, nil),
		repo.EXPECT().Delete(placeID).Return(nil),
		s3.EXPECT().DeleteObject(gomock.Any(), key).Return(nil),
	)

	resp, err := svc.DeleteDesiredPlace(ctx, &pb.DeleteDesiredPlaceRequest{
		UserId: userID, PlaceId: placeID,
	})
	require.NoError(t, err)
	require.True(t, resp.GetSuccess())
}

func TestDeleteDesiredPlace_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, repo, _ := newDesiredPlaceSvc(t, ctrl)
	ctx := context.Background()

	placeID := uuid.NewString()
	repo.EXPECT().GetByID(placeID).Return(nil, sql.ErrNoRows)

	_, err := svc.DeleteDesiredPlace(ctx, &pb.DeleteDesiredPlaceRequest{
		UserId:  uuid.NewString(),
		PlaceId: placeID,
	})
	require.Error(t, err)
	st, _ := status.FromError(err)
	require.Equal(t, codes.NotFound, st.Code())
}

func TestRequestDesiredPlaceImageUpload_Validation(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, _, _ := newDesiredPlaceSvc(t, ctrl)
	ctx := context.Background()

	cases := map[string]struct {
		req  *pb.RequestDesiredPlaceImageUploadRequest
		code codes.Code
	}{
		"missing_user_id": {req: &pb.RequestDesiredPlaceImageUploadRequest{Filename: "x.jpg"}, code: codes.InvalidArgument},
		"missing_filename": {
			req: &pb.RequestDesiredPlaceImageUploadRequest{UserId: uuid.NewString()}, code: codes.InvalidArgument,
		},
		"unsupported_ext": {
			req:  &pb.RequestDesiredPlaceImageUploadRequest{UserId: uuid.NewString(), Filename: "doc.pdf"},
			code: codes.InvalidArgument,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.RequestDesiredPlaceImageUpload(ctx, tc.req)
			require.Error(t, err)
			st, _ := status.FromError(err)
			require.Equal(t, tc.code, st.Code())
		})
	}
}

func TestRequestDesiredPlaceImageUpload_Presign(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, _, s3 := newDesiredPlaceSvc(t, ctrl)
	ctx := context.Background()

	userID := uuid.NewString()
	s3.EXPECT().PresignedUploadURL(gomock.Any(), gomock.Any(), "image/jpeg").DoAndReturn(
		func(_ context.Context, key, _ string) (string, error) {
			require.True(t, strings.HasPrefix(key, "desired-places/"+userID+"/"))
			require.True(t, strings.HasSuffix(key, ".jpg"))
			return "https://s3/upload", nil
		},
	)

	resp, err := svc.RequestDesiredPlaceImageUpload(ctx, &pb.RequestDesiredPlaceImageUploadRequest{
		UserId: userID, Filename: "photo.jpg", ContentType: "image/jpeg",
	})
	require.NoError(t, err)
	require.Equal(t, "https://s3/upload", resp.GetUploadUrl())
	require.NotEmpty(t, resp.GetS3Key())
}

func TestDeleteDesiredPlaceImage_ClearsAndDeletes(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, repo, s3 := newDesiredPlaceSvc(t, ctrl)
	ctx := context.Background()

	userID := uuid.NewString()
	placeID := uuid.NewString()
	key := "desired-places/x/p.jpg"

	gomock.InOrder(
		repo.EXPECT().GetByID(placeID).Return(&models.DesiredPlace{
			ID: placeID, UserID: userID, ImageURL: key,
		}, nil),
		repo.EXPECT().ClearImage(placeID).Return(&models.DesiredPlace{
			ID: placeID, UserID: userID, ImageURL: "",
		}, nil),
		s3.EXPECT().DeleteObject(gomock.Any(), key).Return(nil),
	)

	resp, err := svc.DeleteDesiredPlaceImage(ctx, &pb.DeleteDesiredPlaceImageRequest{
		UserId: userID, PlaceId: placeID,
	})
	require.NoError(t, err)
	require.Equal(t, "", resp.GetPlace().GetImageUrl())
}

func TestGetPublicUserProfile_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	dpRepo := mocks.NewMockDesiredPlaceRepositoryInterface(ctrl)
	svc := NewAuthService(userRepo, nil, nil, dpRepo, validator.New(), nil, nil)
	ctx := context.Background()

	userID := uuid.NewString()
	userRepo.EXPECT().GetUserByID(userID).Return(nil, sql.ErrNoRows)

	_, err := svc.GetPublicUserProfile(ctx, &pb.GetPublicUserProfileRequest{UserId: userID})
	require.Error(t, err)
	st, _ := status.FromError(err)
	require.Equal(t, codes.NotFound, st.Code())
}

func TestGetPublicUserProfile_NoEmailLeak(t *testing.T) {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepositoryInterface(ctrl)
	dpRepo := mocks.NewMockDesiredPlaceRepositoryInterface(ctrl)
	svc := NewAuthService(userRepo, nil, nil, dpRepo, validator.New(), nil, nil)
	ctx := context.Background()

	userID := uuid.NewString()
	userRepo.EXPECT().GetUserByID(userID).Return(&models.User{
		ID:       userID,
		Email:    "secret@example.com",
		Username: "alice",
	}, nil)
	dpRepo.EXPECT().ListByUserID(userID).Return([]*models.DesiredPlace{}, nil)

	resp, err := svc.GetPublicUserProfile(ctx, &pb.GetPublicUserProfileRequest{UserId: userID})
	require.NoError(t, err)
	require.NotContains(t, resp.String(), "secret@example.com")
	require.Equal(t, "alice", resp.GetProfile().GetUsername())
}

// Sanity: validation error wrapping остаётся правильным
func TestUpdateDesiredPlace_ValidationBeforeRepo(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, repo, _ := newDesiredPlaceSvc(t, ctrl)
	ctx := context.Background()

	// repo не должен дёргаться — валидация падает раньше
	repo.EXPECT().GetByID(gomock.Any()).Times(0)

	_, err := svc.UpdateDesiredPlace(ctx, &pb.UpdateDesiredPlaceRequest{
		UserId: uuid.NewString(), PlaceId: uuid.NewString(),
		Name: "", Description: "d",
	})
	require.Error(t, err)
	st, _ := status.FromError(err)
	require.Equal(t, codes.InvalidArgument, st.Code())
}

