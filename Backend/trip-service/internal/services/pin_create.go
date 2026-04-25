package services

import (
	"context"
	"encoding/json"
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

// pinCreationDraftSnapshot — JSON-структура для сохранения результата
// ProcessPinCreation между этапами Process и Finalize. Хранится в
// pin_creation_sessions.draft_snapshot.
//
// Suggested-поля вычислены ML-stub'ом из media: имя/категория/теги (ТЗ 4.7.2.d-f),
// координаты (4.7.2.c) и start/end (4.7.2.a-b). Клиент показывает их в форме
// редактирования, пользователь правит и шлёт финальные значения в Finalize.
type pinCreationDraftSnapshot struct {
	SuggestedName string `json:"suggested_name"`
	SuggestedCategory string `json:"suggested_category"`
	SuggestedTags []string `json:"suggested_tags"`
	SuggestedLatitude *float64 `json:"suggested_latitude,omitempty"`
	SuggestedLongitude *float64 `json:"suggested_longitude,omitempty"`
	SuggestedStartTimeUnix *int64 `json:"suggested_start_time_unix,omitempty"`
	SuggestedEndTimeUnix *int64 `json:"suggested_end_time_unix,omitempty"`
	NewMediaIDs []string `json:"new_media_ids"`
	NSFWMediaIDs []string `json:"nsfw_media_ids"`
	DedupedMediaIDs []string `json:"deduped_media_ids"`
	PinIssues []string `json:"pin_issues"`
	Similar []pinAdditionSimilarGroup `json:"similar"`
}

// =============================================================================
// CreatePinStart (ТЗ 4.6: пользователь выбирает медиа)
// =============================================================================

func (s *TripService) CreatePinStart(ctx context.Context, req *pb.CreatePinStartRequest) (*pb.CreatePinStartResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	files := req.GetFilesToUpload()
	if len(files) == 0 {
		return nil, status.Error(codes.InvalidArgument, "files_to_upload is required")
	}
	if err := s.assertParticipantAndTripReady(ctx, tripID, userID); err != nil {
		return nil, err
	}
	if err := s.assertTripCapacity(tripID, files); err != nil {
		return nil, err
	}
	if s.pinCreationSessionRepo == nil {
		return nil, status.Error(codes.Internal, "pin creation session repository not configured")
	}
	sessionID, err := s.pinCreationSessionRepo.Create(ctx, tripID, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrPinCreationSessionActive) {
			return nil, status.Error(codes.FailedPrecondition, "another pin creation session is already active for this trip")
		}
		return nil, status.Error(codes.Internal, "failed to create pin creation session")
	}
	uploadUrls, err := s.presignPinUploadUrls(ctx, tripID, files)
	if err != nil {
		return nil, err
	}
	return &pb.CreatePinStartResponse{
		SessionId: sessionID,
		UploadUrls: uploadUrls,
	}, nil
}

// RequestPinCreationUploadUrls — догрузка presigned URLs к активной сессии.
func (s *TripService) RequestPinCreationUploadUrls(ctx context.Context, req *pb.RequestPinCreationUploadUrlsRequest) (*pb.RequestPinCreationUploadUrlsResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID, sessionID := req.GetTripId(), req.GetSessionId()
	if tripID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and session_id are required")
	}
	files := req.GetFilesToUpload()
	if len(files) == 0 {
		return nil, status.Error(codes.InvalidArgument, "files_to_upload is required")
	}
	if err := s.assertParticipantAndTripReady(ctx, tripID, userID); err != nil {
		return nil, err
	}
	if _, err := s.assertActivePinCreationSession(ctx, tripID, sessionID, userID); err != nil {
		return nil, err
	}
	if err := s.assertTripCapacity(tripID, files); err != nil {
		return nil, err
	}
	uploadUrls, err := s.presignPinUploadUrls(ctx, tripID, files)
	if err != nil {
		return nil, err
	}
	_ = s.pinCreationSessionRepo.Touch(ctx, sessionID)
	return &pb.RequestPinCreationUploadUrlsResponse{UploadUrls: uploadUrls}, nil
}

