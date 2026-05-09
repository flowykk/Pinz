package services

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
	pb "pinz/backend/trip-service/pkg/proto"
)

// finalizePinUploadCreation: INSERT pin, attach media, tags, geo, PIN_ADDED.
func (s *TripService) finalizePinUploadCreation(
	ctx context.Context,
	trip *models.Trip,
	session *models.PinUploadSession,
	snap *pinUploadDraftSnapshot,
	remaining []*models.Media,
	req *pb.FinalizePinUploadRequest,
	userID string,
) (*pb.FinalizePinUploadResponse, error) {
	tripID := session.TripID
	sessionID := session.SessionID
	if len(remaining) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "pin must contain at least one media")
	}

	suggested := snap.Suggested
	if suggested == nil {
		suggested = &pinSuggestedFields{Category: PinCategoryDefault, Name: PinCategoryDefault}
	}

	name := suggested.Name
	if req.Name != nil && *req.Name != "" {
		name = *req.Name
	}
	if name == "" {
		name = "Pin"
	}
	if len(name) > MaxNameLength {
		return nil, status.Errorf(codes.InvalidArgument, "name must be at most %d characters", MaxNameLength)
	}

	description := ""
	if req.Description != nil {
		description = *req.Description
		if len(description) > MaxDescriptionLength {
			return nil, status.Errorf(codes.InvalidArgument, "description must be at most %d characters", MaxDescriptionLength)
		}
	}

	category := suggested.Category
	if req.Category != nil && *req.Category != "" {
		category = ValidatePinCategory(*req.Category)
	}
	if category == "" {
		category = PinCategoryDefault
	}

	var pinLat, pinLon *float64
	if req.Latitude != nil && req.Longitude != nil {
		lat := req.GetLatitude()
		lon := req.GetLongitude()
		if lat < -90 || lat > 90 {
			return nil, status.Error(codes.InvalidArgument, "latitude must be in [-90, 90]")
		}
		if lon < -180 || lon > 180 {
			return nil, status.Error(codes.InvalidArgument, "longitude must be in [-180, 180]")
		}
		pinLat = &lat
		pinLon = &lon
	} else if suggested.Latitude != nil && suggested.Longitude != nil {
		pinLat = suggested.Latitude
		pinLon = suggested.Longitude
	}

	var pinStart, pinEnd *time.Time
	if req.StartTimeUnix != nil {
		t := time.Unix(req.GetStartTimeUnix(), 0)
		pinStart = &t
	} else if suggested.StartTimeUnix != nil {
		t := time.Unix(*suggested.StartTimeUnix, 0)
		pinStart = &t
	}
	if req.EndTimeUnix != nil {
		t := time.Unix(req.GetEndTimeUnix(), 0)
		pinEnd = &t
	} else if suggested.EndTimeUnix != nil {
		t := time.Unix(*suggested.EndTimeUnix, 0)
		pinEnd = &t
	}
	if pinStart != nil && pinEnd != nil && pinEnd.Before(*pinStart) {
		return nil, status.Error(codes.InvalidArgument, "end_time must be greater than or equal to start_time")
	}

	var pinTags []string
	if req.GetTagsSet() {
		if err := validateTags(req.GetTags()); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		pinTags = req.GetTags()
	} else {
		pinTags = suggested.Tags
	}

	pin := &models.Pin{
		TripID:       tripID,
		Name:         name,
		Description:  description,
		Category:     category,
		PrivacyLevel: trip.PrivacyLevel,
		MediaCount:   int32(len(remaining)),
		Latitude:     pinLat,
		Longitude:    pinLon,
		StartTime:    pinStart,
		EndTime:      pinEnd,
	}
	if err := s.pinRepo.Create(pin); err != nil {
		return nil, status.Error(codes.Internal, "failed to create pin")
	}

	mediaIDs := make([]string, 0, len(remaining))
	for _, m := range remaining {
		mediaIDs = append(mediaIDs, m.ID)
	}
	if err := s.mediaRepo.UpdatePinIDByIDs(mediaIDs, pin.ID); err != nil {
		return nil, status.Error(codes.Internal, "failed to attach media to pin")
	}

	if len(pinTags) > 0 {
		if err := s.tagRepo.SetForPin(tripID, pin.ID, pinTags); err != nil {
			return nil, status.Error(codes.Internal, "failed to set pin tags")
		}
	}

	if req.Latitude == nil && req.StartTimeUnix == nil && req.EndTimeUnix == nil {
		updatePinTimesAndLocation(s.pinRepo, s.mediaRepo, pin.ID)
	}

	if pin.Latitude != nil && pin.Longitude != nil && s.eventRepo != nil {
		_ = s.eventRepo.PublishGeoRequest(ctx, tripID, []repositories.GeoRequestPin{{
			PinID:     pin.ID,
			Latitude:  *pin.Latitude,
			Longitude: *pin.Longitude,
		}})
	}

	if s.eventRepo != nil {
		_ = s.eventRepo.PublishTripEvent(ctx, "PIN_ADDED", tripID, userID)
	}

	if err := s.pinUploadSessionRepo.Close(ctx, sessionID, models.PinUploadSessionCloseReasonConfirmed); err != nil {
		if !errors.Is(err, repositories.ErrPinUploadSessionNotFound) {
			return nil, status.Error(codes.Internal, "failed to close session")
		}
	}

	pin, err := s.pinRepo.GetByID(pin.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to reload pin")
	}
	mediaList, _ := s.mediaRepo.ListByPinID(pin.ID)
	tags, _ := s.tagRepo.GetByPinID(pin.ID)
	if tags == nil {
		tags = []string{}
	}
	return &pb.FinalizePinUploadResponse{Pin: s.pinWithMediaToProto(ctx, pin, mediaList, tags)}, nil
}

