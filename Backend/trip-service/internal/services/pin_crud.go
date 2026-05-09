package services

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
	"pinz/backend/trip-service/internal/server"
	pb "pinz/backend/trip-service/pkg/proto"
)

// коды проблем для review (ТЗ 4.7.3-4.7.4).
const (
	pinIssueMissingCoordinates = "MISSING_COORDINATES"
	pinIssueMissingDates       = "MISSING_DATES"
)

// GetPin — все поля пина. Доступ: participant трипа или favourite-юзер.
// Скрытый для caller'а пин (ТЗ 4.5.2) → NotFound.
func (s *TripService) GetPin(ctx context.Context, req *pb.GetPinRequest) (*pb.GetPinResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	pinID := req.GetPinId()
	if tripID == "" || pinID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and pin_id are required")
	}
	if _, err := s.assertCanReadPin(ctx, tripID, pinID, userID); err != nil {
		return nil, err
	}
	pin, err := s.pinRepo.GetByID(pinID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "pin not found")
		}
		return nil, status.Error(codes.Internal, "failed to get pin")
	}
	if pin.TripID != tripID {
		return nil, status.Error(codes.NotFound, "pin not found")
	}
	mediaList, err := s.mediaRepo.ListByPinID(pinID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list pin media")
	}
	tags, _ := s.tagRepo.GetByPinID(pinID)
	if tags == nil {
		tags = []string{}
	}
	return &pb.GetPinResponse{Pin: s.pinWithMediaToProto(ctx, pin, mediaList, tags)}, nil
}

// assertCanReadPin — guard на GetPin: participant или favourite-юзер.
// Скрытый для caller'а пин → NotFound.
func (s *TripService) assertCanReadPin(ctx context.Context, tripID, pinID, userID string) (bool, error) {
	if s.pinHiddenRepo != nil {
		hidden, err := s.pinHiddenRepo.IsHidden(pinID, userID)
		if err == nil && hidden {
			return false, status.Error(codes.NotFound, "pin not found")
		}
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil {
		return false, status.Error(codes.Internal, "failed to check participant")
	}
	if participant {
		return true, nil
	}
	if s.favouriteRepo != nil {
		hasFav, err := s.favouriteRepo.HasFavourite(userID, tripID)
		if err == nil && hasFav {
			return true, nil
		}
	}
	_ = ctx
	return false, status.Error(codes.PermissionDenied, "not a participant")
}

// UpdatePin применяет правки полей пина на READY-трипе (ТЗ 4.2.1, 4.2.4-4.2.9).
// tags — replace-all только при tags_set=true. Смена координат → geo-reverse upsert.
func (s *TripService) UpdatePin(ctx context.Context, req *pb.UpdatePinRequest) (*pb.UpdatePinResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	pinID := req.GetPinId()
	if tripID == "" || pinID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and pin_id are required")
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check participant")
	}
	if !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	if trip.Status != models.TripStatusReady {
		return nil, errWrongStatus(models.TripStatusReady, trip.Status)
	}
	pin, err := s.pinRepo.GetByID(pinID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "pin not found")
		}
		return nil, status.Error(codes.Internal, "failed to get pin")
	}
	if pin.TripID != tripID {
		return nil, status.Error(codes.NotFound, "pin not found")
	}

	coordsChanged := false
	if req.Name != nil {
		name := req.GetName()
		if len(name) == 0 {
			return nil, status.Error(codes.InvalidArgument, "name must not be empty")
		}
		if len(name) > MaxNameLength {
			return nil, status.Errorf(codes.InvalidArgument, "name must be at most %d characters", MaxNameLength)
		}
		pin.Name = name
	}
	if req.Description != nil {
		desc := req.GetDescription()
		if len(desc) > MaxDescriptionLength {
			return nil, status.Errorf(codes.InvalidArgument, "description must be at most %d characters", MaxDescriptionLength)
		}
		pin.Description = desc
	}
	if req.Category != nil {
		pin.Category = ValidatePinCategory(req.GetCategory())
	}
	if req.Latitude != nil || req.Longitude != nil {
		if req.Latitude == nil || req.Longitude == nil {
			return nil, status.Error(codes.InvalidArgument, "latitude and longitude must be provided together")
		}
		lat := req.GetLatitude()
		lon := req.GetLongitude()
		if lat < -90 || lat > 90 {
			return nil, status.Error(codes.InvalidArgument, "latitude must be in [-90, 90]")
		}
		if lon < -180 || lon > 180 {
			return nil, status.Error(codes.InvalidArgument, "longitude must be in [-180, 180]")
		}
		pin.Latitude = &lat
		pin.Longitude = &lon
		coordsChanged = true
	}
	if req.StartTimeUnix != nil {
		t := time.Unix(req.GetStartTimeUnix(), 0)
		pin.StartTime = &t
	}
	if req.EndTimeUnix != nil {
		t := time.Unix(req.GetEndTimeUnix(), 0)
		pin.EndTime = &t
	}
	if pin.StartTime != nil && pin.EndTime != nil && pin.EndTime.Before(*pin.StartTime) {
		return nil, status.Error(codes.InvalidArgument, "end_time must be greater than or equal to start_time")
	}
	if req.GetTagsSet() {
		if err := validateTags(req.GetTags()); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	if err := s.pinRepo.Update(pin); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "pin not found")
		}
		return nil, status.Error(codes.Internal, "failed to update pin")
	}
	if req.GetTagsSet() {
		if err := s.tagRepo.SetForPin(tripID, pinID, req.GetTags()); err != nil {
			return nil, status.Error(codes.Internal, "failed to update pin tags")
		}
	}
	if coordsChanged && pin.Latitude != nil && pin.Longitude != nil && s.eventRepo != nil {
		_ = s.eventRepo.PublishGeoRequest(ctx, tripID, []repositories.GeoRequestPin{{
			PinID:     pinID,
			Latitude:  *pin.Latitude,
			Longitude: *pin.Longitude,
		}})
	}
	mediaList, err := s.mediaRepo.ListByPinID(pinID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list pin media")
	}
	tags, _ := s.tagRepo.GetByPinID(pinID)
	if tags == nil {
		tags = []string{}
	}
	return &pb.UpdatePinResponse{Pin: s.pinWithMediaToProto(ctx, pin, mediaList, tags)}, nil
}

