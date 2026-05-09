package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/auth-service/internal/models"
	pb "pinz/backend/auth-service/pkg/proto"
)

const (
	desiredPlaceNameMaxLen        = 200
	desiredPlaceDescriptionMaxLen = 1000
)

var allowedDesiredPlaceImageExt = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".heic": {},
}

func (s *AuthService) ListDesiredPlaces(ctx context.Context, req *pb.ListDesiredPlacesRequest) (*pb.ListDesiredPlacesResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.ListDesiredPlaces")
	defer span.End()

	userID := req.GetUserId()
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	span.SetAttributes(attribute.String("user_id", userID))

	places, err := s.listDesiredPlacesProto(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &pb.ListDesiredPlacesResponse{Places: places}, nil
}

func (s *AuthService) CreateDesiredPlace(ctx context.Context, req *pb.CreateDesiredPlaceRequest) (*pb.CreateDesiredPlaceResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.CreateDesiredPlace")
	defer span.End()

	userID := req.GetUserId()
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	name := strings.TrimSpace(req.GetName())
	description := strings.TrimSpace(req.GetDescription())
	if err := validateDesiredPlaceContent(name, description); err != nil {
		return nil, err
	}

	place := &models.DesiredPlace{
		ID:          uuid.NewString(),
		UserID:      userID,
		Name:        name,
		Description: description,
		ImageURL:    req.GetS3Key(),
	}
	created, err := s.desiredPlaceRepo.Create(place)
	if err != nil {
		slog.ErrorContext(ctx, "CreateDesiredPlace: repo create", "error", err)
		return nil, status.Error(codes.Internal, "failed to create desired place")
	}
	return &pb.CreateDesiredPlaceResponse{Place: s.desiredPlaceToProto(ctx, created)}, nil
}

func (s *AuthService) UpdateDesiredPlace(ctx context.Context, req *pb.UpdateDesiredPlaceRequest) (*pb.UpdateDesiredPlaceResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.UpdateDesiredPlace")
	defer span.End()

	userID := req.GetUserId()
	placeID := req.GetPlaceId()
	if userID == "" || placeID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and place_id are required")
	}
	name := strings.TrimSpace(req.GetName())
	description := strings.TrimSpace(req.GetDescription())
	if err := validateDesiredPlaceContent(name, description); err != nil {
		return nil, err
	}

	existing, err := s.desiredPlaceRepo.GetByID(placeID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "desired place not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "UpdateDesiredPlace: repo get", "error", err)
		return nil, status.Error(codes.Internal, "failed to load desired place")
	}
	if existing.UserID != userID {
		return nil, status.Error(codes.NotFound, "desired place not found")
	}

	var updated *models.DesiredPlace
	if req.GetSetImageKey() {
		newKey := req.GetS3Key()
		updated, err = s.desiredPlaceRepo.Update(placeID, name, description, newKey)
		if err != nil {
			slog.ErrorContext(ctx, "UpdateDesiredPlace: repo update", "error", err)
			return nil, status.Error(codes.Internal, "failed to update desired place")
		}
		if existing.ImageURL != "" && existing.ImageURL != newKey && s.s3 != nil {
			if derr := s.s3.DeleteObject(ctx, existing.ImageURL); derr != nil {
				slog.ErrorContext(ctx, "UpdateDesiredPlace: delete old image (best-effort)", "key", existing.ImageURL, "error", derr)
			}
		}
	} else {
		updated, err = s.desiredPlaceRepo.UpdateContent(placeID, name, description)
		if err != nil {
			slog.ErrorContext(ctx, "UpdateDesiredPlace: repo update content", "error", err)
			return nil, status.Error(codes.Internal, "failed to update desired place")
		}
	}
	return &pb.UpdateDesiredPlaceResponse{Place: s.desiredPlaceToProto(ctx, updated)}, nil
}

func (s *AuthService) DeleteDesiredPlace(ctx context.Context, req *pb.DeleteDesiredPlaceRequest) (*pb.DeleteDesiredPlaceResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.DeleteDesiredPlace")
	defer span.End()

	userID := req.GetUserId()
	placeID := req.GetPlaceId()
	if userID == "" || placeID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and place_id are required")
	}

	existing, err := s.desiredPlaceRepo.GetByID(placeID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "desired place not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "DeleteDesiredPlace: repo get", "error", err)
		return nil, status.Error(codes.Internal, "failed to load desired place")
	}
	if existing.UserID != userID {
		return nil, status.Error(codes.NotFound, "desired place not found")
	}

	if err := s.desiredPlaceRepo.Delete(placeID); err != nil {
		slog.ErrorContext(ctx, "DeleteDesiredPlace: repo delete", "error", err)
		return nil, status.Error(codes.Internal, "failed to delete desired place")
	}

	if existing.ImageURL != "" && s.s3 != nil {
		if derr := s.s3.DeleteObject(ctx, existing.ImageURL); derr != nil {
			slog.ErrorContext(ctx, "DeleteDesiredPlace: s3 delete (best-effort)", "key", existing.ImageURL, "error", derr)
		}
	}
	return &pb.DeleteDesiredPlaceResponse{Success: true}, nil
}