// CommitPinCreationUpload — клиент зовёт после успешного PUT в S3. Создаёт
// media с pin_id=NULL и pin_creation_session_id=session.
func (s *TripService) CommitPinCreationUpload(ctx context.Context, req *pb.CommitPinCreationUploadRequest) (*pb.CommitPinCreationUploadResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID, sessionID := req.GetTripId(), req.GetSessionId()
	if tripID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and session_id are required")
	}
	if req.GetS3Key() == "" || req.GetMediaType() == "" {
		return nil, status.Error(codes.InvalidArgument, "s3_key and media_type are required")
	}
	if req.GetMediaType() != "image" && req.GetMediaType() != "video" {
		return nil, status.Error(codes.InvalidArgument, "media_type must be 'image' or 'video'")
	}
	if err := s.assertParticipantAndTripReady(ctx, tripID, userID); err != nil {
		return nil, err
	}
	if _, err := s.assertActivePinCreationSession(ctx, tripID, sessionID, userID); err != nil {
		return nil, err
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	total, videos, _ := s.mediaRepo.CountByTripID(tripID)
	if total+1 > MaxMediaPerTrip {
		return nil, errLimitExceeded("media", MaxMediaPerTrip, total+1)
	}
	if req.GetMediaType() == "video" && videos+1 > MaxVideosPerTrip {
		return nil, errLimitExceeded("video", MaxVideosPerTrip, videos+1)
	}
	m := &models.Media{
		TripID: tripID,
		S3Key: req.GetS3Key(),
		MediaType: req.GetMediaType(),
		PrivacyLevel: trip.PrivacyLevel,
		PinCreationSessionID: &sessionID,
		UploadedBy: &userID,
	}
	if req.CapturedAtUnix != nil {
		t := time.Unix(req.GetCapturedAtUnix(), 0)
		m.CapturedAt = &t
	}
	if req.Latitude != nil && req.Longitude != nil {
		lat := req.GetLatitude()
		lon := req.GetLongitude()
		m.Latitude = &lat
		m.Longitude = &lon
	}
	if err := s.mediaRepo.Create(m); err != nil {
		return nil, status.Error(codes.Internal, "failed to save media")
	}
	_ = s.pinCreationSessionRepo.Touch(ctx, sessionID)
	mediaInSession, _ := s.mediaRepo.ListByPinCreationSession(sessionID)
	return &pb.CommitPinCreationUploadResponse{
		MediaId: m.ID,
		MediaCountInSession: int32(len(mediaInSession)),
	}, nil
}

// =============================================================================
// ProcessPinCreation (ТЗ 4.7: система обрабатывает медиа)
// =============================================================================