// finalizePinUploadAddition: attach media к target_pin, пересчёт агрегатов.
func (s *TripService) finalizePinUploadAddition(
	ctx context.Context,
	session *models.PinUploadSession,
	remaining []*models.Media,
	req *pb.FinalizePinUploadRequest,
	userID string,
) (*pb.FinalizePinUploadResponse, error) {
	_ = userID
	pinID := *session.TargetPinID
	tripID := session.TripID
	sessionID := session.SessionID

	pin, err := s.pinRepo.GetByID(pinID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get pin")
	}
	prevHadCoords := pin.Latitude != nil && pin.Longitude != nil

	if len(remaining) > 0 {
		ids := make([]string, 0, len(remaining))
		for _, m := range remaining {
			ids = append(ids, m.ID)
		}
		if err := s.mediaRepo.UpdatePinIDByIDs(ids, pinID); err != nil {
			return nil, status.Error(codes.Internal, "failed to attach media to pin")
		}
		if err := s.pinRepo.IncMediaCount(pinID, len(ids)); err != nil {
			slog.WarnContext(ctx, "FinalizePinUpload addition: media_count update failed", "pin_id", pinID, "err", err)
		}
		if updated := updatePinTimesAndLocation(s.pinRepo, s.mediaRepo, pinID); updated != nil {
			pin = updated
		}
	}

	if pin.Latitude != nil && pin.Longitude != nil && !prevHadCoords && s.eventRepo != nil {
		_ = s.eventRepo.PublishGeoRequest(ctx, tripID, []repositories.GeoRequestPin{{
			PinID:     pinID,
			Latitude:  *pin.Latitude,
			Longitude: *pin.Longitude,
		}})
	}

	if err := s.pinUploadSessionRepo.Close(ctx, sessionID, models.PinUploadSessionCloseReasonConfirmed); err != nil {
		if !errors.Is(err, repositories.ErrPinUploadSessionNotFound) {
			return nil, status.Error(codes.Internal, "failed to close session")
		}
	}

	mediaList, _ := s.mediaRepo.ListByPinID(pinID)
	tags, _ := s.tagRepo.GetByPinID(pinID)
	if tags == nil {
		tags = []string{}
	}
	return &pb.FinalizePinUploadResponse{Pin: s.pinWithMediaToProto(ctx, pin, mediaList, tags)}, nil
}