func (s *AuthService) RequestDesiredPlaceImageUpload(ctx context.Context, req *pb.RequestDesiredPlaceImageUploadRequest) (*pb.RequestDesiredPlaceImageUploadResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.RequestDesiredPlaceImageUpload")
	defer span.End()

	userID := req.GetUserId()
	filename := req.GetFilename()
	if userID == "" || filename == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and filename are required")
	}
	if s.s3 == nil {
		return nil, status.Error(codes.Unavailable, "image upload is not configured")
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".jpg"
	}
	if _, ok := allowedDesiredPlaceImageExt[ext]; !ok {
		return nil, status.Error(codes.InvalidArgument, "image must be .jpg, .jpeg, .png or .heic")
	}

	s3Key := fmt.Sprintf("desired-places/%s/%s%s", userID, uuid.NewString(), ext)
	uploadURL, err := s.s3.PresignedUploadURL(ctx, s3Key, req.GetContentType())
	if err != nil {
		slog.ErrorContext(ctx, "RequestDesiredPlaceImageUpload: presign", "error", err)
		return nil, status.Error(codes.Internal, "failed to generate upload URL")
	}
	return &pb.RequestDesiredPlaceImageUploadResponse{UploadUrl: uploadURL, S3Key: s3Key}, nil
}

func (s *AuthService) DeleteDesiredPlaceImage(ctx context.Context, req *pb.DeleteDesiredPlaceImageRequest) (*pb.DeleteDesiredPlaceImageResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.DeleteDesiredPlaceImage")
	defer span.End()

	userID := req.GetUserId()
	placeID := req.GetPlaceId()
	if userID == "" || placeID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and place_id are required")
	}

	existing, err := s.desiredPlaceRepo.GetByID(placeID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "desired place not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "DeleteDesiredPlaceImage: repo get", "error", err)
		return nil, status.Error(codes.Internal, "failed to load desired place")
	}
	if existing.UserID != userID {
		return nil, status.Error(codes.NotFound, "desired place not found")
	}

	updated, err := s.desiredPlaceRepo.ClearImage(placeID)
	if err != nil {
		slog.ErrorContext(ctx, "DeleteDesiredPlaceImage: clear", "error", err)
		return nil, status.Error(codes.Internal, "failed to clear image")
	}

	if existing.ImageURL != "" && s.s3 != nil {
		if derr := s.s3.DeleteObject(ctx, existing.ImageURL); derr != nil {
			slog.ErrorContext(ctx, "DeleteDesiredPlaceImage: s3 delete (best-effort)", "key", existing.ImageURL, "error", derr)
		}
	}
	return &pb.DeleteDesiredPlaceImageResponse{Place: s.desiredPlaceToProto(ctx, updated)}, nil
}

func (s *AuthService) listDesiredPlacesProto(ctx context.Context, userID string) ([]*pb.DesiredPlace, error) {
	if s.desiredPlaceRepo == nil {
		return nil, nil
	}
	rows, err := s.desiredPlaceRepo.ListByUserID(userID)
	if err != nil {
		slog.ErrorContext(ctx, "listDesiredPlacesProto: repo list", "user_id", userID, "error", err)
		return nil, status.Error(codes.Internal, "failed to list desired places")
	}
	out := make([]*pb.DesiredPlace, 0, len(rows))
	for _, p := range rows {
		out = append(out, s.desiredPlaceToProto(ctx, p))
	}
	return out, nil
}

func (s *AuthService) desiredPlaceToProto(ctx context.Context, p *models.DesiredPlace) *pb.DesiredPlace {
	imageURL := ""
	if p.ImageURL != "" && s.s3 != nil {
		if url, err := s.s3.ReadURL(ctx, p.ImageURL); err != nil {
			slog.WarnContext(ctx, "desiredPlaceToProto: presign image", "key", p.ImageURL, "error", err)
		} else {
			imageURL = url
		}
	}
	return &pb.DesiredPlace{
		Id:            p.ID,
		UserId:        p.UserID,
		Name:          p.Name,
		Description:   p.Description,
		ImageUrl:      imageURL,
		CreatedAtUnix: p.CreatedAt.Unix(),
	}
}

func validateDesiredPlaceContent(name, description string) error {
	if name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	if len([]rune(name)) > desiredPlaceNameMaxLen {
		return status.Errorf(codes.InvalidArgument, "name must be at most %d characters", desiredPlaceNameMaxLen)
	}
	if description == "" {
		return status.Error(codes.InvalidArgument, "description is required")
	}
	if len([]rune(description)) > desiredPlaceDescriptionMaxLen {
		return status.Errorf(codes.InvalidArgument, "description must be at most %d characters", desiredPlaceDescriptionMaxLen)
	}
	return nil
}