// ProcessPinCreation — синхронный ML-stub: хеш-дедуп (4.7.6.a, дубли удаляются),
// NSFW (4.7.5, заглушка), similar (4.7.7, заглушка), suggested поля для пина
// (4.7.2.a-f), pin issues (4.7.3-4.7.4). Snapshot сохраняется в сессию.
func (s *TripService) ProcessPinCreation(ctx context.Context, req *pb.ProcessPinCreationRequest) (*pb.ProcessPinCreationResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID, sessionID := req.GetTripId(), req.GetSessionId()
	if tripID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and session_id are required")
	}
	if err := s.assertParticipantAndTripReady(ctx, tripID, userID); err != nil {
		return nil, err
	}
	if _, err := s.assertActivePinCreationSession(ctx, tripID, sessionID, userID); err != nil {
		return nil, err
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get trip")
	}

	// 1. Текущие медиа сессии.
	sessionMedia, err := s.mediaRepo.ListByPinCreationSession(sessionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list session media")
	}

	// 2. Хеш-дедупликация (ТЗ 4.7.6.a): идентичные файлы по content_hash в рамках сессии.
	seen := map[string]struct{}{}
	var deduped []*models.Media
	for _, m := range sessionMedia {
		if m.ContentHash == nil {
			continue
		}
		if _, dup := seen[*m.ContentHash]; dup {
			deduped = append(deduped, m)
			continue
		}
		seen[*m.ContentHash] = struct{}{}
	}
	dedupedIDs := make([]string, 0, len(deduped))
	dedupedKeys := make([]string, 0, len(deduped))
	for _, m := range deduped {
		dedupedIDs = append(dedupedIDs, m.ID)
		if m.S3Key != "" {
			dedupedKeys = append(dedupedKeys, m.S3Key)
		}
	}
	if len(dedupedIDs) > 0 {
		if err := s.mediaRepo.DeleteByIDs(dedupedIDs); err != nil {
			return nil, status.Error(codes.Internal, "failed to delete duplicates")
		}
		if s.mediaURLs != nil {
			for _, k := range dedupedKeys {
				_ = s.mediaURLs.DeleteObject(ctx, k)
			}
		}
	}

	// 3. Перечитать оставшиеся.
	remaining, err := s.mediaRepo.ListByPinCreationSession(sessionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to reload session media")
	}

	// 4. NSFW и similar — заглушки до подключения реального ML.
	var nsfwIDs []string
	var similar []pinAdditionSimilarGroup

	// 5. Suggested-поля пина (ТЗ 4.7.2):
	// Категория — заглушка PinCategoryDefault до подключения реального ML
	// (категории пина и трипа — разные множества, нельзя брать trip.Category).
	// Имя = категория (4.7.2.f). Теги — пустые до ML.
	_ = trip
	suggestedCategory := PinCategoryDefault
	suggestedName := suggestedCategory
	suggestedTags := []string{}

	// Координаты (4.7.2.c): первое медиа с координатами после сортировки по времени.
	// Старт/конец (4.7.2.a-b): первое/последнее медиа по captured_at.
	var suggestedLat, suggestedLon *float64
	var suggestedStart, suggestedEnd *time.Time
	for _, m := range remaining {
		if suggestedLat == nil && m.Latitude != nil && m.Longitude != nil {
			suggestedLat = m.Latitude
			suggestedLon = m.Longitude
		}
		if m.CapturedAt != nil {
			if suggestedStart == nil || m.CapturedAt.Before(*suggestedStart) {
				suggestedStart = m.CapturedAt
			}
			if suggestedEnd == nil || m.CapturedAt.After(*suggestedEnd) {
				suggestedEnd = m.CapturedAt
			}
		}
	}

	// 6. Pin issues (ТЗ 4.7.3-4.7.4).
	var issues []string
	if suggestedLat == nil || suggestedLon == nil {
		issues = append(issues, pinIssueMissingCoordinates)
	}
	if suggestedStart == nil {
		issues = append(issues, pinIssueMissingDates)
	}

	// 7. Snapshot.
	newIDs := make([]string, 0, len(remaining))
	for _, m := range remaining {
		newIDs = append(newIDs, m.ID)
	}
	snap := pinCreationDraftSnapshot{
		SuggestedName: suggestedName,
		SuggestedCategory: suggestedCategory,
		SuggestedTags: suggestedTags,
		SuggestedLatitude: suggestedLat,
		SuggestedLongitude: suggestedLon,
		NewMediaIDs: newIDs,
		NSFWMediaIDs: nsfwIDs,
		DedupedMediaIDs: dedupedIDs,
		PinIssues: issues,
		Similar: similar,
	}
	if suggestedStart != nil {
		v := suggestedStart.Unix()
		snap.SuggestedStartTimeUnix = &v
	}
	if suggestedEnd != nil {
		v := suggestedEnd.Unix()
		snap.SuggestedEndTimeUnix = &v
	}
	snapBytes, err := json.Marshal(snap)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to encode snapshot")
	}
	if err := s.pinCreationSessionRepo.SetDraftSnapshot(ctx, sessionID, snapBytes); err != nil {
		return nil, status.Error(codes.Internal, "failed to save snapshot")
	}
	return &pb.ProcessPinCreationResponse{
		SessionId: sessionID,
		Draft: snapshotToCreationDraftProto(ctx, s, snap, remaining),
		Similar: snapshotToCreationSimilarProto(snap),
	}, nil
}

