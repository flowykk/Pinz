package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/notification-service/internal/mocks"
	pb "pinz/backend/notification-service/pkg/proto"
)

func TestRegisterDeviceToken_Validation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	svc := NewNotificationService(mocks.NewMockDeviceTokensRepositoryInterface(ctrl))

	cases := map[string]*pb.RegisterDeviceTokenRequest{
		"empty_user":  {ApnsToken: "t"},
		"empty_token": {UserId: "u"},
		"spaces_only": {UserId: "  ", ApnsToken: "  "},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.RegisterDeviceToken(context.Background(), req)
			require.Error(t, err)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestRegisterDeviceToken_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockDeviceTokensRepositoryInterface(ctrl)
	repo.EXPECT().Upsert(gomock.Any(), "u", "tok").Return("id-1", nil)

	svc := NewNotificationService(repo)
	resp, err := svc.RegisterDeviceToken(context.Background(), &pb.RegisterDeviceTokenRequest{UserId: "u", ApnsToken: "tok"})
	require.NoError(t, err)
	require.Equal(t, "id-1", resp.GetTokenId())
}

func TestRegisterDeviceToken_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockDeviceTokensRepositoryInterface(ctrl)
	repo.EXPECT().Upsert(gomock.Any(), "u", "tok").Return("", errors.New("boom"))

	svc := NewNotificationService(repo)
	_, err := svc.RegisterDeviceToken(context.Background(), &pb.RegisterDeviceTokenRequest{UserId: "u", ApnsToken: "tok"})
	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestUnregisterDeviceToken_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockDeviceTokensRepositoryInterface(ctrl)
	repo.EXPECT().Delete(gomock.Any(), "u", "tok").Return(int64(1), nil)

	svc := NewNotificationService(repo)
	resp, err := svc.UnregisterDeviceToken(context.Background(), &pb.UnregisterDeviceTokenRequest{UserId: "u", ApnsToken: "tok"})
	require.NoError(t, err)
	require.True(t, resp.GetSuccess())
}

func TestUnregisterDeviceToken_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockDeviceTokensRepositoryInterface(ctrl)
	repo.EXPECT().Delete(gomock.Any(), "u", "tok").Return(int64(0), nil)

	svc := NewNotificationService(repo)
	resp, err := svc.UnregisterDeviceToken(context.Background(), &pb.UnregisterDeviceTokenRequest{UserId: "u", ApnsToken: "tok"})
	require.NoError(t, err)
	require.False(t, resp.GetSuccess())
}

func TestUnregisterDeviceToken_Validation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	svc := NewNotificationService(mocks.NewMockDeviceTokensRepositoryInterface(ctrl))

	_, err := svc.UnregisterDeviceToken(context.Background(), &pb.UnregisterDeviceTokenRequest{ApnsToken: "t"})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}