// DeletePin: full delete если трип ни у кого не в favourites, иначе soft-delete-for-self
// через pin_hidden_by_user (ТЗ 4.5.1/4.5.2). Запрещён при активной pin_media_addition_session.
func (s *TripService) DeletePin(ctx context.Context, req *pb.DeletePinRequest) (*pb.DeletePinResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	pinID := req.GetPinId()
	if tripID == "" || pinID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and pin_id are required")
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check participant")
	}
	if !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	pin, err := s.pinRepo.GetByID(pinID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "pin not found")
		}
		return nil, status.Error(codes.Internal, "failed to get pin")
	}
	if pin.TripID != tripID {
		return nil, status.Error(codes.NotFound, "pin not found")
	}
	if s.pinUploadSessionRepo != nil {
		if _, err := s.pinUploadSessionRepo.GetActiveAdditionForPin(ctx, pinID); err == nil {
			return nil, status.Error(codes.FailedPrecondition, "pin has an active media addition session; cancel it first")
		} else if !errors.Is(err, repositories.ErrPinUploadSessionNotFound) {
			return nil, status.Error(codes.Internal, "failed to check active pin media session")
		}
	}
	inOthersFav := false
	if s.favouriteRepo != nil {
		v, err := s.favouriteRepo.HasFavouritesByOtherUsers(tripID, userID)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to check favourites")
		}
		inOthersFav = v
	}
	if inOthersFav {
		// soft-delete-for-self (ТЗ 4.5.2)
		if s.pinHiddenRepo == nil {
			return nil, status.Error(codes.Internal, "pin hidden repository not configured")
		}
		if err := s.pinHiddenRepo.HidePinForUser(pinID, userID); err != nil {
			return nil, status.Error(codes.Internal, "failed to hide pin")
		}
		return &pb.DeletePinResponse{DeletionMode: "soft_for_user"}, nil
	}
	// media.pin_id = ON DELETE SET NULL, поэтому media каскадим явно (s3 cleanup).
	s3Keys, err := s.mediaRepo.DeleteByPinID(pinID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to delete pin media")
	}
	if s.mediaURLs != nil {
		for _, key := range s3Keys {
			if err := s.mediaURLs.DeleteObject(ctx, key); err != nil {
				slog.WarnContext(ctx, "DeletePin: s3 delete failed (best-effort)", "pin_id", pinID, "key", key, "err", err)
			}
		}
	}
	if err := s.tagRepo.DeleteForPin(pinID); err != nil {
		slog.WarnContext(ctx, "DeletePin: tag delete failed", "pin_id", pinID, "err", err)
	}
	if err := s.pinRepo.Delete(pinID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "pin not found")
		}
		return nil, status.Error(codes.Internal, "failed to delete pin")
	}
	return &pb.DeletePinResponse{DeletionMode: "full"}, nil
}