// GetPinCreationReview — повторное чтение snapshot (ТЗ 4.8: пользователь
// получает сформированный пин со всеми полями и медиа).
func (s *TripService) GetPinCreationReview(ctx context.Context, req *pb.GetPinCreationReviewRequest) (*pb.GetPinCreationReviewResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID, sessionID := req.GetTripId(), req.GetSessionId()
	if tripID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and session_id are required")
	}
	if err := s.assertParticipantAndTripReady(ctx, tripID, userID); err != nil {
		return nil, err
	}
	session, err := s.assertActivePinCreationSession(ctx, tripID, sessionID, userID)
	if err != nil {
		return nil, err
	}
	var snap pinCreationDraftSnapshot
	if len(session.DraftSnapshot) > 0 {
		_ = json.Unmarshal(session.DraftSnapshot, &snap)
	}
	remaining, _ := s.mediaRepo.ListByPinCreationSession(sessionID)
	return &pb.GetPinCreationReviewResponse{
		SessionId: sessionID,
		Draft: snapshotToCreationDraftProto(ctx, s, snap, remaining),
		Similar: snapshotToCreationSimilarProto(snap),
	}, nil
}

// =============================================================================
// FinalizePinCreation (ТЗ 4.9-4.11)
// =============================================================================

// FinalizePinCreation: применяет media_to_delete (4.10.1), создаёт запись pins
// с финальными полями (пользовательские правки поверх suggested, 4.9 + 4.10.3),
// привязывает media к новому пину, применяет теги, делает reverse-geocoding,
// публикует PIN_ADDED (ТЗ 11.2.1), закрывает сессию.
func (s *TripService) FinalizePinCreation(ctx context.Context, req *pb.FinalizePinCreationRequest) (*pb.FinalizePinCreationResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID, sessionID := req.GetTripId(), req.GetSessionId()
	if tripID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and session_id are required")
	}
	if err := s.assertParticipantAndTripReady(ctx, tripID, userID); err != nil {
		return nil, err
	}
	session, err := s.assertActivePinCreationSession(ctx, tripID, sessionID, userID)
	if err != nil {
		return nil, err
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get trip")
	}

	// 1. Загрузить snapshot для suggested полей.
	var snap pinCreationDraftSnapshot
	if len(session.DraftSnapshot) > 0 {
		_ = json.Unmarshal(session.DraftSnapshot, &snap)
	}

	// 2. Применить media_to_delete (только media текущей сессии — orphan).
	if del := req.GetMediaToDelete(); len(del) > 0 {
		toDelete, s3Keys, derr := s.collectDeletableCreationSessionMedia(sessionID, del)
		if derr != nil {
			return nil, derr
		}
		if len(toDelete) > 0 {
			if err := s.mediaRepo.DeleteByIDs(toDelete); err != nil {
				return nil, status.Error(codes.Internal, "failed to delete media")
			}
			if s.mediaURLs != nil {
				for _, k := range s3Keys {
					_ = s.mediaURLs.DeleteObject(ctx, k)
				}
			}
		}
	}

	// 3. Получить оставшиеся media — они станут содержимым нового пина.
	remaining, err := s.mediaRepo.ListByPinCreationSession(sessionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to reload session media")
	}
	if len(remaining) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "pin must contain at least one media")
	}

	// 4. Финальные значения полей: req → snapshot.Suggested.
	name := snap.SuggestedName
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

	category := snap.SuggestedCategory
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
	} else if snap.SuggestedLatitude != nil && snap.SuggestedLongitude != nil {
		pinLat = snap.SuggestedLatitude
		pinLon = snap.SuggestedLongitude
	}

	var pinStart, pinEnd *time.Time
	if req.StartTimeUnix != nil {
		t := time.Unix(req.GetStartTimeUnix(), 0)
		pinStart = &t
	} else if snap.SuggestedStartTimeUnix != nil {
		t := time.Unix(*snap.SuggestedStartTimeUnix, 0)
		pinStart = &t
	}
	if req.EndTimeUnix != nil {
		t := time.Unix(req.GetEndTimeUnix(), 0)
		pinEnd = &t
	} else if snap.SuggestedEndTimeUnix != nil {
		t := time.Unix(*snap.SuggestedEndTimeUnix, 0)
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
		pinTags = snap.SuggestedTags
	}

	// 5. Создать запись pins.
	pin := &models.Pin{
		TripID: tripID,
		Name: name,
		Description: description,
		Category: category,
		PrivacyLevel: trip.PrivacyLevel,
		MediaCount: int32(len(remaining)),
		Latitude: pinLat,
		Longitude: pinLon,
		StartTime: pinStart,
		EndTime: pinEnd,
	}
	if err := s.pinRepo.Create(pin); err != nil {
		return nil, status.Error(codes.Internal, "failed to create pin")
	}

	// 6. Привязать media к новому пину.
	mediaIDs := make([]string, 0, len(remaining))
	for _, m := range remaining {
		mediaIDs = append(mediaIDs, m.ID)
	}
	if err := s.mediaRepo.UpdatePinIDByIDs(mediaIDs, pin.ID); err != nil {
		return nil, status.Error(codes.Internal, "failed to attach media to pin")
	}

	// 7. Применить теги.
	if len(pinTags) > 0 {
		if err := s.tagRepo.SetForPin(tripID, pin.ID, pinTags); err != nil {
			return nil, status.Error(codes.Internal, "failed to set pin tags")
		}
	}

	// 8. Согласованность start/end/lat/lon: пересчитываем из media.
	// Если пользователь явно задал значения — Update перезапишет их обратно,
	// поэтому делаем это только при отсутствии ручных полей.
	if req.Latitude == nil && req.StartTimeUnix == nil && req.EndTimeUnix == nil {
		updatePinTimesAndLocation(s.pinRepo, s.mediaRepo, pin.ID)
	}

	// 9. Reverse-geocoding.
	if pin.Latitude != nil && pin.Longitude != nil && s.geocoder != nil {
		country, city, displayName, gerr := s.geocoder.ResolveLocation(ctx, *pin.Latitude, *pin.Longitude)
		if gerr != nil {
			slog.WarnContext(ctx, "FinalizePinCreation: reverse geocoding failed", "pin_id", pin.ID, "err", gerr)
		} else if displayName != "" {
			pin.LocationName = displayName
			_ = s.pinRepo.Update(pin)
			if s.geoRepo != nil {
				countryID, cityID, _, ensureErr := s.geoRepo.EnsureLocationByName(ctx, country, city)
				if ensureErr == nil {
					var locIDs []int
					if countryID != nil {
						locIDs = append(locIDs, *countryID)
					}
					if cityID != nil {
						locIDs = append(locIDs, *cityID)
					}
					_ = s.geoRepo.UpsertTripLocations(ctx, tripID, locIDs)
				}
			}
		}
	}

	// 10. PIN_ADDED (ТЗ 11.2.1).
	if s.eventRepo != nil {
		_ = s.eventRepo.PublishTripEvent(ctx, "PIN_ADDED", tripID, userID)
	}

	// 11. Закрыть сессию.
	if err := s.pinCreationSessionRepo.Close(ctx, sessionID, models.PinCreationSessionCloseReasonConfirmed); err != nil {
		if !errors.Is(err, repositories.ErrPinCreationSessionNotFound) {
			return nil, status.Error(codes.Internal, "failed to close session")
		}
	}

	// 12. Финальный TripPin для ответа.
	pin, err = s.pinRepo.GetByID(pin.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to reload pin")
	}
	mediaList, _ := s.mediaRepo.ListByPinID(pin.ID)
	tags, _ := s.tagRepo.GetByPinID(pin.ID)
	if tags == nil {
		tags = []string{}
	}
	return &pb.FinalizePinCreationResponse{Pin: s.pinWithMediaToProto(ctx, pin, mediaList, tags)}, nil
}

