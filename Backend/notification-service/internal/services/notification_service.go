package services

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/notification-service/internal/repositories"
	pb "pinz/backend/notification-service/pkg/proto"
)

// NotificationService — реализация pb.NotificationServiceServer. Бизнес-логика
// по device_tokens. Сами пуши отправляет worker/scheduler, не этот сервис.
type NotificationService struct {
	pb.UnimplementedNotificationServiceServer
	tokens repositories.DeviceTokensRepositoryInterface
}

func NewNotificationService(tokens repositories.DeviceTokensRepositoryInterface) *NotificationService {
	return &NotificationService{tokens: tokens}
}

func (s *NotificationService) RegisterDeviceToken(ctx context.Context, req *pb.RegisterDeviceTokenRequest) (*pb.RegisterDeviceTokenResponse, error) {
	userID := strings.TrimSpace(req.GetUserId())
	apnsToken := strings.TrimSpace(req.GetApnsToken())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if apnsToken == "" {
		return nil, status.Error(codes.InvalidArgument, "apns_token is required")
	}
	id, err := s.tokens.Upsert(ctx, userID, apnsToken)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, status.Error(codes.Canceled, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "upsert device_token: %v", err)
	}
	return &pb.RegisterDeviceTokenResponse{TokenId: id}, nil
}

func (s *NotificationService) UnregisterDeviceToken(ctx context.Context, req *pb.UnregisterDeviceTokenRequest) (*pb.UnregisterDeviceTokenResponse, error) {
	userID := strings.TrimSpace(req.GetUserId())
	apnsToken := strings.TrimSpace(req.GetApnsToken())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if apnsToken == "" {
		return nil, status.Error(codes.InvalidArgument, "apns_token is required")
	}
	n, err := s.tokens.Delete(ctx, userID, apnsToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete device_token: %v", err)
	}
	return &pb.UnregisterDeviceTokenResponse{Success: n > 0}, nil
}