// =============================================================================
// RemoveMediaFromPin — sessionless удаление одного медиа (ТЗ 4.2.3)
// =============================================================================

// RemoveMediaFromPin удаляет одно медиа из пина с пересчётом агрегатов.
// Защита: пин не может остаться пустым (ТЗ 2.2.9 — set of media обязателен).
func (s *TripService) RemoveMediaFromPin(ctx context.Context, req *pb.RemoveMediaFromPinRequest) (*pb.RemoveMediaFromPinResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID, pinID, mediaID := req.GetTripId(), req.GetPinId(), req.GetMediaId()
	if tripID == "" || pinID == "" || mediaID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id, pin_id, media_id are required")
	}
	if err := s.assertParticipantAndPinReady(ctx, tripID, pinID, userID); err != nil {
		return nil, err
	}
	m, err := s.mediaRepo.GetByID(mediaID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "media not found")
		}
		return nil, status.Error(codes.Internal, "failed to get media")
	}
	if m.TripID != tripID || m.PinID == nil || *m.PinID != pinID {
		return nil, status.Error(codes.NotFound, "media not found")
	}
	// Пин не должен остаться без медиа.
	pinMedia, err := s.mediaRepo.ListByPinID(pinID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list pin media")
	}
	if len(pinMedia) <= 1 {
		return nil, status.Error(codes.FailedPrecondition, "pin must contain at least one media; delete pin instead")
	}
	if err := s.mediaRepo.DeleteByIDs([]string{mediaID}); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete media")
	}
	if s.mediaURLs != nil && m.S3Key != "" {
		_ = s.mediaURLs.DeleteObject(ctx, m.S3Key)
	}
	if err := s.pinRepo.IncMediaCount(pinID, -1); err != nil {
		slog.WarnContext(ctx, "RemoveMediaFromPin: media_count update failed", "pin_id", pinID, "err", err)
	}
	updatePinTimesAndLocation(s.pinRepo, s.mediaRepo, pinID)
	pin, err := s.pinRepo.GetByID(pinID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get pin")
	}
	mediaList, _ := s.mediaRepo.ListByPinID(pinID)
	tags, _ := s.tagRepo.GetByPinID(pinID)
	if tags == nil {
		tags = []string{}
	}
	return &pb.RemoveMediaFromPinResponse{Pin: s.pinWithMediaToProto(ctx, pin, mediaList, tags)}, nil
}

// =============================================================================
// helpers
// =============================================================================

// assertParticipantAndPinReady — общий guard: caller — participant, trip в READY,
// pin принадлежит trip. PermissionDenied/NotFound/FailedPrecondition.
func (s *TripService) assertParticipantAndPinReady(ctx context.Context, tripID, pinID, userID string) error {
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil {
		return status.Error(codes.Internal, "failed to check participant")
	}
	if !participant {
		return status.Error(codes.PermissionDenied, "not a participant")
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return status.Error(codes.NotFound, "trip not found")
		}
		return status.Error(codes.Internal, "failed to get trip")
	}
	if trip.Status != models.TripStatusReady {
		return errWrongStatus(models.TripStatusReady, trip.Status)
	}
	pin, err := s.pinRepo.GetByID(pinID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return status.Error(codes.NotFound, "pin not found")
		}
		return status.Error(codes.Internal, "failed to get pin")
	}
	if pin.TripID != tripID {
		return status.Error(codes.NotFound, "pin not found")
	}
	if s.pinHiddenRepo != nil {
		if hidden, herr := s.pinHiddenRepo.IsHidden(pinID, userID); herr == nil && hidden {
			return status.Error(codes.NotFound, "pin not found")
		}
	}
	_ = ctx
	return nil
}