// CancelPinCreation удаляет orphan media сессии и закрывает сессию.
func (s *TripService) CancelPinCreation(ctx context.Context, req *pb.CancelPinCreationRequest) (*pb.CancelPinCreationResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID, sessionID := req.GetTripId(), req.GetSessionId()
	if tripID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and session_id are required")
	}
	participant, perr := s.participantRepo.IsParticipant(tripID, userID)
	if perr != nil || !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	if _, err := s.assertActivePinCreationSession(ctx, tripID, sessionID, userID); err != nil {
		return nil, err
	}
	s3Keys, err := s.mediaRepo.DeleteOrphanByPinCreationSession(sessionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to cleanup orphan media")
	}
	if s.mediaURLs != nil {
		for _, k := range s3Keys {
			_ = s.mediaURLs.DeleteObject(ctx, k)
		}
	}
	if err := s.pinCreationSessionRepo.Close(ctx, sessionID, models.PinCreationSessionCloseReasonCancelled); err != nil {
		if !errors.Is(err, repositories.ErrPinCreationSessionNotFound) {
			return nil, status.Error(codes.Internal, "failed to close session")
		}
	}
	return &pb.CancelPinCreationResponse{Status: "cancelled"}, nil
}

// =============================================================================
// helpers
// =============================================================================

// assertParticipantAndTripReady — упрощённая версия assertParticipantAndPinReady
// без pin-проверки (на этапе CreatePin пина ещё нет).
func (s *TripService) assertParticipantAndTripReady(ctx context.Context, tripID, userID string) error {
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil {
		return status.Error(codes.Internal, "failed to check participant")
	}
	if !participant {
		return status.Error(codes.PermissionDenied, "not a participant")
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		return status.Error(codes.Internal, "failed to get trip")
	}
	if trip.Status != models.TripStatusReady {
		return errWrongStatus(models.TripStatusReady, trip.Status)
	}
	_ = ctx
	return nil
}

// assertActivePinCreationSession — сессия активна, принадлежит трипу, инициатор = caller.
func (s *TripService) assertActivePinCreationSession(ctx context.Context, tripID, sessionID, userID string) (*models.PinCreationSession, error) {
	if s.pinCreationSessionRepo == nil {
		return nil, status.Error(codes.Internal, "pin creation session repository not configured")
	}
	session, err := s.pinCreationSessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, repositories.ErrPinCreationSessionNotFound) {
			return nil, status.Error(codes.NotFound, "pin creation session not found")
		}
		return nil, status.Error(codes.Internal, "failed to load pin creation session")
	}
	if session.ClosedAt != nil {
		return nil, status.Error(codes.FailedPrecondition, "pin creation session is closed")
	}
	if session.TripID != tripID {
		return nil, status.Error(codes.PermissionDenied, "session does not belong to this trip")
	}
	if session.InitiatorUserID != userID {
		return nil, status.Error(codes.PermissionDenied, "only session initiator can act on it")
	}
	return session, nil
}

func (s *TripService) collectDeletableCreationSessionMedia(sessionID string, ids []string) ([]string, []string, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}
	idSet := map[string]struct{}{}
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	sessionMedia, err := s.mediaRepo.ListByPinCreationSession(sessionID)
	if err != nil {
		return nil, nil, status.Error(codes.Internal, "failed to list session media")
	}
	var allowed []string
	var keys []string
	for _, m := range sessionMedia {
		if _, ok := idSet[m.ID]; !ok {
			continue
		}
		allowed = append(allowed, m.ID)
		if m.S3Key != "" {
			keys = append(keys, m.S3Key)
		}
	}
	return allowed, keys, nil
}

func snapshotToCreationDraftProto(ctx context.Context, s *TripService, snap pinCreationDraftSnapshot, media []*models.Media) *pb.PinCreationDraft {
	out := &pb.PinCreationDraft{
		SuggestedName: snap.SuggestedName,
		SuggestedCategory: snap.SuggestedCategory,
		SuggestedTags: snap.SuggestedTags,
		PinIssues: snap.PinIssues,
		NsfwMediaIds: snap.NSFWMediaIDs,
		DedupedMediaIds: snap.DedupedMediaIDs,
	}
	if snap.SuggestedLatitude != nil {
		out.SuggestedLatitude = snap.SuggestedLatitude
	}
	if snap.SuggestedLongitude != nil {
		out.SuggestedLongitude = snap.SuggestedLongitude
	}
	if snap.SuggestedStartTimeUnix != nil {
		out.SuggestedStartTimeUnix = snap.SuggestedStartTimeUnix
	}
	if snap.SuggestedEndTimeUnix != nil {
		out.SuggestedEndTimeUnix = snap.SuggestedEndTimeUnix
	}
	for _, m := range media {
		out.Media = append(out.Media, &pb.ReviewPinMedia{
			MediaId: m.ID,
			Url: s.presignedReadURL(ctx, m.S3Key),
			PrivacyLevel: m.PrivacyLevel,
		})
	}
	return out
}

func snapshotToCreationSimilarProto(snap pinCreationDraftSnapshot) []*pb.MediaSimilarGroup {
	out := make([]*pb.MediaSimilarGroup, 0, len(snap.Similar))
	for _, g := range snap.Similar {
		out = append(out, &pb.MediaSimilarGroup{MediaIds: g.MediaIDs})
	}
	return out
}
